import 'dart:async';
import 'dart:io';
import 'dart:typed_data';
import 'package:flutter/foundation.dart';
import 'package:path_provider/path_provider.dart';
import 'package:path/path.dart' as p;
import 'package:record/record.dart';
import 'package:sherpa_onnx/sherpa_onnx.dart' as sherpa_onnx;
import 'package:http/http.dart' as http;
import 'package:archive/archive.dart';

/// Service for offline speech-to-text using sherpa-onnx
class VoiceService extends ChangeNotifier {
  // Model files for English streaming ASR
  static const String _modelName = 'sherpa-onnx-streaming-zipformer-en-2023-06-26';
  static const String _modelUrl =
      'https://github.com/k2-fsa/sherpa-onnx/releases/download/asr-models/$_modelName.tar.bz2';

  AudioRecorder? _audioRecorder;
  sherpa_onnx.OnlineRecognizer? _recognizer;
  sherpa_onnx.OnlineStream? _stream;
  StreamSubscription? _audioSubscription;

  bool _isInitialized = false;
  bool _isDownloading = false;
  bool _isListening = false;
  String _currentText = '';
  String? _error;
  double _downloadProgress = 0.0;

  bool get isAvailable => _isInitialized;
  bool get isDownloading => _isDownloading;
  bool get isListening => _isListening;
  String get currentText => _currentText;
  String? get error => _error;
  double get downloadProgress => _downloadProgress;

  /// Initialize sherpa-onnx with model download if needed
  Future<bool> initialize() async {
    if (_isInitialized) return true;

    try {
      // Initialize bindings
      sherpa_onnx.initBindings();

      // Check if model exists, download if not
      final modelDir = await _getModelDir();
      final encoderPath = p.join(modelDir, 'encoder-epoch-99-avg-1-chunk-16-left-128.int8.onnx');

      if (!await File(encoderPath).exists()) {
        await _downloadModel();
      }

      // Create recognizer config
      final modelConfig = sherpa_onnx.OnlineModelConfig(
        transducer: sherpa_onnx.OnlineTransducerModelConfig(
          encoder: p.join(modelDir, 'encoder-epoch-99-avg-1-chunk-16-left-128.int8.onnx'),
          decoder: p.join(modelDir, 'decoder-epoch-99-avg-1-chunk-16-left-128.onnx'),
          joiner: p.join(modelDir, 'joiner-epoch-99-avg-1-chunk-16-left-128.onnx'),
        ),
        tokens: p.join(modelDir, 'tokens.txt'),
        modelType: 'zipformer2',
      );

      final config = sherpa_onnx.OnlineRecognizerConfig(
        model: modelConfig,
        ruleFsts: '',
      );

      _recognizer = sherpa_onnx.OnlineRecognizer(config);
      _audioRecorder = AudioRecorder();

      _isInitialized = true;
      _error = null;
      notifyListeners();
      return true;
    } catch (e) {
      _error = 'Failed to initialize: $e';
      _isInitialized = false;
      notifyListeners();
      debugPrint('Voice init error: $e');
      return false;
    }
  }

  Future<String> _getModelDir() async {
    final appDir = await getApplicationSupportDirectory();
    return p.join(appDir.path, _modelName);
  }

  Future<void> _downloadModel() async {
    _isDownloading = true;
    _downloadProgress = 0.0;
    _error = 'Downloading speech model (~70MB)...';
    notifyListeners();

    try {
      final modelDir = await _getModelDir();
      await Directory(modelDir).create(recursive: true);

      // Download tar.bz2 file
      debugPrint('Downloading model from $_modelUrl');
      final response = await http.Client().send(http.Request('GET', Uri.parse(_modelUrl)));

      final contentLength = response.contentLength ?? 0;
      final bytes = <int>[];
      int received = 0;

      await for (final chunk in response.stream) {
        bytes.addAll(chunk);
        received += chunk.length;
        if (contentLength > 0) {
          _downloadProgress = received / contentLength;
          _error = 'Downloading: ${(_downloadProgress * 100).toStringAsFixed(1)}%';
          notifyListeners();
        }
      }

      _error = 'Extracting model...';
      notifyListeners();

      // Extract tar.bz2
      final decompressed = BZip2Decoder().decodeBytes(bytes);
      final archive = TarDecoder().decodeBytes(decompressed);

      for (final file in archive) {
        if (file.isFile) {
          // Extract to model dir, flattening the archive structure
          final filename = p.basename(file.name);
          final outFile = File(p.join(modelDir, filename));
          await outFile.writeAsBytes(file.content as List<int>);
          debugPrint('Extracted: $filename');
        }
      }

      _downloadProgress = 1.0;
      debugPrint('Model downloaded and extracted to $modelDir');
    } catch (e) {
      _error = 'Download failed: $e';
      rethrow;
    } finally {
      _isDownloading = false;
      notifyListeners();
    }
  }

  /// Start listening for speech
  Future<void> startListening({
    required Function(String) onResult,
  }) async {
    if (!_isInitialized || _recognizer == null || _audioRecorder == null) {
      _error = 'Not initialized';
      notifyListeners();
      return;
    }

    if (_isListening) return;

    _currentText = '';
    _error = null;
    _isListening = true;
    notifyListeners();

    try {
      // Check permission
      if (!await _audioRecorder!.hasPermission()) {
        _error = 'Microphone permission denied';
        _isListening = false;
        notifyListeners();
        return;
      }

      // Create new stream for this session
      _stream = _recognizer!.createStream();

      // Configure audio recording
      const config = RecordConfig(
        encoder: AudioEncoder.pcm16bits,
        sampleRate: 16000,
        numChannels: 1,
      );

      // Start recording stream
      final audioStream = await _audioRecorder!.startStream(config);

      _audioSubscription = audioStream.listen(
        (data) {
          if (!_isListening || _stream == null) return;

          // Convert bytes to float32 samples
          final samples = _convertBytesToFloat32(Uint8List.fromList(data));

          // Feed to recognizer
          _stream!.acceptWaveform(samples: samples, sampleRate: 16000);

          // Decode while ready
          while (_recognizer!.isReady(_stream!)) {
            _recognizer!.decode(_stream!);
          }

          // Get current result
          final result = _recognizer!.getResult(_stream!);
          if (result.text.isNotEmpty) {
            _currentText = result.text;
            notifyListeners();
          }

          // Check for endpoint (pause in speech)
          if (_recognizer!.isEndpoint(_stream!)) {
            if (_currentText.isNotEmpty) {
              onResult(_currentText);
              _currentText = '';
            }
            _recognizer!.reset(_stream!);
          }
        },
        onError: (e) {
          _error = 'Audio error: $e';
          _isListening = false;
          notifyListeners();
        },
      );
    } catch (e) {
      _error = 'Failed to start: $e';
      _isListening = false;
      notifyListeners();
    }
  }

  Float32List _convertBytesToFloat32(Uint8List bytes) {
    final values = Float32List(bytes.length ~/ 2);
    final data = ByteData.view(bytes.buffer);

    for (var i = 0; i < bytes.length; i += 2) {
      int short = data.getInt16(i, Endian.little);
      values[i ~/ 2] = short / 32768.0;
    }

    return values;
  }

  /// Stop listening and return final result
  Future<void> stopListening() async {
    if (!_isListening) return;

    await _audioSubscription?.cancel();
    _audioSubscription = null;
    await _audioRecorder?.stop();

    // Get final result before stopping
    if (_stream != null && _currentText.isNotEmpty) {
      // Result already sent via callback
    }

    _stream = null;
    _isListening = false;
    notifyListeners();
  }

  /// Cancel listening without returning result
  Future<void> cancelListening() async {
    if (!_isListening) return;

    await _audioSubscription?.cancel();
    _audioSubscription = null;
    await _audioRecorder?.stop();

    _stream = null;
    _isListening = false;
    _currentText = '';
    notifyListeners();
  }

  @override
  void dispose() {
    _audioSubscription?.cancel();
    _audioRecorder?.dispose();
    _stream = null;
    _recognizer = null;
    super.dispose();
  }
}
