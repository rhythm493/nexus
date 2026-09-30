import 'package:flutter/material.dart';

import 'content_block.dart';

/// Signature for a component widget builder.
///
/// [block] — the content block to render.
/// [onAction] — callback for interactive components; fires when user interacts.
/// Returns a [Widget] that renders the component.
typedef ComponentWidgetBuilder = Widget Function(
  ContentBlock block,
  BuildContext context, {
  void Function(String action, Map<String, dynamic>? args)? onAction,
});

/// IoC-style registry that maps [ComponentType] to [ComponentWidgetBuilder].
///
/// Register builders at app startup, then build components by type.
/// Unknown types render a fallback instead of crashing.
class WidgetRegistry {
  static final Map<ComponentType, ComponentWidgetBuilder> _builders = {};
  static ComponentWidgetBuilder? _fallback;

  /// Register a builder for [type].
  static void register(ComponentType type, ComponentWidgetBuilder builder) {
    _builders[type] = builder;
  }

  /// Register a fallback builder for unregistered types.
  static void setFallback(ComponentWidgetBuilder builder) {
    _fallback = builder;
  }

  /// Returns true if a builder is registered for [type].
  static bool hasBuilder(ComponentType type) => _builders.containsKey(type);

  /// Build a widget for [block] using the registered builder.
  ///
  /// If no builder is registered for the block's type, returns the fallback
  /// or a simple error placeholder.
  static Widget build(
    ContentBlock block,
    BuildContext context, {
    void Function(String action, Map<String, dynamic>? args)? onAction,
  }) {
    final builder = _builders[block.type];
    if (builder != null) {
      return builder(block, context, onAction: onAction);
    }
    if (_fallback != null) {
      return _fallback!(block, context, onAction: onAction);
    }
    return _defaultFallback(block);
  }

  /// Build a sequence of blocks.
  static List<Widget> buildAll(
    List<ContentBlock> blocks,
    BuildContext context, {
    void Function(String action, Map<String, dynamic>? args)? onAction,
  }) {
    return blocks.map((b) => build(b, context, onAction: onAction)).toList();
  }

  static Widget _defaultFallback(ContentBlock block) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 2),
      child: Text(
        '[${block.type.name}]',
        style: const TextStyle(color: Colors.grey, fontSize: 11),
      ),
    );
  }

  /// Clear all registered builders (for testing).
  static void reset() {
    _builders.clear();
    _fallback = null;
  }
}
