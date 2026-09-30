import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../services/model_cache_service.dart';
import '../services/settings_service.dart';

class ModelInfoBar extends StatelessWidget {
  const ModelInfoBar({super.key});

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;
    final settings = context.watch<SettingsService>();
    final cache = context.watch<ModelCacheService>();

    final provider = settings.selectedProvider;
    final model = settings.selectedModel;
    if (provider == null || model == null) return const SizedBox.shrink();

    String label = model;
    String? subtitle;

    if (cache.isLoaded) {
      final info = cache.lookup(provider, model);
      if (info != null) {
        label = info.displayLabel;
        subtitle = info.subtitle;
      }
    }

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
      decoration: BoxDecoration(
        color: cs.surfaceContainerLow,
        border: Border(
          top: BorderSide(color: cs.outline.withValues(alpha: 0.1)),
          bottom: BorderSide(color: cs.outline.withValues(alpha: 0.1)),
        ),
      ),
      child: Row(
        children: [
          Icon(Icons.smart_toy, size: 14, color: cs.onSurface.withValues(alpha: 0.5)),
          const SizedBox(width: 6),
          Flexible(
            child: Text(
              label,
              style: TextStyle(
                fontSize: 12,
                color: cs.onSurface.withValues(alpha: 0.6),
              ),
              overflow: TextOverflow.ellipsis,
            ),
          ),
          if (subtitle != null) ...[
            Text(
              ' · ',
              style: TextStyle(
                fontSize: 12,
                color: cs.onSurface.withValues(alpha: 0.3),
              ),
            ),
            Flexible(
              child: Text(
                subtitle,
                style: TextStyle(
                  fontSize: 11,
                  color: cs.onSurface.withValues(alpha: 0.4),
                ),
                overflow: TextOverflow.ellipsis,
              ),
            ),
          ],
        ],
      ),
    );
  }
}
