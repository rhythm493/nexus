import 'package:flutter/material.dart';
import '../models/cart_state.dart';

/// A single cart item with swipe-to-swap provider functionality.
class CartItemTile extends StatefulWidget {
  final CartItemSummary item;
  final void Function(String provider) onSwap;
  final VoidCallback onRemove;

  const CartItemTile({
    super.key,
    required this.item,
    required this.onSwap,
    required this.onRemove,
  });

  @override
  State<CartItemTile> createState() => _CartItemTileState();
}

class _CartItemTileState extends State<CartItemTile> {
  late PageController _pageController;
  late List<String> _providers;
  late int _currentIndex;

  @override
  void initState() {
    super.initState();
    _providers = widget.item.sortedProviders;
    _currentIndex = 0;
    _pageController = PageController(initialPage: 0);
  }

  @override
  void didUpdateWidget(CartItemTile oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.item.query != widget.item.query) {
      _providers = widget.item.sortedProviders;
      _currentIndex = 0;
    }
  }

  @override
  void dispose() {
    _pageController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;

    return Dismissible(
      key: Key('cart_item_${widget.item.query}'),
      direction: DismissDirection.endToStart,
      background: Container(
        alignment: Alignment.centerRight,
        padding: const EdgeInsets.only(right: 16),
        decoration: BoxDecoration(
          color: cs.error.withValues(alpha: 0.2),
          borderRadius: BorderRadius.circular(10),
        ),
        child: Icon(Icons.delete_outline, color: cs.error),
      ),
      confirmDismiss: (_) async {
        widget.onRemove();
        return true;
      },
      child: Container(
        height: 52,
        margin: const EdgeInsets.only(bottom: 6),
        decoration: BoxDecoration(
          color: cs.surfaceContainerHigh,
          borderRadius: BorderRadius.circular(10),
        ),
        child: Row(
          children: [
            // Item query name
            Padding(
              padding: const EdgeInsets.only(left: 12),
              child: SizedBox(
                width: 70,
                child: Text(
                  widget.item.query,
                  style: TextStyle(
                    fontSize: 13,
                    fontWeight: FontWeight.w500,
                    color: cs.onSurface,
                  ),
                  overflow: TextOverflow.ellipsis,
                ),
              ),
            ),
            // Swipeable provider cards
            Expanded(
              child: _providers.length <= 1
                  ? _buildProviderCard(context, _providers.isNotEmpty ? _providers[0] : '')
                  : PageView.builder(
                      controller: _pageController,
                      itemCount: _providers.length,
                      onPageChanged: (index) {
                        setState(() => _currentIndex = index);
                        if (index != 0) {
                          widget.onSwap(_providers[index]);
                        }
                      },
                      itemBuilder: (context, index) => _buildProviderCard(context, _providers[index]),
                    ),
            ),
            // Provider dots
            if (_providers.length > 1)
              Padding(
                padding: const EdgeInsets.only(right: 8),
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: List.generate(_providers.length, (i) {
                    return Container(
                      width: 5,
                      height: 5,
                      margin: const EdgeInsets.symmetric(vertical: 1),
                      decoration: BoxDecoration(
                        shape: BoxShape.circle,
                        color: i == _currentIndex
                            ? cs.primary
                            : cs.onSurface.withValues(alpha: 0.3),
                      ),
                    );
                  }),
                ),
              ),
          ],
        ),
      ),
    );
  }

  Widget _buildProviderCard(BuildContext context, String provider) {
    final cs = Theme.of(context).colorScheme;
    final priceText = widget.item.prices[provider] ?? 'N/A';
    final isBest = provider == widget.item.bestProvider;

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 8),
      child: Row(
        children: [
          // Provider chip
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
            decoration: BoxDecoration(
              color: isBest
                  ? cs.primary.withValues(alpha: 0.15)
                  : cs.surfaceContainerLow,
              borderRadius: BorderRadius.circular(6),
            ),
            child: Text(
              _shortProvider(provider),
              style: TextStyle(
                fontSize: 11,
                fontWeight: isBest ? FontWeight.w600 : FontWeight.w400,
                color: isBest ? cs.primary : cs.onSurface.withValues(alpha: 0.7),
              ),
            ),
          ),
          const SizedBox(width: 8),
          // Price
          Expanded(
            child: Text(
              priceText,
              style: TextStyle(
                fontSize: 12,
                color: cs.onSurface.withValues(alpha: 0.8),
              ),
              overflow: TextOverflow.ellipsis,
            ),
          ),
          if (isBest)
            Icon(Icons.check_circle, size: 14, color: cs.primary),
        ],
      ),
    );
  }

  String _shortProvider(String provider) {
    switch (provider) {
      case 'blinkit':
        return 'Blinkit';
      case 'instamart':
        return 'Instamart';
      case 'zepto':
        return 'Zepto';
      default:
        return provider;
    }
  }
}
