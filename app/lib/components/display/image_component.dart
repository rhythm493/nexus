import 'package:flutter/material.dart';

import 'package:nexus/core/content_block.dart';
import 'package:nexus/core/widget_registry.dart';

class ImageComponent extends StatelessWidget {
  final ContentBlock block;
  final void Function(String action, Map<String, dynamic>? args)? onAction;

  const ImageComponent({
    super.key,
    required this.block,
    this.onAction,
  });

  @override
  Widget build(BuildContext context) {
    final source = block.data['source'] as String?;
    if (source == null || source.isEmpty) {
      return const SizedBox(
        width: double.infinity,
        child: Text(
          '[Image] Missing source',
          style: TextStyle(color: Colors.grey, fontSize: 11),
        ),
      );
    }

    final alt = block.data['alt'] as String?;
    final caption = block.data['caption'] as String?;
    final width =
        block.data['width'] != null
            ? (block.data['width'] as num).toDouble()
            : null;
    final height =
        block.data['height'] != null
            ? (block.data['height'] as num).toDouble()
            : null;
    final cs = Theme.of(context).colorScheme;

    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        ClipRRect(
          borderRadius: BorderRadius.circular(8),
          child: Image.network(
            source,
            width: width,
            height: height,
            fit: BoxFit.contain,
            loadingBuilder: (context, child, loadingProgress) {
              if (loadingProgress == null) return child;
              final total = loadingProgress.expectedTotalBytes;
              final progress =
                  total != null
                      ? loadingProgress.cumulativeBytesLoaded / total
                      : null;
              return Container(
                width: width ?? double.infinity,
                height: height ?? 200,
                color: cs.surfaceContainerHighest,
                child: Center(
                  child: CircularProgressIndicator(value: progress),
                ),
              );
            },
            errorBuilder: (context, error, stackTrace) {
              return Container(
                width: width ?? double.infinity,
                height: height ?? 200,
                color: cs.surfaceContainerHighest,
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Icon(
                      Icons.broken_image,
                      size: 48,
                      color: cs.onSurface.withValues(alpha: 0.3),
                    ),
                    const SizedBox(height: 8),
                    Text(
                      alt ?? 'Failed to load image',
                      style: TextStyle(
                        fontSize: 12,
                        color: cs.onSurface.withValues(alpha: 0.5),
                      ),
                    ),
                  ],
                ),
              );
            },
          ),
        ),
        if (caption != null)
          Padding(
            padding: const EdgeInsets.only(top: 4),
            child: Text(
              caption,
              style: TextStyle(
                fontSize: 11,
                fontStyle: FontStyle.italic,
                color: cs.onSurface.withValues(alpha: 0.5),
              ),
            ),
          ),
      ],
    );
  }
}

void registerImageComponent() {
  WidgetRegistry.register(ComponentType.image, (block, context, {onAction}) {
    return ImageComponent(block: block, onAction: onAction);
  });
}
