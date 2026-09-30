import 'package:flutter/material.dart';

import 'package:nexus/core/content_block.dart';
import 'package:nexus/core/widget_registry.dart';

class _SliderComponent extends StatefulWidget {
  final ContentBlock block;
  final void Function(String action, Map<String, dynamic>? args)? onAction;

  const _SliderComponent({required this.block, this.onAction});

  @override
  State<_SliderComponent> createState() => _SliderComponentState();
}

class _SliderComponentState extends State<_SliderComponent> {
  late double _value;

  @override
  void initState() {
    super.initState();
    _value = (widget.block.data['value'] as num?)?.toDouble() ?? 0;
  }

  @override
  void didUpdateWidget(_SliderComponent oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.block.data['value'] != widget.block.data['value']) {
      _value = (widget.block.data['value'] as num?)?.toDouble() ?? 0;
    }
  }

  void _onChanged(double newValue) {
    setState(() => _value = newValue);
  }

  void _onChangeEnd(double newValue) {
    if (widget.block.action == null || widget.onAction == null) return;
    final args = Map<String, dynamic>.from(widget.block.actionArgs ?? {});
    args['value'] = newValue;
    widget.onAction!(widget.block.action!, args);
  }

  @override
  Widget build(BuildContext context) {
    final label = widget.block.data['label'] as String? ?? '';
    final min = (widget.block.data['min'] as num?)?.toDouble() ?? 0;
    final max = (widget.block.data['max'] as num?)?.toDouble() ?? 100;
    final isInteractive =
        widget.block.action != null && widget.onAction != null;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(label, style: Theme.of(context).textTheme.bodyMedium),
            Text(_value.toStringAsFixed(1),
                style: Theme.of(context).textTheme.bodySmall),
          ],
        ),
        Slider(
          value: _value.clamp(min, max),
          min: min,
          max: max,
          divisions: ((max - min) % 1 == 0 && (max - min) <= 100)
              ? (max - min).toInt()
              : null,
          onChanged: isInteractive ? _onChanged : null,
          onChangeEnd: isInteractive ? _onChangeEnd : null,
        ),
      ],
    );
  }
}

void registerSliderComponent() {
  WidgetRegistry.register(ComponentType.slider, (block, context, {onAction}) {
    return _SliderComponent(block: block, onAction: onAction);
  });
}
