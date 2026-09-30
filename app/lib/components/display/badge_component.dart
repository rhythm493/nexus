import 'package:flutter/material.dart';

import 'package:nexus/core/content_block.dart';
import 'package:nexus/core/widget_registry.dart';

Color _parseColor(String? colorStr) {
  if (colorStr == null) return Colors.grey;

  if (colorStr.startsWith('#')) {
    final hex = colorStr.replaceAll('#', '');
    if (hex.length == 6) {
      return Color(int.parse('FF$hex', radix: 16));
    } else if (hex.length == 8) {
      return Color(int.parse(hex, radix: 16));
    }
  }

  switch (colorStr.toLowerCase()) {
    case 'red':
      return Colors.red;
    case 'blue':
      return Colors.blue;
    case 'green':
      return Colors.green;
    case 'yellow':
      return Colors.yellow;
    case 'orange':
      return Colors.orange;
    case 'purple':
      return Colors.purple;
    case 'pink':
      return Colors.pink;
    case 'cyan':
      return Colors.cyan;
    case 'teal':
      return Colors.teal;
    case 'grey':
    case 'gray':
      return Colors.grey;
    case 'white':
      return Colors.white;
    case 'black':
      return Colors.black;
    case 'amber':
      return Colors.amber;
    case 'indigo':
      return Colors.indigo;
    case 'lime':
      return Colors.lime;
    case 'brown':
      return Colors.brown;
    default:
      return Colors.grey;
  }
}

IconData _badgeIconFromString(String? name) {
  if (name == null) return Icons.help_outline;
  switch (name) {
    case 'check':
    case 'check_circle':
      return Icons.check_circle;
    case 'close':
      return Icons.close;
    case 'info':
      return Icons.info;
    case 'warning':
      return Icons.warning;
    case 'error':
      return Icons.error;
    case 'star':
      return Icons.star;
    case 'favorite':
      return Icons.favorite;
    case 'person':
      return Icons.person;
    case 'lock':
      return Icons.lock;
    case 'clock':
    case 'time':
      return Icons.access_time;
    case 'bolt':
    case 'lightning':
      return Icons.bolt;
    default:
      return Icons.circle;
  }
}

class BadgeComponent extends StatelessWidget {
  final ContentBlock block;
  final void Function(String action, Map<String, dynamic>? args)? onAction;

  const BadgeComponent({
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
          '[Badge] Missing text',
          style: TextStyle(color: Colors.grey, fontSize: 11),
        ),
      );
    }

    final colorStr = block.data['color'] as String?;
    final iconName = block.data['icon'] as String?;

    final bgColor = _parseColor(colorStr);
    final textColor =
        bgColor.computeLuminance() > 0.5 ? Colors.black87 : Colors.white;

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: bgColor,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (iconName != null) ...[
            Icon(
              _badgeIconFromString(iconName),
              size: 14,
              color: textColor,
            ),
            const SizedBox(width: 4),
          ],
          Text(
            text,
            style: TextStyle(
              fontSize: 11,
              fontWeight: FontWeight.w600,
              color: textColor,
            ),
          ),
        ],
      ),
    );
  }
}

void registerBadgeComponent() {
  WidgetRegistry.register(ComponentType.badge, (block, context, {onAction}) {
    return BadgeComponent(block: block, onAction: onAction);
  });
}
