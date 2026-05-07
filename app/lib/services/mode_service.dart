import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'dart:convert';
import 'dart:io';
import 'dart:async';

class ModeService extends ChangeNotifier {
  SharedPreferences? _prefs;
  late final Future<void> _initialized;
  HttpClient? _httpClient;

  String? _selectedModeId;
  List<AppMode> _availableModes = [];
  bool _isLoading = false;
  String? _error;

  // Getters
  String? get selectedModeId => _selectedModeId;
  AppMode? get selectedMode {
    if (_selectedModeId == null || _availableModes.isEmpty) {
      return _availableModes.isNotEmpty ? _availableModes.first : null;
    }
    try {
      return _availableModes.firstWhere(
        (m) => m.id == _selectedModeId,
        orElse: () => _availableModes.first,
      );
    } catch (e) {
      return _availableModes.first;
    }
  }

  List<AppMode> get availableModes => _availableModes;
  bool get isLoading => _isLoading;
  String? get error => _error;

  ModeService() {
    _httpClient = HttpClient();
    _initialized = _init();
  }

  Future<void> _init() async {
    _prefs = await SharedPreferences.getInstance();
    _selectedModeId = _prefs?.getString('selected_mode');
    notifyListeners();
  }

  /// Ensure initialization is complete before accessing prefs
  Future<void> _ensureInitialized() => _initialized;

  Future<void> fetchModes(String baseUrl, {bool forceRefresh = false}) async {
    await _ensureInitialized();

    if (_availableModes.isNotEmpty && !forceRefresh) return;

    // Hardcoded modes — no server fetch needed
    _availableModes = [
      AppMode(
          id: 'sonos',
          name: 'Music',
          icon: 'speaker',
          description: 'Control speakers and play music',
          defaultMode: true),
      AppMode(
          id: 'grocery',
          name: 'Grocery',
          icon: 'shopping_cart',
          description: 'Build optimized grocery carts'),
      AppMode(
          id: 'websearch',
          name: 'Search',
          icon: 'search',
          description: 'Search the web'),
    ];

    if (_selectedModeId == null) {
      _selectedModeId = _availableModes
          .firstWhere((m) => m.defaultMode, orElse: () => _availableModes.first)
          .id;
      await _prefs?.setString('selected_mode', _selectedModeId!);
    }

    _isLoading = false;
    notifyListeners();
  }

  Future<void> selectMode(String modeId) async {
    await _ensureInitialized();
    _selectedModeId = modeId;
    await _prefs?.setString('selected_mode', modeId);
    notifyListeners();
  }

  /// Set location on backend (for grocery mode)
  Future<bool> setLocationOnBackend(
    String baseUrl,
    double latitude,
    double longitude,
    String locationName,
  ) async {
    try {
      final uri = Uri.parse('$baseUrl/api/v1/chat');
      final request = await _httpClient!.postUrl(uri);
      request.headers.contentType = ContentType.json;

      // Send location_set tool request as a chat message
      final requestBody = {
        'message': 'Setting delivery location to $locationName',
        'mode': 'grocery',
        'conversation_id': 'location_setup',
      };

      request.write(jsonEncode(requestBody));

      final response = await request.close().timeout(
            const Duration(seconds: 10),
            onTimeout: () => throw TimeoutException('Request timed out'),
          );
      final statusCode = response.statusCode;
      // Drain the response body to free resources
      await response.drain<void>();

      if (statusCode == 200) {
        debugPrint(
            'Location set on backend: $locationName ($latitude, $longitude)');
        return true;
      } else {
        debugPrint('Failed to set location on backend: $statusCode');
        return false;
      }
    } catch (e) {
      debugPrint('Error setting location on backend: $e');
      return false;
    }
  }

  @override
  void dispose() {
    _httpClient?.close();
    _httpClient = null;
    super.dispose();
  }
}

class AppMode {
  final String id;
  final String name;
  final String icon;
  final String description;
  final bool defaultMode;

  AppMode({
    required this.id,
    required this.name,
    required this.icon,
    required this.description,
    this.defaultMode = false,
  });

  factory AppMode.fromJson(Map<String, dynamic> json) {
    return AppMode(
      id: (json['ID'] ?? json['id']) as String,
      name: (json['Name'] ?? json['name']) as String,
      icon: (json['Icon'] ?? json['icon'] as String?) ?? 'help',
      description:
          (json['Description'] ?? json['description'] as String?) ?? '',
      defaultMode:
          (json['DefaultMode'] ?? json['default_mode']) as bool? ?? false,
    );
  }
}
