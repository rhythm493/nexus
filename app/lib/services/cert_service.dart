import 'dart:io';
import 'package:flutter/foundation.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:file_picker/file_picker.dart';

/// Service for managing mTLS certificates
class CertService extends ChangeNotifier {
  final FlutterSecureStorage _storage = const FlutterSecureStorage();

  String? _caCert;
  String? _clientCert;
  String? _clientKey;
  bool _isLoaded = false;
  String? _error;

  bool get hasCertificates => _caCert != null && _clientCert != null && _clientKey != null;
  bool get isLoaded => _isLoaded;
  String? get error => _error;
  String? get caCert => _caCert;
  String? get clientCert => _clientCert;
  String? get clientKey => _clientKey;

  static const _caCertKey = 'pocket_assistant_ca_cert';
  static const _clientCertKey = 'pocket_assistant_client_cert';
  static const _clientKeyKey = 'pocket_assistant_client_key';

  /// Load certificates from secure storage
  Future<void> loadCertificates() async {
    try {
      _caCert = await _storage.read(key: _caCertKey);
      _clientCert = await _storage.read(key: _clientCertKey);
      _clientKey = await _storage.read(key: _clientKeyKey);
      _isLoaded = true;
      _error = null;
    } catch (e) {
      _error = 'Failed to load certificates: $e';
    }
    notifyListeners();
  }

  /// Import CA certificate from file
  Future<bool> importCACert() async {
    try {
      final result = await FilePicker.platform.pickFiles(
        type: FileType.any,
        allowMultiple: false,
      );

      if (result == null || result.files.isEmpty) return false;

      final file = File(result.files.first.path!);
      final content = await file.readAsString();

      if (!content.contains('BEGIN CERTIFICATE')) {
        _error = 'Invalid certificate file';
        notifyListeners();
        return false;
      }

      _caCert = content;
      await _storage.write(key: _caCertKey, value: content);
      _error = null;
      notifyListeners();
      return true;
    } catch (e) {
      _error = 'Failed to import CA certificate: $e';
      notifyListeners();
      return false;
    }
  }

  /// Import client certificate from file
  Future<bool> importClientCert() async {
    try {
      final result = await FilePicker.platform.pickFiles(
        type: FileType.any,
        allowMultiple: false,
      );

      if (result == null || result.files.isEmpty) return false;

      final file = File(result.files.first.path!);
      final content = await file.readAsString();

      if (!content.contains('BEGIN CERTIFICATE')) {
        _error = 'Invalid certificate file';
        notifyListeners();
        return false;
      }

      _clientCert = content;
      await _storage.write(key: _clientCertKey, value: content);
      _error = null;
      notifyListeners();
      return true;
    } catch (e) {
      _error = 'Failed to import client certificate: $e';
      notifyListeners();
      return false;
    }
  }

  /// Import client key from file
  Future<bool> importClientKey() async {
    try {
      final result = await FilePicker.platform.pickFiles(
        type: FileType.any,
        allowMultiple: false,
      );

      if (result == null || result.files.isEmpty) return false;

      final file = File(result.files.first.path!);
      final content = await file.readAsString();

      if (!content.contains('BEGIN') || !content.contains('KEY')) {
        _error = 'Invalid key file';
        notifyListeners();
        return false;
      }

      _clientKey = content;
      await _storage.write(key: _clientKeyKey, value: content);
      _error = null;
      notifyListeners();
      return true;
    } catch (e) {
      _error = 'Failed to import client key: $e';
      notifyListeners();
      return false;
    }
  }

  /// Clear all stored certificates
  Future<void> clearCertificates() async {
    await _storage.delete(key: _caCertKey);
    await _storage.delete(key: _clientCertKey);
    await _storage.delete(key: _clientKeyKey);
    _caCert = null;
    _clientCert = null;
    _clientKey = null;
    _error = null;
    notifyListeners();
  }

  /// Create a SecurityContext with loaded certificates
  SecurityContext? createSecurityContext() {
    if (!hasCertificates) return null;

    try {
      final context = SecurityContext();
      context.setTrustedCertificatesBytes(_caCert!.codeUnits);
      context.useCertificateChainBytes(_clientCert!.codeUnits);
      context.usePrivateKeyBytes(_clientKey!.codeUnits);
      return context;
    } catch (e) {
      _error = 'Failed to create security context: $e';
      notifyListeners();
      return null;
    }
  }
}
