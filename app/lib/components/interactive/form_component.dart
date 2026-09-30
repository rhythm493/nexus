import 'package:flutter/material.dart';

import 'package:nexus/core/content_block.dart';
import 'package:nexus/core/widget_registry.dart';

class _FormComponent extends StatefulWidget {
  final ContentBlock block;
  final void Function(String action, Map<String, dynamic>? args)? onAction;

  const _FormComponent({required this.block, this.onAction});

  @override
  State<_FormComponent> createState() => _FormComponentState();
}

class _FormComponentState extends State<_FormComponent> {
  final _formKey = GlobalKey<FormState>();
  final _controllers = <String, TextEditingController>{};
  final _dropdownValues = <String, String>{};

  @override
  void initState() {
    super.initState();
    _initFields();
  }

  @override
  void didUpdateWidget(_FormComponent oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.block.data['fields'] != widget.block.data['fields']) {
      _disposeControllers();
      _initFields();
    }
  }

  @override
  void dispose() {
    _disposeControllers();
    super.dispose();
  }

  void _disposeControllers() {
    for (final c in _controllers.values) {
      c.dispose();
    }
    _controllers.clear();
  }

  void _initFields() {
    final fields = widget.block.data['fields'] as List<dynamic>? ?? [];
    for (final field in fields) {
      final name = field['name'] as String? ?? '';
      final type = field['type'] as String? ?? 'text';
      if (name.isEmpty) continue;
      if (type == 'select') {
        _dropdownValues[name] = '';
      } else {
        _controllers[name] = TextEditingController();
      }
    }
  }

  void _submit() {
    if (!(_formKey.currentState?.validate() ?? false)) return;

    final args = <String, dynamic>{};
    for (final entry in _controllers.entries) {
      args[entry.key] = entry.value.text;
    }
    for (final entry in _dropdownValues.entries) {
      if (entry.value.isNotEmpty) {
        args[entry.key] = entry.value;
      }
    }

    final action = widget.block.data['submitAction'] as String?;
    if (action != null && widget.onAction != null) {
      widget.onAction!(action, args);
    }
  }

  @override
  Widget build(BuildContext context) {
    final fields = widget.block.data['fields'] as List<dynamic>? ?? [];
    final submitLabel =
        widget.block.data['submitLabel'] as String? ?? 'Submit';
    final canSubmit =
        widget.block.data['submitAction'] != null && widget.onAction != null;

    return Form(
      key: _formKey,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          for (final field in fields) _buildField(field),
          if (fields.isNotEmpty) const SizedBox(height: 12),
          FilledButton(
            onPressed: canSubmit ? _submit : null,
            child: Text(submitLabel),
          ),
        ],
      ),
    );
  }

  Widget _buildField(dynamic field) {
    final name = field['name'] as String? ?? '';
    final label = field['label'] as String? ?? name;
    final type = field['type'] as String? ?? 'text';
    final required = field['required'] as bool? ?? false;

    if (type == 'select') {
      final rawOptions =
          (field['options'] as List<dynamic>?) ?? <dynamic>[];
      final options = rawOptions
          .map((o) => DropdownMenuItem<String>(
                value: o['value'] as String? ?? '',
                child: Text(o['label'] as String? ?? ''),
              ))
          .toList();

      return Padding(
        padding: const EdgeInsets.only(bottom: 8),
        child: DropdownButtonFormField<String>(
          initialValue: _dropdownValues[name]?.isEmpty != false ? null : _dropdownValues[name],
          decoration: InputDecoration(
            labelText: label,
            border: const OutlineInputBorder(),
          ),
          items: options,
          onChanged: widget.onAction != null
              ? (v) => setState(() => _dropdownValues[name] = v ?? '')
              : null,
          validator: required && widget.onAction != null
              ? (v) =>
                  (v == null || v.isEmpty) ? '$label is required' : null
              : null,
        ),
      );
    }

    TextInputType keyboardType;
    switch (type) {
      case 'number':
        keyboardType = TextInputType.number;
        break;
      case 'email':
        keyboardType = TextInputType.emailAddress;
        break;
      default:
        keyboardType = TextInputType.text;
    }

    final controller = _controllers.putIfAbsent(
      name,
      () => TextEditingController(),
    );

    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: TextFormField(
        controller: controller,
        decoration: InputDecoration(
          labelText: label,
          border: const OutlineInputBorder(),
        ),
        keyboardType: keyboardType,
        enabled: widget.onAction != null,
        validator: required && widget.onAction != null
            ? (v) =>
                (v == null || v.isEmpty) ? '$label is required' : null
            : null,
      ),
    );
  }
}

void registerFormComponent() {
  WidgetRegistry.register(ComponentType.form, (block, context, {onAction}) {
    return _FormComponent(block: block, onAction: onAction);
  });
}
