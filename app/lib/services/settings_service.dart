import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';

class SettingsService extends ChangeNotifier {
  SharedPreferences? _prefs;
  late final Future<void> _initialized;
  HttpClient? _httpClient;

  String? _selectedProvider;
  String? _selectedModel;
  List<ProviderInfo> _availableProviders = [];
  List<ModelInfo> _availableModels = [];
  bool _isLoading = false;
  String? _error;

  // Getters
  String? get selectedProvider => _selectedProvider;
  String? get selectedModel => _selectedModel;
  List<ProviderInfo> get availableProviders => _availableProviders;
  List<ModelInfo> get availableModels => _availableModels;
  bool get isLoading => _isLoading;
  String? get error => _error;

  SettingsService() {
    _httpClient = HttpClient(); // Proper HTTPS validation
    _initialized = _init();
  }

  Future<void> _init() async {
    _prefs = await SharedPreferences.getInstance();
    _selectedProvider = _prefs?.getString('llm_provider');
    _selectedModel = _prefs?.getString('llm_model');
    notifyListeners();
  }

  /// Ensure initialization is complete before accessing prefs
  Future<void> _ensureInitialized() => _initialized;

  Future<void> fetchProviders(String baseUrl) async {
    await _ensureInitialized();
    _isLoading = true;
    _error = null;
    notifyListeners();

    try {
      final uri = Uri.parse('$baseUrl/api/v1/providers');
      final request = await _httpClient!.getUrl(uri);
      final response = await request.close().timeout(
        const Duration(seconds: 10),
        onTimeout: () => throw TimeoutException('Request timed out'),
      );

      if (response.statusCode == 200) {
        final body = await response.transform(utf8.decoder).join();
        final json = jsonDecode(body) as Map<String, dynamic>;

        _availableProviders = (json['providers'] as List)
            .map((p) => ProviderInfo.fromJson(p as Map<String, dynamic>))
            .toList();

        if (_selectedProvider == null && json['current'] != null) {
          final current = json['current'] as Map<String, dynamic>;
          _selectedProvider = current['provider'] as String?;
          _selectedModel = current['model'] as String?;
        }
      } else {
        _error = 'Failed to fetch providers: HTTP ${response.statusCode}';
      }
    } catch (e) {
      _error = 'Failed to fetch providers: $e';
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  Future<void> fetchModels(String baseUrl, String provider) async {
    await _ensureInitialized();
    _isLoading = true;
    _error = null;
    notifyListeners();

    try {
      final uri = Uri.parse('$baseUrl/api/v1/providers/$provider/models');
      final request = await _httpClient!.getUrl(uri);
      final response = await request.close().timeout(
        const Duration(seconds: 10),
        onTimeout: () => throw TimeoutException('Request timed out'),
      );

      if (response.statusCode == 200) {
        final body = await response.transform(utf8.decoder).join();
        final json = jsonDecode(body) as Map<String, dynamic>;

        _availableModels = (json['models'] as List)
            .map((m) => ModelInfo.fromJson(m as Map<String, dynamic>))
            .toList();
      } else {
        _error = 'Failed to fetch models: HTTP ${response.statusCode}';
      }
    } catch (e) {
      _error = 'Failed to fetch models: $e';
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  Future<void> selectProvider(String provider) async {
    await _ensureInitialized();
    _selectedProvider = provider;
    _selectedModel = null;
    _availableModels = [];
    await _prefs?.setString('llm_provider', provider);
    await _prefs?.remove('llm_model'); // Clear model when provider changes
    notifyListeners();
  }

  Future<void> selectModel(String model) async {
    await _ensureInitialized();
    _selectedModel = model;
    await _prefs?.setString('llm_model', model);
    notifyListeners();
  }

  @override
  void dispose() {
    _httpClient?.close();
    _httpClient = null;
    super.dispose();
  }
}

class ProviderInfo {
  final String name;
  final String displayName;
  final bool requiresAuth;
  final String defaultUrl;

  ProviderInfo({
    required this.name,
    required this.displayName,
    required this.requiresAuth,
    required this.defaultUrl,
  });

  factory ProviderInfo.fromJson(Map<String, dynamic> json) {
    return ProviderInfo(
      name: json['name'] as String,
      displayName: json['display_name'] as String,
      requiresAuth: json['requires_auth'] as bool,
      defaultUrl: json['default_url'] as String,
    );
  }
}

class ModelInfo {
  final String id;
  final String name;
  final int? contextSize;

  ModelInfo({required this.id, required this.name, this.contextSize});

  factory ModelInfo.fromJson(Map<String, dynamic> json) {
    return ModelInfo(
      id: json['id'] as String,
      name: (json['name'] ?? json['id']) as String,
      contextSize: json['context_size'] as int?,
    );
  }
}
