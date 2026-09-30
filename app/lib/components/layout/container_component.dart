import 'package:flutter/material.dart';

import 'package:nexus/core/content_block.dart';
import 'package:nexus/core/widget_registry.dart';

EdgeInsets? _parseEdgeInsets(Map<String, dynamic>? map) {
  if (map == null || map.isEmpty) return null;
  return EdgeInsets.only(
    left: (map['left'] as num?)?.toDouble() ?? 0,
    right: (map['right'] as num?)?.toDouble() ?? 0,
    top: (map['top'] as num?)?.toDouble() ?? 0,
    bottom: (map['bottom'] as num?)?.toDouble() ?? 0,
  );
}

Color? _parseColor(String? colorStr) {
  if (colorStr == null || colorStr.isEmpty) return null;
  try {
    final hex = colorStr.replaceFirst('#', '');
    final value = int.parse('FF$hex', radix: 16);
    return Color(value);
  } catch (_) {
    return null;
  }
}

BoxDecoration? _parseDecoration(Map<String, dynamic>? map) {
  if (map == null || map.isEmpty) return null;
  final color = _parseColor(map['color'] as String?);
  final borderRadius = (map['borderRadius'] as num?)?.toDouble();
  final borderColor = _parseColor(map['borderColor'] as String?);
  final borderWidth = (map['borderWidth'] as num?)?.toDouble() ?? 1;

  return BoxDecoration(
    color: color,
    borderRadius: borderRadius != null ? BorderRadius.circular(borderRadius) : null,
    border: borderColor != null ? Border.all(color: borderColor, width: borderWidth) : null,
  );
}

class ContainerComponent {
  const ContainerComponent();

  static Widget build(ContentBlock block, BuildContext context, {void Function(String action, Map<String, dynamic>? args)? onAction}) {
    final data = block.data;
    final children = block.children;

    Widget? child;
    if (children.isNotEmpty) {
      child = WidgetRegistry.build(children.first, context, onAction: onAction);
    }

    final padding = _parseEdgeInsets(data['padding'] as Map<String, dynamic>?);
    final margin = _parseEdgeInsets(data['margin'] as Map<String, dynamic>?);
    final width = (data['width'] as num?)?.toDouble();
    final height = (data['height'] as num?)?.toDouble();
    final decoration = _parseDecoration(data['decoration'] as Map<String, dynamic>?);

    return Container(
      padding: padding,
      margin: margin,
      width: width,
      height: height,
      decoration: decoration,
      child: child,
    );
  }
}
