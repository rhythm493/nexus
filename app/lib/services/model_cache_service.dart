import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:flutter_cache_manager/flutter_cache_manager.dart';

import 'settings_service.dart' show ModelInfo;

class ModelCacheService extends ChangeNotifier {
  bool _isLoaded = false;
  bool _isLoading = false;
  String? _error;

  List<ModelInfo> _models = [];
  Map<String, Map<String, String>> _mappings = {};
  String _updatedAt = '';

  bool get isLoaded => _isLoaded;
  bool get isLoading => _isLoading;
  String? get error => _error;
  bool get needsLoad => !_isLoaded && !_isLoading;
  String get updatedAt => _updatedAt;

  Future<void> load(String baseUrl) async {
    if (_isLoading) return;
    _isLoading = true;
    _error = null;
    notifyListeners();

    try {
      final url = '$baseUrl/api/v1/models';
      // flutter_cache_manager handles ETag + 304 automatically:
      //   - cached + fresh: returns file instantly
      //   - stale: sends If-None-Match, 304 → cached file, 200 → new file
      final file = await DefaultCacheManager().getSingleFile(url);
      final json = jsonDecode(await file.readAsString()) as Map<String, dynamic>;

      _models = (json['models'] as List)
          .map((m) => ModelInfo.fromJson(m as Map<String, dynamic>))
          .toList();

      if (json['mappings'] is Map) {
        _mappings = (json['mappings'] as Map).map(
          (k, v) => MapEntry(
            k as String,
            (v as Map).map((mk, mv) => MapEntry(mk as String, mv as String)),
          ),
        );
      }

      _updatedAt = json['updated_at'] as String? ?? '';
      _isLoaded = true;
    } catch (e) {
      debugPrint('ModelCacheService: failed to load: $e');
      _error = e.toString();
      // Keep _isLoaded false so we retry on next load
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  ModelInfo? lookup(String provider, String modelId) {
    final orId = _resolveOpenRouterId(provider, modelId);
    if (orId != null) {
      try {
        return _models.firstWhere((m) => m.id == orId);
      } catch (_) {}
    }
    // Fallback: fuzzy search by keyword
    return _fuzzySearch(provider, modelId);
  }

  List<ModelInfo>? getModelsForProvider(String provider) {
    // Filter models that are relevant to this provider by checking mappings
    final providerMapping = _mappings[provider];
    if (providerMapping == null) return null;

    final orIds = providerMapping.values.toSet();
    final result = _models.where((m) => orIds.contains(m.id)).toList();
    return result.isNotEmpty ? result : null;
  }

  String? _resolveOpenRouterId(String provider, String modelId) {
    return _mappings[provider]?[modelId];
  }

  ModelInfo? _fuzzySearch(String provider, String modelId) {
    // Build keywords from the model ID similar to server-side enricher
    String keyword;
    switch (provider) {
      case 'gemini':
        keyword = 'google/${modelId.replaceFirst('models/', '')}';
        break;
      case 'groq':
        final parts = modelId.split('/');
        var base = parts.last;
        base = base.replaceAll(RegExp(r'-(versatile|instruct|instant)$'), '');
        keyword = base;
        break;
      case 'cerebras':
        keyword = 'cerebras/$modelId';
        break;
      default:
        return null;
    }

    try {
      return _models.firstWhere(
        (m) => m.id.toLowerCase().contains(keyword.toLowerCase()),
      );
    } catch (_) {
      return null;
    }
  }

  void invalidate() {
    DefaultCacheManager().emptyCache();
    _isLoaded = false;
    notifyListeners();
  }
}
