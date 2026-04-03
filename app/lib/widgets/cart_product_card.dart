import 'package:flutter/material.dart';
import '../models/cart_full_state.dart';

class CartProductCard extends StatelessWidget {
  final CartFullItem item;
  final void Function(String provider) onSwap;
  final VoidCallback onRemove;

  const CartProductCard({
    super.key,
    required this.item,
    required this.onSwap,
    required this.onRemove,
  });

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;
    final best = item.bestProduct;
    if (best == null) return const SizedBox.shrink();

    return Dismissible(
      key: Key('cart_${item.query}'),
      direction: DismissDirection.endToStart,
      background: Container(
        alignment: Alignment.centerRight,
        padding: const EdgeInsets.only(right: 20),
        decoration: BoxDecoration(
          color: cs.error.withValues(alpha: 0.15),
          borderRadius: BorderRadius.circular(12),
        ),
        child: Icon(Icons.delete_outline, color: cs.error),
      ),
      onDismissed: (_) => onRemove(),
      child: Container(
        margin: const EdgeInsets.only(bottom: 8),
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: cs.surfaceContainerHigh,
          borderRadius: BorderRadius.circular(12),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Product info row
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                // Product image
                ClipRRect(
                  borderRadius: BorderRadius.circular(8),
                  child: best.imageUrl.isNotEmpty
                      ? Image.network(
                          best.imageUrl,
                          width: 52,
                          height: 52,
                          fit: BoxFit.cover,
                          cacheWidth: 104,
                          errorBuilder: (_, __, ___) => _placeholder(cs),
                        )
                      : _placeholder(cs),
                ),
                const SizedBox(width: 12),
                // Name, brand, quantity
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        best.name,
                        style: TextStyle(fontSize: 13, fontWeight: FontWeight.w500, color: cs.onSurface),
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                      ),
                      const SizedBox(height: 2),
                      Row(
                        children: [
                          if (best.brand != null && best.brand!.isNotEmpty)
                            Text(
                              '${best.brand}',
                              style: TextStyle(fontSize: 11, color: cs.onSurface.withValues(alpha: 0.6)),
                            ),
                          if (best.brand != null && best.brand!.isNotEmpty)
                            Text(' \u00b7 ', style: TextStyle(fontSize: 11, color: cs.onSurface.withValues(alpha: 0.4))),
                          Text(
                            best.quantity,
                            style: TextStyle(fontSize: 11, color: cs.onSurface.withValues(alpha: 0.6)),
                          ),
                        ],
                      ),
                    ],
                  ),
                ),
                // Price + discount
                Column(
                  crossAxisAlignment: CrossAxisAlignment.end,
                  children: [
                    Text(
                      best.priceDisplay,
                      style: TextStyle(fontSize: 15, fontWeight: FontWeight.w600, color: cs.onSurface),
                    ),
                    if (best.discountPct != null && best.discountPct! > 0)
                      Container(
                        margin: const EdgeInsets.only(top: 2),
                        padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 1),
                        decoration: BoxDecoration(
                          color: Colors.green.withValues(alpha: 0.15),
                          borderRadius: BorderRadius.circular(4),
                        ),
                        child: Text(
                          '${best.discountPct}% OFF',
                          style: TextStyle(fontSize: 9, fontWeight: FontWeight.w600, color: Colors.green[400]),
                        ),
                      ),
                  ],
                ),
              ],
            ),
            // Provider selection row
            if (item.selections.length > 1) ...[
              const SizedBox(height: 8),
              Wrap(
                spacing: 6,
                runSpacing: 4,
                children: item.sortedProviders.map((provider) {
                  final product = item.selections[provider]!;
                  final isSelected = provider == item.currentProvider;
                  return GestureDetector(
                    onTap: isSelected ? null : () => onSwap(provider),
                    child: Container(
                      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                      decoration: BoxDecoration(
                        color: isSelected
                            ? cs.primary.withValues(alpha: 0.15)
                            : cs.surfaceContainerLow,
                        borderRadius: BorderRadius.circular(6),
                        border: isSelected
                            ? Border.all(color: cs.primary.withValues(alpha: 0.4))
                            : null,
                      ),
                      child: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          if (isSelected)
                            Icon(Icons.check_circle, size: 12, color: cs.primary)
                          else
                            Icon(Icons.circle_outlined, size: 12, color: cs.onSurface.withValues(alpha: 0.4)),
                          const SizedBox(width: 4),
                          Text(
                            _shortProvider(provider),
                            style: TextStyle(
                              fontSize: 11,
                              fontWeight: isSelected ? FontWeight.w600 : FontWeight.w400,
                              color: isSelected ? cs.primary : cs.onSurface.withValues(alpha: 0.7),
                            ),
                          ),
                          const SizedBox(width: 4),
                          Text(
                            product.priceDisplay,
                            style: TextStyle(
                              fontSize: 11,
                              color: isSelected ? cs.primary : cs.onSurface.withValues(alpha: 0.5),
                            ),
                          ),
                        ],
                      ),
                    ),
                  );
                }).toList(),
              ),
            ],
          ],
        ),
      ),
    );
  }

  Widget _placeholder(ColorScheme cs) {
    return Container(
      width: 52,
      height: 52,
      decoration: BoxDecoration(
        color: cs.surfaceContainerLow,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Icon(Icons.shopping_bag_outlined, size: 24, color: cs.onSurface.withValues(alpha: 0.3)),
    );
  }

  String _shortProvider(String provider) {
    switch (provider) {
      case 'blinkit': return 'Blinkit';
      case 'instamart': return 'Instamart';
      case 'zepto': return 'Zepto';
      default: return provider;
    }
  }
}
