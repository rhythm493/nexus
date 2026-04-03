import 'dart:convert';
import 'dart:io';
import 'package:flutter/foundation.dart';
import '../models/cart_full_state.dart';
import '../models/cart_state.dart';

class CartService extends ChangeNotifier {
  CartFullState? _fullCart;
  bool _isExpanded = false;
  final HttpClient _httpClient = HttpClient()
    ..badCertificateCallback = (cert, host, port) => true;

  CartFullState? get fullCart => _fullCart;
  bool get isExpanded => _isExpanded;
  bool get hasItems => _fullCart != null && _fullCart!.itemCount > 0;

  /// Update from enriched SSE cart_update event (full detail).
  void updateFromSSEFull(dynamic data) {
    debugPrint('[CartService] updateFromSSEFull: data type=${data.runtimeType}, isMap=${data is Map}');
    if (data is Map<String, dynamic>) {
      _fullCart = CartFullState.fromJson(data);
      debugPrint('[CartService] Cart updated: ${_fullCart?.itemCount} items, state=${_fullCart?.state}');
      notifyListeners();
    } else if (data is Map) {
      // Handle case where Dart JSON decoder returns Map<dynamic, dynamic>
      final converted = Map<String, dynamic>.from(data);
      _fullCart = CartFullState.fromJson(converted);
      debugPrint('[CartService] Cart updated (converted): ${_fullCart?.itemCount} items');
      notifyListeners();
    } else {
      debugPrint('[CartService] updateFromSSEFull: unexpected data type ${data.runtimeType}');
    }
  }

  /// Backward compat: update from compact SSE (old format).
  void updateFromSSE(dynamic data) {
    updateFromSSEFull(data);
  }

  /// Update optimization from SSE cart_optimized event.
  void updateOptimization(dynamic data) {
    if (_fullCart != null && data is Map<String, dynamic>) {
      _fullCart = CartFullState(
        id: _fullCart!.id,
        conversationId: _fullCart!.conversationId,
        state: 'optimized',
        itemCount: _fullCart!.itemCount,
        items: _fullCart!.items,
        optimization: OptimizationResult.fromJson(data),
      );
      notifyListeners();
    }
  }

  /// Fetch full cart from REST.
  Future<void> fetchFullCart(String baseUrl, String convId) async {
    try {
      final req = await _httpClient.getUrl(Uri.parse('$baseUrl/api/v1/cart/$convId/full'));
      final resp = await req.close();
      if (resp.statusCode == 200) {
        final body = await resp.transform(utf8.decoder).join();
        _fullCart = CartFullState.fromJson(jsonDecode(body) as Map<String, dynamic>);
        notifyListeners();
      }
    } catch (e) {
      debugPrint('[CartService] fetchFullCart error: $e');
    }
  }

  /// Add product from inline search (direct REST, no LLM).
  Future<bool> addProduct(String baseUrl, String convId, String query, String provider) async {
    try {
      final req = await _httpClient.postUrl(Uri.parse('$baseUrl/api/v1/cart/$convId/add'));
      req.headers.contentType = ContentType.json;
      req.write(jsonEncode({'query': query, 'provider': provider}));
      final resp = await req.close();
      if (resp.statusCode == 200) {
        final body = await resp.transform(utf8.decoder).join();
        final json = jsonDecode(body) as Map<String, dynamic>;
        if (json['cart'] != null) {
          _fullCart = CartFullState.fromJson(json['cart'] as Map<String, dynamic>);
          notifyListeners();
        }
        return json['success'] == true;
      }
    } catch (e) {
      debugPrint('[CartService] addProduct error: $e');
    }
    return false;
  }

  /// Swap item provider via REST.
  Future<bool> swapItem(String baseUrl, String convId, String query, String provider) async {
    try {
      final req = await _httpClient.postUrl(Uri.parse('$baseUrl/api/v1/cart/$convId/swap'));
      req.headers.contentType = ContentType.json;
      req.write(jsonEncode({'query': query, 'provider': provider}));
      final resp = await req.close();
      if (resp.statusCode == 200) {
        final body = await resp.transform(utf8.decoder).join();
        final json = jsonDecode(body) as Map<String, dynamic>;
        if (json['cart'] != null) {
          _fullCart = CartFullState.fromJson(json['cart'] as Map<String, dynamic>);
          notifyListeners();
        }
        return true;
      }
    } catch (e) {
      debugPrint('[CartService] swapItem error: $e');
    }
    return false;
  }

  /// Remove item via REST.
  Future<bool> removeItem(String baseUrl, String convId, String query) async {
    try {
      final req = await _httpClient.deleteUrl(
        Uri.parse('$baseUrl/api/v1/cart/$convId/item/${Uri.encodeComponent(query)}'),
      );
      final resp = await req.close();
      if (resp.statusCode == 200) {
        final body = await resp.transform(utf8.decoder).join();
        final json = jsonDecode(body) as Map<String, dynamic>;
        if (json['cart'] != null) {
          _fullCart = CartFullState.fromJson(json['cart'] as Map<String, dynamic>);
          notifyListeners();
        }
        return true;
      }
    } catch (e) {
      debugPrint('[CartService] removeItem error: $e');
    }
    return false;
  }

  void toggleExpanded() {
    _isExpanded = !_isExpanded;
    notifyListeners();
  }

  void clear() {
    _fullCart = null;
    _isExpanded = false;
    notifyListeners();
  }
}
