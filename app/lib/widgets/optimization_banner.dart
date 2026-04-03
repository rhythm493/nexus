import 'package:flutter/material.dart';
import '../models/cart_state.dart';

/// Displays the optimization result — single provider or split recommendation.
class OptimizationBanner extends StatelessWidget {
  final OptimizationResult optimization;

  const OptimizationBanner({super.key, required this.optimization});

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;

    if (optimization.isSplit) {
      return _splitBanner(context, cs);
    }
    return _singleBanner(context, cs);
  }

  Widget _singleBanner(BuildContext context, ColorScheme cs) {
    final best = optimization.singleBest;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: BoxDecoration(
        color: cs.primary.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        children: [
          Icon(Icons.check_circle_outline, size: 16, color: cs.primary),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              'Best: All from ${_shortProvider(best.provider)} \u2014 ${CartState.formatPaise(best.totalPaise)}',
              style: TextStyle(fontSize: 12, fontWeight: FontWeight.w500, color: cs.primary),
            ),
          ),
        ],
      ),
    );
  }

  Widget _splitBanner(BuildContext context, ColorScheme cs) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: BoxDecoration(
        color: Colors.green.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(Icons.savings_outlined, size: 16, color: Colors.green[400]),
              const SizedBox(width: 8),
              Text(
                'Split saves ${CartState.formatPaise(optimization.splitSavings)} (${optimization.splitSavingsPct.toInt()}%)',
                style: TextStyle(fontSize: 12, fontWeight: FontWeight.w600, color: Colors.green[400]),
              ),
            ],
          ),
          const SizedBox(height: 4),
          ...optimization.splitCarts.map((pc) => Padding(
                padding: const EdgeInsets.only(left: 24, top: 2),
                child: Text(
                  '${_shortProvider(pc.provider)}: ${pc.itemCount} items \u2014 ${CartState.formatPaise(pc.totalPaise)}',
                  style: TextStyle(fontSize: 11, color: cs.onSurface.withValues(alpha: 0.7)),
                ),
              )),
          Padding(
            padding: const EdgeInsets.only(left: 24, top: 2),
            child: Text(
              'Total: ${CartState.formatPaise(optimization.splitTotalPaise)}',
              style: TextStyle(fontSize: 11, fontWeight: FontWeight.w500, color: cs.onSurface.withValues(alpha: 0.9)),
            ),
          ),
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
