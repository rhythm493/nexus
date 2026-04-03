import 'package:flutter/material.dart';

// CartPanel is deprecated — GroceryScreen has its own cart UI.
// Kept as stub to avoid breaking any remaining imports.
class CartPanel extends StatelessWidget {
  final void Function(String, String)? onSwap;
  final void Function(String)? onRemove;
  final VoidCallback? onClear;

  const CartPanel({super.key, this.onSwap, this.onRemove, this.onClear});

  @override
  Widget build(BuildContext context) => const SizedBox.shrink();
}
