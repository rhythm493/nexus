import 'package:flutter/material.dart';

import 'package:nexus/core/content_block.dart';
import 'package:nexus/core/widget_registry.dart';

class _DropdownComponent extends StatefulWidget {
  final ContentBlock block;
  final void Function(String action, Map<String, dynamic>? args)? onAction;

  const _DropdownComponent({required this.block, this.onAction});

  @override
  State<_DropdownComponent> createState() => _DropdownComponentState();
}

class _DropdownComponentState extends State<_DropdownComponent> {
  String? _selected;

  @override
  void initState() {
    super.initState();
    _selected = widget.block.data['selected'] as String?;
  }

  @override
  void didUpdateWidget(_DropdownComponent oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.block.data['selected'] != widget.block.data['selected']) {
      _selected = widget.block.data['selected'] as String?;
    }
  }

  void _onChanged(String? newValue) {
    if (newValue == null ||
        widget.block.action == null ||
        widget.onAction == null) {
      return;
    }
    setState(() => _selected = newValue);
    final args = Map<String, dynamic>.from(widget.block.actionArgs ?? {});
    args['selected'] = newValue;
    widget.onAction!(widget.block.action!, args);
  }

  @override
  Widget build(BuildContext context) {
    final label = widget.block.data['label'] as String? ?? '';
    final rawOptions =
        (widget.block.data['options'] as List<dynamic>?) ?? <dynamic>[];
    final options = rawOptions
        .map((o) => DropdownMenuItem<String>(
              value: o['value'] as String? ?? '',
              child: Text(o['label'] as String? ?? ''),
            ))
        .toList();
    final isInteractive =
        widget.block.action != null && widget.onAction != null;

    return InputDecorator(
      decoration: InputDecoration(
        labelText: label,
        border: const OutlineInputBorder(),
      ),
      child: DropdownButtonHideUnderline(
        child: DropdownButton<String>(
          value: _selected,
          isDense: true,
          isExpanded: true,
          items: options,
          onChanged: isInteractive ? _onChanged : null,
        ),
      ),
    );
  }
}

void registerDropdownComponent() {
  WidgetRegistry.register(
      ComponentType.dropdown, (block, context, {onAction}) {
    return _DropdownComponent(block: block, onAction: onAction);
  });
}
