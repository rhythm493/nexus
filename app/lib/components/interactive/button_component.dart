import 'package:flutter/material.dart';

import 'package:nexus/core/content_block.dart';
import 'package:nexus/core/widget_registry.dart';

IconData? _iconFromString(String? name) {
  if (name == null) return null;
  switch (name) {
    case 'add':
      return Icons.add;
    case 'arrow_back':
      return Icons.arrow_back;
    case 'arrow_forward':
      return Icons.arrow_forward;
    case 'check':
      return Icons.check;
    case 'close':
      return Icons.close;
    case 'delete':
      return Icons.delete;
    case 'edit':
      return Icons.edit;
    case 'email':
      return Icons.email;
    case 'favorite':
      return Icons.favorite;
    case 'home':
      return Icons.home;
    case 'info':
      return Icons.info;
    case 'menu':
      return Icons.menu;
    case 'more_horiz':
      return Icons.more_horiz;
    case 'more_vert':
      return Icons.more_vert;
    case 'person':
      return Icons.person;
    case 'phone':
      return Icons.phone;
    case 'refresh':
      return Icons.refresh;
    case 'save':
      return Icons.save;
    case 'search':
      return Icons.search;
    case 'send':
      return Icons.send;
    case 'settings':
      return Icons.settings;
    case 'share':
      return Icons.share;
    case 'star':
      return Icons.star;
    case 'warning':
      return Icons.warning;
    case 'play_arrow':
      return Icons.play_arrow;
    case 'pause':
      return Icons.pause;
    case 'stop':
      return Icons.stop;
    case 'skip_next':
      return Icons.skip_next;
    case 'skip_previous':
      return Icons.skip_previous;
    case 'shuffle':
      return Icons.shuffle;
    case 'repeat':
      return Icons.repeat;
    default:
      return null;
  }
}

class _ButtonComponent extends StatelessWidget {
  final ContentBlock block;
  final void Function(String action, Map<String, dynamic>? args)? onAction;

  const _ButtonComponent({required this.block, this.onAction});

  @override
  Widget build(BuildContext context) {
    final label = block.data['label'] as String? ?? 'Button';
    final icon = _iconFromString(block.data['icon'] as String?);
    final variant = (block.data['variant'] as String?) ?? 'filled';
    final hasAction = block.action != null && onAction != null;

    final child = icon != null
        ? Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(icon, size: 18),
              const SizedBox(width: 8),
              Text(label),
            ],
          )
        : Text(label);

    final onPressed =
        hasAction ? () => onAction!(block.action!, block.actionArgs) : null;

    switch (variant) {
      case 'outlined':
        return OutlinedButton(onPressed: onPressed, child: child);
      case 'text':
        return TextButton(onPressed: onPressed, child: child);
      default:
        return FilledButton(onPressed: onPressed, child: child);
    }
  }
}

void registerButtonComponent() {
  WidgetRegistry.register(ComponentType.button, (block, context, {onAction}) {
    return _ButtonComponent(block: block, onAction: onAction);
  });
}
