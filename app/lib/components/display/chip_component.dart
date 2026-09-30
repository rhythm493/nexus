import 'package:flutter/material.dart';

import 'package:nexus/core/content_block.dart';
import 'package:nexus/core/widget_registry.dart';

IconData _chipIconFromString(String? name) {
  if (name == null) return Icons.help_outline;
  switch (name) {
    case 'check':
      return Icons.check;
    case 'close':
      return Icons.close;
    case 'add':
      return Icons.add;
    case 'remove':
      return Icons.remove;
    case 'edit':
      return Icons.edit;
    case 'delete':
      return Icons.delete;
    case 'search':
      return Icons.search;
    case 'filter':
    case 'filter_list':
      return Icons.filter_list;
    case 'sort':
      return Icons.sort;
    case 'more':
      return Icons.more_horiz;
    case 'settings':
      return Icons.settings;
    case 'refresh':
      return Icons.refresh;
    case 'star':
      return Icons.star;
    case 'favorite':
      return Icons.favorite;
    case 'person':
      return Icons.person;
    case 'home':
      return Icons.home;
    case 'menu':
      return Icons.menu;
    case 'arrow_drop_down':
      return Icons.arrow_drop_down;
    default:
      return Icons.help_outline;
  }
}

class ChipComponent extends StatelessWidget {
  final ContentBlock block;
  final void Function(String action, Map<String, dynamic>? args)? onAction;

  const ChipComponent({
    super.key,
    required this.block,
    this.onAction,
  });

  @override
  Widget build(BuildContext context) {
    final label = block.data['label'] as String?;
    if (label == null || label.isEmpty) {
      return const SizedBox(
        width: double.infinity,
        child: Text(
          '[Chip] Missing label',
          style: TextStyle(color: Colors.grey, fontSize: 11),
        ),
      );
    }

    final iconName = block.data['icon'] as String?;
    final selected = block.data['selected'] as bool? ?? false;

    final icon = iconName != null
        ? Icon(_chipIconFromString(iconName), size: 16)
        : null;

    final hasAction = block.action != null;

    if (hasAction) {
      return ActionChip(
        avatar: icon,
        label: Text(label),
        onPressed: () {
          onAction?.call(block.action!, block.actionArgs);
        },
      );
    }

    return FilterChip(
      avatar: icon,
      label: Text(label),
      selected: selected,
      onSelected: null,
    );
  }
}

void registerChipComponent() {
  WidgetRegistry.register(ComponentType.chip, (block, context, {onAction}) {
    return ChipComponent(block: block, onAction: onAction);
  });
}
