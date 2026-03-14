import 'dart:io';
import 'dart:async';
import 'package:flutter/foundation.dart';
import 'package:bonsoir/bonsoir.dart';

import '../models/message.dart';

/// Service for discovering Nexus servers via mDNS
class DiscoveryService extends ChangeNotifier {
  BonsoirDiscovery? _discovery;
  RawDatagramSocket? _udpSocket;
  StreamSubscription? _bonsoirSubscription;
  StreamSubscription? _udpSubscription;
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
      _discovery = BonsoirDiscovery(type: '_nexus._tcp');
      await _discovery!.ready;

      _bonsoirSubscription = _discovery!.eventStream!.listen((event) {
        if (event.type == BonsoirDiscoveryEventType.discoveryServiceResolved) {
          final service = event.service as ResolvedBonsoirService;
          _addServer(DiscoveredServer(
            name: service.name,
            host: service.host ?? service.name,
            port: service.port,
          ));
        }
      });

      await _discovery!.start();

      // Auto-stop discovery after 10 seconds to save resources
      Future.delayed(const Duration(seconds: 10), () {
        if (_isSearching) {
          debugPrint('Auto-stopping discovery after timeout');
          stopDiscovery();
        }
      });
    } catch (e) {
      debugPrint('mDNS Discovery failed: $e');
      _isSearching = false;
      _error = e.toString();
      notifyListeners();
    }

    _startUDPDiscovery();
  }

  Future<void> _startUDPDiscovery() async {
    try {
      _udpSocket = await RawDatagramSocket.bind(InternetAddress.anyIPv4, 9999);
      _udpSubscription = _udpSocket!.listen(
        (RawSocketEvent event) {
          if (event == RawSocketEvent.read) {
            Datagram? dg = _udpSocket!.receive();
            if (dg != null) {
              final message = String.fromCharCodes(dg.data);
              if (message.startsWith("NEXUS:")) {
                final parts = message.split(":");
                if (parts.length >= 3) {
                  _addServer(DiscoveredServer(
                    name: "LAN (UDP)",
                    host: parts[1],
                    port: int.tryParse(parts[2]) ?? 8443,
                  ));
                }
              }
            }
          }
        },
        onError: (e) {
          debugPrint('UDP socket error: $e');
        },
      );
    } catch (e) {
      debugPrint('UDP Discovery failed: $e');
    }
  }

  void _addServer(DiscoveredServer server) {
    if (!_servers.any((s) => s.host == server.host && s.port == server.port)) {
      _servers.add(server);
      _selectedServer ??= server;
      notifyListeners();
    }
  }

  /// Stop searching
  Future<void> stopDiscovery() async {
    await _bonsoirSubscription?.cancel();
    _bonsoirSubscription = null;
    if (_discovery != null) {
      await _discovery!.stop();
      _discovery = null;
    }
    await _udpSubscription?.cancel();
    _udpSubscription = null;
    if (_udpSocket != null) {
      _udpSocket!.close();
      _udpSocket = null;
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

  /// Set server from scanned QR code URL
  void setFromQR(String url) {
    try {
      final uri = Uri.parse(url);
      setManualServer(uri.host, uri.port);
    } catch (e) {
      _error = "Invalid QR code URL";
      notifyListeners();
    }
  }

  @override
  void dispose() {
    stopDiscovery();
    super.dispose();
  }
}
