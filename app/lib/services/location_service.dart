import 'package:flutter/foundation.dart';
import 'package:geolocator/geolocator.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'dart:convert';

class LocationService extends ChangeNotifier {
  SharedPreferences? _prefs;
  late final Future<void> _initialized;

  /// Cache expires after this duration
  static const _cacheDuration = Duration(hours: 1);

  Position? _currentPosition;
  String? _locationName;
  bool _isLoading = false;
  String? _error;
  bool _permissionGranted = false;

  // Getters
  Position? get currentPosition => _currentPosition;
  String? get locationName => _locationName;
  bool get isLoading => _isLoading;
  String? get error => _error;
  bool get hasLocation => _currentPosition != null;
  bool get permissionGranted => _permissionGranted;

  LocationService() {
    _initialized = _init();
  }

  Future<void> _init() async {
    _prefs = await SharedPreferences.getInstance();

    // Load cached location
    final cachedLocation = _prefs?.getString('cached_location');
    if (cachedLocation != null) {
      try {
        final json = jsonDecode(cachedLocation) as Map<String, dynamic>;
        _currentPosition = Position(
          latitude: json['latitude'] as double,
          longitude: json['longitude'] as double,
          timestamp: DateTime.parse(json['timestamp'] as String),
          accuracy: json['accuracy'] as double,
          altitude: json['altitude'] as double,
          altitudeAccuracy: json['altitudeAccuracy'] as double,
          heading: json['heading'] as double,
          headingAccuracy: json['headingAccuracy'] as double,
          speed: json['speed'] as double,
          speedAccuracy: json['speedAccuracy'] as double,
        );
        _locationName = json['name'] as String?;
      } catch (e) {
        debugPrint('Failed to load cached location: $e');
      }
    }

    // Check permission status
    _permissionGranted = await _checkPermission();
    notifyListeners();
  }

  /// Ensure initialization is complete before accessing prefs
  Future<void> _ensureInitialized() => _initialized;

  /// Check if the cached location has expired
  bool get _isCacheExpired {
    if (_currentPosition == null) return true;
    return DateTime.now().difference(_currentPosition!.timestamp) > _cacheDuration;
  }

  Future<bool> _checkPermission() async {
    final permission = await Geolocator.checkPermission();
    return permission == LocationPermission.always ||
        permission == LocationPermission.whileInUse;
  }

  /// Request location permissions
  Future<bool> requestPermission() async {
    _error = null;
    notifyListeners();

    // Check if location services are enabled
    final serviceEnabled = await Geolocator.isLocationServiceEnabled();
    if (!serviceEnabled) {
      _error = 'Location services are disabled. Please enable them in settings.';
      notifyListeners();
      return false;
    }

    // Check current permission
    LocationPermission permission = await Geolocator.checkPermission();

    if (permission == LocationPermission.denied) {
      permission = await Geolocator.requestPermission();
      if (permission == LocationPermission.denied) {
        _error = 'Location permission denied';
        notifyListeners();
        return false;
      }
    }

    if (permission == LocationPermission.deniedForever) {
      _error = 'Location permissions are permanently denied. Please enable them in app settings.';
      notifyListeners();
      return false;
    }

    _permissionGranted = true;
    notifyListeners();
    return true;
  }

  /// Get current location
  Future<Position?> getCurrentLocation({bool forceRefresh = false}) async {
    await _ensureInitialized();

    // Return cached if available, not forcing refresh, and not expired
    if (!forceRefresh && _currentPosition != null && !_isCacheExpired) {
      return _currentPosition;
    }

    _isLoading = true;
    _error = null;
    notifyListeners();

    try {
      // Request permission if not granted
      if (!_permissionGranted) {
        final granted = await requestPermission();
        if (!granted) {
          _isLoading = false;
          notifyListeners();
          return null;
        }
      }

      // Get current position
      _currentPosition = await Geolocator.getCurrentPosition(
        locationSettings: const LocationSettings(
          accuracy: LocationAccuracy.high,
          timeLimit: Duration(seconds: 10),
        ),
      );

      // Try to get location name (reverse geocoding would go here)
      // For now, just use coordinates as name
      _locationName = '${_currentPosition!.latitude.toStringAsFixed(4)}, ${_currentPosition!.longitude.toStringAsFixed(4)}';

      // Cache the location
      await _cacheLocation();

      _error = null;
    } catch (e) {
      _error = 'Failed to get location: $e';
      debugPrint(_error);
    } finally {
      _isLoading = false;
      notifyListeners();
    }

    return _currentPosition;
  }

  /// Set location manually
  Future<void> setLocation(double latitude, double longitude, String name) async {
    await _ensureInitialized();
    _currentPosition = Position(
      latitude: latitude,
      longitude: longitude,
      timestamp: DateTime.now(),
      accuracy: 0,
      altitude: 0,
      altitudeAccuracy: 0,
      heading: 0,
      headingAccuracy: 0,
      speed: 0,
      speedAccuracy: 0,
    );
    _locationName = name;

    await _cacheLocation();
    notifyListeners();
  }

  /// Cache location to SharedPreferences
  Future<void> _cacheLocation() async {
    if (_currentPosition == null) return;

    final json = {
      'latitude': _currentPosition!.latitude,
      'longitude': _currentPosition!.longitude,
      'timestamp': _currentPosition!.timestamp.toIso8601String(),
      'accuracy': _currentPosition!.accuracy,
      'altitude': _currentPosition!.altitude,
      'altitudeAccuracy': _currentPosition!.altitudeAccuracy,
      'heading': _currentPosition!.heading,
      'headingAccuracy': _currentPosition!.headingAccuracy,
      'speed': _currentPosition!.speed,
      'speedAccuracy': _currentPosition!.speedAccuracy,
      'name': _locationName,
    };

    await _prefs?.setString('cached_location', jsonEncode(json));
  }

  /// Clear cached location
  Future<void> clearLocation() async {
    await _ensureInitialized();
    _currentPosition = null;
    _locationName = null;
    await _prefs?.remove('cached_location');
    notifyListeners();
  }

  /// Get location summary for display
  String getLocationSummary() {
    if (_currentPosition == null) {
      return 'No location set';
    }
    return _locationName ??
           '${_currentPosition!.latitude.toStringAsFixed(4)}, ${_currentPosition!.longitude.toStringAsFixed(4)}';
  }
}
