import 'package:flutter/material.dart';

import 'package:nexus/core/content_block.dart';
import 'package:nexus/core/widget_registry.dart';

class TableComponent extends StatelessWidget {
  final ContentBlock block;
  final void Function(String action, Map<String, dynamic>? args)? onAction;

  const TableComponent({
    super.key,
    required this.block,
    this.onAction,
  });

  @override
  Widget build(BuildContext context) {
    final headers = block.data['headers'] as List<dynamic>?;
    final rows = block.data['rows'] as List<dynamic>?;
    final caption = block.data['caption'] as String?;

    if (headers == null || rows == null || headers.isEmpty) {
      return const SizedBox(
        width: double.infinity,
        child: Text(
          '[Table] Missing data',
          style: TextStyle(color: Colors.grey, fontSize: 11),
        ),
      );
    }

    final cs = Theme.of(context).colorScheme;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        SingleChildScrollView(
          scrollDirection: Axis.horizontal,
          child: DataTable(
            headingRowColor:
                WidgetStateProperty.all(cs.surfaceContainerHighest),
            columns: headers.map((h) {
              return DataColumn(
                label: Text(
                  h.toString(),
                  style: TextStyle(
                    fontWeight: FontWeight.bold,
                    fontSize: 12,
                    color: cs.onSurface,
                  ),
                ),
              );
            }).toList(),
            rows: rows.map((row) {
              final cells = row as List<dynamic>;
              return DataRow(
                cells: cells.map((cell) {
                  return DataCell(
                    Text(
                      cell.toString(),
                      style: TextStyle(
                        fontSize: 12,
                        color: cs.onSurface.withValues(alpha: 0.87),
                      ),
                    ),
                  );
                }).toList(),
              );
            }).toList(),
          ),
        ),
        if (caption != null)
          Padding(
            padding: const EdgeInsets.only(top: 4, left: 16),
            child: Text(
              caption,
              style: TextStyle(
                fontSize: 11,
                fontStyle: FontStyle.italic,
                color: cs.onSurface.withValues(alpha: 0.5),
              ),
            ),
          ),
      ],
    );
  }
}

void registerTableComponent() {
  WidgetRegistry.register(ComponentType.table, (block, context, {onAction}) {
    return TableComponent(block: block, onAction: onAction);
  });
}
