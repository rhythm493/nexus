import 'package:flutter/foundation.dart';
import 'package:bonsoir/bonsoir.dart';

import '../models/message.dart';

/// Service for discovering pocket-assistant servers via mDNS
class DiscoveryService extends ChangeNotifier {
  BonsoirDiscovery? _discovery;
  final List<DiscoveredServer> _servers = [];
  DiscoveredServer? _selectedServer;
  bool _isSearching = false;
  String? _error;

  List<DiscoveredServer> get servers => List.unmodifiable(_servers);
  DiscoveredServer? get selectedServer => _selectedServer;
  bool get isSearching => _isSearching;
  String? get error => _error;
  bool get hasServer => _selectedServer != null;

  /// Start searching for servers
  Future<void> startDiscovery() async {
    if (_isSearching) return;

    _isSearching = true;
    _error = null;
    _servers.clear();
    notifyListeners();

    try {
      _discovery = BonsoirDiscovery(type: '_pocket-assistant._tcp');
      await _discovery!.ready;

      _discovery!.eventStream!.listen((event) {
        if (event.type == BonsoirDiscoveryEventType.discoveryServiceResolved) {
          final service = event.service as ResolvedBonsoirService;
          final server = DiscoveredServer(
            name: service.name,
            host: service.host ?? service.name,
            port: service.port,
          );

          // Avoid duplicates
          if (!_servers.any((s) => s.host == server.host && s.port == server.port)) {
            _servers.add(server);

            // Auto-select first server
            if (_selectedServer == null) {
              _selectedServer = server;
            }
            notifyListeners();
          }
        }
      });

      await _discovery!.start();
    } catch (e) {
      _error = 'Discovery failed: $e';
      _isSearching = false;
      notifyListeners();
    }
  }

  /// Stop searching
  Future<void> stopDiscovery() async {
    if (_discovery != null) {
      await _discovery!.stop();
      _discovery = null;
    }
    _isSearching = false;
    notifyListeners();
  }

  /// Select a server to connect to
  void selectServer(DiscoveredServer server) {
    _selectedServer = server;
    notifyListeners();
  }

  /// Manually set server address
  void setManualServer(String host, int port) {
    _selectedServer = DiscoveredServer(
      name: 'Manual',
      host: host,
      port: port,
    );
    notifyListeners();
  }

  /// Clear server selection
  void clearServer() {
    _selectedServer = null;
    notifyListeners();
  }

  @override
  void dispose() {
    stopDiscovery();
    super.dispose();
  }
}
