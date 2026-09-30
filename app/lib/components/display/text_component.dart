import 'package:flutter/material.dart';

import 'package:nexus/core/content_block.dart';
import 'package:nexus/core/widget_registry.dart';

class TextComponent extends StatelessWidget {
  final ContentBlock block;
  final void Function(String action, Map<String, dynamic>? args)? onAction;

  const TextComponent({
    super.key,
    required this.block,
    this.onAction,
  });

  @override
  Widget build(BuildContext context) {
    final text = block.data['text'] as String?;
    if (text == null || text.isEmpty) {
      return const SizedBox(
        width: double.infinity,
        child: Text(
          '[Text] Missing data',
          style: TextStyle(color: Colors.grey, fontSize: 11),
        ),
      );
    }

    final cs = Theme.of(context).colorScheme;

    final sizeStr = block.data['size'] as String?;
    final weightStr = block.data['weight'] as String?;
    final colorStr = block.data['color'] as String?;
    final alignmentStr = block.data['alignment'] as String?;

    double fontSize;
    switch (sizeStr) {
      case 'small':
        fontSize = 11;
      case 'large':
        fontSize = 18;
      default:
        fontSize = 14;
    }

    final fontWeight =
        weightStr == 'bold' ? FontWeight.bold : FontWeight.normal;

    Color textColor = cs.onSurface;
    if (colorStr != null) {
      if (colorStr.startsWith('#')) {
        final hex = colorStr.replaceAll('#', '');
        if (hex.length == 6) {
          textColor = Color(int.parse('FF$hex', radix: 16));
        } else if (hex.length == 8) {
          textColor = Color(int.parse(hex, radix: 16));
        }
      } else {
        switch (colorStr.toLowerCase()) {
          case 'primary':
            textColor = cs.primary;
          case 'secondary':
            textColor = cs.secondary;
          case 'tertiary':
            textColor = cs.tertiary;
          case 'error':
            textColor = cs.error;
          case 'onSurface':
            textColor = cs.onSurface;
          case 'onSurfaceVariant':
            textColor = cs.onSurfaceVariant;
          case 'surface':
            textColor = cs.surface;
          case 'white':
            textColor = Colors.white;
          case 'black':
            textColor = Colors.black;
          case 'grey':
          case 'gray':
            textColor = Colors.grey;
        }
      }
    }

    TextAlign alignment;
    switch (alignmentStr) {
      case 'center':
        alignment = TextAlign.center;
      case 'right':
        alignment = TextAlign.right;
      case 'justify':
        alignment = TextAlign.justify;
      default:
        alignment = TextAlign.start;
    }

    return Text(
      text,
      style: TextStyle(
        fontSize: fontSize,
        fontWeight: fontWeight,
        color: textColor,
        height: 1.4,
      ),
      textAlign: alignment,
    );
  }
}

void registerTextComponent() {
  WidgetRegistry.register(ComponentType.text, (block, context, {onAction}) {
    return TextComponent(block: block, onAction: onAction);
  });
}
