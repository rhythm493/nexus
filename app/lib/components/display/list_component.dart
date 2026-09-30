import 'package:flutter/material.dart';

import 'package:nexus/core/content_block.dart';
import 'package:nexus/core/widget_registry.dart';

IconData _iconFromString(String? name) {
  if (name == null) return Icons.help_outline;
  switch (name) {
    case 'home':
      return Icons.home;
    case 'check':
      return Icons.check;
    case 'close':
      return Icons.close;
    case 'person':
      return Icons.person;
    case 'search':
      return Icons.search;
    case 'settings':
      return Icons.settings;
    case 'star':
      return Icons.star;
    case 'favorite':
      return Icons.favorite;
    case 'info':
      return Icons.info;
    case 'warning':
      return Icons.warning;
    case 'error':
      return Icons.error;
    case 'add':
      return Icons.add;
    case 'remove':
      return Icons.remove;
    case 'edit':
      return Icons.edit;
    case 'delete':
      return Icons.delete;
    case 'play':
      return Icons.play_arrow;
    case 'pause':
      return Icons.pause;
    case 'stop':
      return Icons.stop;
    case 'music':
      return Icons.music_note;
    case 'volume':
      return Icons.volume_up;
    case 'email':
      return Icons.email;
    case 'phone':
      return Icons.phone;
    case 'location':
      return Icons.location_on;
    case 'refresh':
      return Icons.refresh;
    case 'more':
      return Icons.more_horiz;
    case 'menu':
      return Icons.menu;
    case 'arrow_back':
      return Icons.arrow_back;
    case 'arrow_forward':
      return Icons.arrow_forward;
    case 'check_circle':
      return Icons.check_circle;
    case 'cancel':
      return Icons.cancel;
    case 'time':
      return Icons.access_time;
    case 'calendar':
      return Icons.calendar_today;
    case 'link':
      return Icons.link;
    case 'image':
      return Icons.image;
    case 'download':
      return Icons.download;
    case 'upload':
      return Icons.upload;
    case 'send':
      return Icons.send;
    case 'done':
      return Icons.done;
    case 'list':
      return Icons.list;
    case 'shopping_cart':
      return Icons.shopping_cart;
    default:
      return Icons.help_outline;
  }
}

class ListComponent extends StatelessWidget {
  final ContentBlock block;
  final void Function(String action, Map<String, dynamic>? args)? onAction;

  const ListComponent({
    super.key,
    required this.block,
    this.onAction,
  });

  @override
  Widget build(BuildContext context) {
    final items = block.data['items'] as List<dynamic>?;
    if (items == null || items.isEmpty) {
      return const SizedBox(
        width: double.infinity,
        child: Text(
          '[List] Missing data',
          style: TextStyle(color: Colors.grey, fontSize: 11),
        ),
      );
    }

    final numbered = block.data['numbered'] as bool? ?? false;
    final cs = Theme.of(context).colorScheme;

    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: List.generate(items.length, (i) {
        final item = items[i] as Map<String, dynamic>;
        final title = item['title'] as String? ?? '';
        final subtitle = item['subtitle'] as String?;
        final iconName = item['icon'] as String?;
        final trailingStr = item['trailing'] as String?;

        return ListTile(
          leading:
              iconName != null
                  ? Icon(_iconFromString(iconName), size: 20, color: cs.primary)
                  : (numbered
                      ? SizedBox(
                          width: 20,
                          child: Text(
                            '${i + 1}.',
                            style: TextStyle(
                              fontSize: 13,
                              color: cs.onSurface.withValues(alpha: 0.5),
                            ),
                            textAlign: TextAlign.right,
                          ),
                        )
                      : null),
          title: Text(
            title,
            style: TextStyle(fontSize: 13, color: cs.onSurface),
          ),
          subtitle:
              subtitle != null
                  ? Text(
                      subtitle,
                      style: TextStyle(
                        fontSize: 11,
                        color: cs.onSurface.withValues(alpha: 0.6),
                      ),
                    )
                  : null,
          trailing:
              trailingStr != null
                  ? Text(
                      trailingStr,
                      style: TextStyle(
                        fontSize: 11,
                        color: cs.onSurface.withValues(alpha: 0.4),
                      ),
                    )
                  : null,
          dense: true,
          contentPadding: EdgeInsets.zero,
        );
      }),
    );
  }
}

void registerListComponent() {
  WidgetRegistry.register(ComponentType.list, (block, context, {onAction}) {
    return ListComponent(block: block, onAction: onAction);
  });
}
