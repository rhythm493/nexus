import 'package:flutter/material.dart';

import 'package:nexus/core/content_block.dart';

class SpacerComponent {
  const SpacerComponent();

  static Widget build(ContentBlock block, BuildContext context, {void Function(String action, Map<String, dynamic>? args)? onAction}) {
    final data = block.data;
    final flex = data['flex'] as int? ?? 1;

    return Spacer(flex: flex);
  }
}
