import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'package:flutter/material.dart';

class InlineSearch extends StatefulWidget {
  final String? baseUrl;
  final void Function(String query, String provider) onAdd;

  const InlineSearch({super.key, required this.baseUrl, required this.onAdd});

  @override
  State<InlineSearch> createState() => _InlineSearchState();
}

class _InlineSearchState extends State<InlineSearch> {
  final _controller = TextEditingController();
  final _focusNode = FocusNode();
  final _layerLink = LayerLink();
  OverlayEntry? _overlay;
  Timer? _debounce;
  Map<String, List<_SearchProduct>>? _results;
  bool _isSearching = false;

  @override
  void dispose() {
    _debounce?.cancel();
    _removeOverlay();
    _controller.dispose();
    _focusNode.dispose();
    super.dispose();
  }

  void _onChanged(String value) {
    _debounce?.cancel();
    if (value.trim().length < 2) {
      _removeOverlay();
      return;
    }
    _debounce = Timer(const Duration(milliseconds: 400), () => _search(value.trim()));
  }

  Future<void> _search(String query) async {
    if (widget.baseUrl == null) return;
    setState(() => _isSearching = true);
    _showOverlay();

    try {
      final client = HttpClient()..badCertificateCallback = (_, __, ___) => true;
      final req = await client.getUrl(Uri.parse('${widget.baseUrl}/api/v1/search?q=${Uri.encodeComponent(query)}'));
      final resp = await req.close();
      if (resp.statusCode != 200) return;

      final body = await resp.transform(utf8.decoder).join();
      final json = jsonDecode(body) as Map<String, dynamic>;
      final results = <String, List<_SearchProduct>>{};

      final providerResults = json['results'] as Map<String, dynamic>? ?? {};
      for (final entry in providerResults.entries) {
        final products = (entry.value['products'] as List? ?? [])
            .take(3)
            .map((p) => _SearchProduct.fromJson(p as Map<String, dynamic>, entry.key))
            .toList();
        if (products.isNotEmpty) results[entry.key] = products;
      }

      if (mounted) {
        setState(() {
          _results = results;
          _isSearching = false;
        });
        _showOverlay();
      }
    } catch (e) {
      if (mounted) setState(() => _isSearching = false);
    }
  }

  void _showOverlay() {
    _removeOverlay();
    _overlay = OverlayEntry(builder: (_) => _buildDropdown());
    Overlay.of(context).insert(_overlay!);
  }

  void _removeOverlay() {
    _overlay?.remove();
    _overlay = null;
  }

  void _selectProduct(_SearchProduct product) {
    widget.onAdd(product.name, product.source);
    _controller.clear();
    _results = null;
    _removeOverlay();
    _focusNode.unfocus();
  }

  Widget _buildDropdown() {
    return Positioned(
      width: MediaQuery.of(context).size.width - 24,
      child: CompositedTransformFollower(
        link: _layerLink,
        offset: const Offset(0, 48),
        child: Material(
          elevation: 8,
          borderRadius: BorderRadius.circular(12),
          color: Theme.of(context).colorScheme.surfaceContainerHigh,
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxHeight: 300),
            child: _isSearching
                ? const Padding(
                    padding: EdgeInsets.all(16),
                    child: Center(child: CircularProgressIndicator(strokeWidth: 2)),
                  )
                : _results == null || _results!.isEmpty
                    ? const Padding(
                        padding: EdgeInsets.all(16),
                        child: Text('No results', style: TextStyle(color: Colors.grey)),
                      )
                    : ListView(
                        shrinkWrap: true,
                        padding: const EdgeInsets.symmetric(vertical: 4),
                        children: _results!.entries.expand((entry) {
                          final provider = entry.key;
                          return [
                            Padding(
                              padding: const EdgeInsets.fromLTRB(12, 8, 12, 4),
                              child: Text(
                                _shortProvider(provider),
                                style: TextStyle(
                                  fontSize: 10,
                                  fontWeight: FontWeight.w600,
                                  color: Theme.of(context).colorScheme.primary,
                                ),
                              ),
                            ),
                            ...entry.value.map((p) => _productTile(p)),
                          ];
                        }).toList(),
                      ),
          ),
        ),
      ),
    );
  }

  Widget _productTile(_SearchProduct product) {
    final cs = Theme.of(context).colorScheme;
    return InkWell(
      onTap: () => _selectProduct(product),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
        child: Row(
          children: [
            ClipRRect(
              borderRadius: BorderRadius.circular(6),
              child: product.imageUrl.isNotEmpty
                  ? Image.network(product.imageUrl, width: 36, height: 36, fit: BoxFit.cover,
                      cacheWidth: 72, errorBuilder: (_, __, ___) => _imgPlaceholder(cs))
                  : _imgPlaceholder(cs),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(product.name, style: const TextStyle(fontSize: 12), maxLines: 1, overflow: TextOverflow.ellipsis),
                  Text(product.quantity, style: TextStyle(fontSize: 10, color: cs.onSurface.withValues(alpha: 0.5))),
                ],
              ),
            ),
            Text(product.priceDisplay, style: TextStyle(fontSize: 12, fontWeight: FontWeight.w600, color: cs.onSurface)),
          ],
        ),
      ),
    );
  }

  Widget _imgPlaceholder(ColorScheme cs) {
    return Container(
      width: 36, height: 36,
      decoration: BoxDecoration(color: cs.surfaceContainerLow, borderRadius: BorderRadius.circular(6)),
      child: Icon(Icons.shopping_bag_outlined, size: 16, color: cs.onSurface.withValues(alpha: 0.3)),
    );
  }

  String _shortProvider(String p) {
    switch (p) { case 'blinkit': return 'Blinkit'; case 'instamart': return 'Instamart'; case 'zepto': return 'Zepto'; default: return p; }
  }

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;
    return CompositedTransformTarget(
      link: _layerLink,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(12, 8, 12, 4),
        child: TextField(
          controller: _controller,
          focusNode: _focusNode,
          onChanged: _onChanged,
          decoration: InputDecoration(
            hintText: 'Search for items...',
            hintStyle: TextStyle(fontSize: 13, color: cs.onSurface.withValues(alpha: 0.4)),
            prefixIcon: Icon(Icons.search, size: 20, color: cs.onSurface.withValues(alpha: 0.4)),
            suffixIcon: _controller.text.isNotEmpty
                ? IconButton(
                    icon: const Icon(Icons.clear, size: 18),
                    onPressed: () {
                      _controller.clear();
                      _removeOverlay();
                      setState(() => _results = null);
                    },
                  )
                : null,
            filled: true,
            fillColor: cs.surfaceContainerHigh,
            border: OutlineInputBorder(borderRadius: BorderRadius.circular(10), borderSide: BorderSide.none),
            contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
            isDense: true,
          ),
          style: const TextStyle(fontSize: 13),
        ),
      ),
    );
  }
}

class _SearchProduct {
  final String id, name, priceDisplay, quantity, imageUrl, source;

  _SearchProduct({required this.id, required this.name, required this.priceDisplay,
    required this.quantity, required this.imageUrl, required this.source});

  factory _SearchProduct.fromJson(Map<String, dynamic> json, String source) {
    return _SearchProduct(
      id: json['id'] as String? ?? '',
      name: json['name'] as String? ?? '',
      priceDisplay: json['priceDisplay'] as String? ?? '',
      quantity: json['quantity'] as String? ?? '',
      imageUrl: json['imageUrl'] as String? ?? '',
      source: source,
    );
  }
}
