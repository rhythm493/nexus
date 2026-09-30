import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../services/cart_service.dart';

class ContextBar extends StatelessWidget {
  const ContextBar({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer<CartService>(
      builder: (context, cartService, _) {
        final cart = cartService.fullCart;
        if (cart == null || cart.isEmpty) return const SizedBox.shrink();

        final cs = Theme.of(context).colorScheme;
        final total = cart.cheapestTotal;

        return Container(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
          decoration: BoxDecoration(
            color: cs.surfaceContainerLow,
            border: Border(top: BorderSide(color: cs.outline.withValues(alpha: 0.15))),
          ),
          child: Row(
            children: [
              Icon(Icons.shopping_cart, size: 16, color: cs.primary),
              const SizedBox(width: 8),
              Text(
                _formatPrice(total),
                style: TextStyle(fontSize: 14, fontWeight: FontWeight.w600, color: cs.onSurface),
              ),
              const SizedBox(width: 4),
              Text(
                '\u00b7 ${cart.itemCount} item${cart.itemCount == 1 ? '' : 's'}',
                style: TextStyle(fontSize: 12, color: cs.onSurface.withValues(alpha: 0.5)),
              ),
              const Spacer(),
              if (!cart.isOptimized && cart.itemCount > 1)
                TextButton.icon(
                  onPressed: () {},
                  icon: const Icon(Icons.auto_fix_high, size: 14),
                  label: const Text('Optimize', style: TextStyle(fontSize: 12)),
                  style: TextButton.styleFrom(
                    padding: const EdgeInsets.symmetric(horizontal: 8),
                    minimumSize: Size.zero,
                    tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                  ),
                ),
            ],
          ),
        );
      },
    );
  }

  String _formatPrice(int paise) {
    if (paise >= 100) {
      final rupees = paise / 100;
      if (rupees == rupees.roundToDouble()) {
        return '\u20B9${rupees.toInt()}';
      }
      return '\u20B9${rupees.toStringAsFixed(2)}';
    }
    return '\u20B9$paise';
  }
}
