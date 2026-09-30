import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'package:nexus/core/content_block.dart';
import 'package:nexus/core/widget_registry.dart';

class CodeComponent extends StatelessWidget {
  final ContentBlock block;
  final void Function(String action, Map<String, dynamic>? args)? onAction;

  const CodeComponent({
    super.key,
    required this.block,
    this.onAction,
  });

  @override
  Widget build(BuildContext context) {
    final code = block.data['code'] as String?;
    if (code == null || code.isEmpty) {
      return const SizedBox(
        width: double.infinity,
        child: Text(
          '[Code] Missing data',
          style: TextStyle(color: Colors.grey, fontSize: 11),
        ),
      );
    }

    final language = block.data['language'] as String?;
    final showLineNumbers =
        block.data['showLineNumbers'] as bool? ?? false;
    final cs = Theme.of(context).colorScheme;

    final lines = code.split('\n');

    return Container(
      width: double.infinity,
      margin: const EdgeInsets.symmetric(vertical: 4),
      decoration: BoxDecoration(
        color: const Color(0xFF1E1E1E),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: cs.outline.withValues(alpha: 0.2)),
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (language != null)
            _buildHeader(code, language, context, cs),
          Padding(
            padding: const EdgeInsets.all(12),
            child:
                showLineNumbers
                    ? _buildNumberedLines(lines)
                    : _buildPlainCode(code),
          ),
        ],
      ),
    );
  }

  Widget _buildHeader(
    String code,
    String language,
    BuildContext context,
    ColorScheme cs,
  ) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      decoration: const BoxDecoration(
        color: Color(0xFF2D2D2D),
        borderRadius: BorderRadius.vertical(top: Radius.circular(8)),
      ),
      child: Row(
        children: [
          Text(
            language,
            style: TextStyle(
              fontSize: 11,
              color: cs.primary,
              fontWeight: FontWeight.w600,
            ),
          ),
          const Spacer(),
          InkWell(
            borderRadius: BorderRadius.circular(4),
            onTap: () {
              Clipboard.setData(ClipboardData(text: code));
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(
                  content: Text('Copied'),
                  duration: Duration(seconds: 1),
                ),
              );
            },
            child: Padding(
              padding: const EdgeInsets.all(4),
              child: Icon(
                Icons.copy,
                size: 14,
                color: cs.onSurface.withValues(alpha: 0.5),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildNumberedLines(List<String> lines) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: List.generate(lines.length, (i) {
        return Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            SizedBox(
              width: 24,
              child: Text(
                '${i + 1}',
                style: const TextStyle(
                  fontSize: 12,
                  fontFamily: 'monospace',
                  color: Color(0xFF858585),
                ),
                textAlign: TextAlign.right,
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Text(
                lines[i],
                style: const TextStyle(
                  fontSize: 12,
                  fontFamily: 'monospace',
                  color: Color(0xFFD4D4D4),
                ),
              ),
            ),
          ],
        );
      }),
    );
  }

  Widget _buildPlainCode(String code) {
    return Text(
      code,
      style: const TextStyle(
        fontSize: 12,
        fontFamily: 'monospace',
        color: Color(0xFFD4D4D4),
        height: 1.4,
      ),
    );
  }
}

void registerCodeComponent() {
  WidgetRegistry.register(ComponentType.code, (block, context, {onAction}) {
    return CodeComponent(block: block, onAction: onAction);
  });
}
