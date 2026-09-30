import 'package:flutter/material.dart';

import 'package:nexus/core/content_block.dart';
import 'package:nexus/core/widget_registry.dart';

class _ToggleComponent extends StatefulWidget {
  final ContentBlock block;
  final void Function(String action, Map<String, dynamic>? args)? onAction;

  const _ToggleComponent({required this.block, this.onAction});

  @override
  State<_ToggleComponent> createState() => _ToggleComponentState();
}

class _ToggleComponentState extends State<_ToggleComponent> {
  late bool _value;

  @override
  void initState() {
    super.initState();
    _value = widget.block.data['value'] as bool? ?? false;
  }

  @override
  void didUpdateWidget(_ToggleComponent oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.block.data['value'] != widget.block.data['value']) {
      _value = widget.block.data['value'] as bool? ?? false;
    }
  }

  void _onChanged(bool newValue) {
    if (widget.block.action == null || widget.onAction == null) return;
    setState(() => _value = newValue);
    final args = Map<String, dynamic>.from(widget.block.actionArgs ?? {});
    args['value'] = newValue;
    widget.onAction!(widget.block.action!, args);
  }

  @override
  Widget build(BuildContext context) {
    final label = widget.block.data['label'] as String? ?? '';
    final isInteractive =
        widget.block.action != null && widget.onAction != null;

    return Row(
      children: [
        Expanded(
          child: Text(label, style: Theme.of(context).textTheme.bodyMedium),
        ),
        Switch(
          value: _value,
          onChanged: isInteractive ? _onChanged : null,
        ),
      ],
    );
  }
}

void registerToggleComponent() {
  WidgetRegistry.register(ComponentType.toggle, (block, context, {onAction}) {
    return _ToggleComponent(block: block, onAction: onAction);
  });
}
