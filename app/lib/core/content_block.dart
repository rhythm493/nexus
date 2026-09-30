/// Types of content blocks that can be rendered in the conversation stream.
enum ComponentType {
  // Display
  markdown,
  card,
  metric,
  table,
  chart,
  image,
  code,
  list,
  grid,
  progress,
  badge,
  chip,
  divider,
  text,

  // Layout
  row,
  column,
  expanded,
  wrap,
  stack,
  scroll,
  container,
  spacer,

  // Interactive
  button,
  form,
  toggle,
  slider,
  dropdown,
  link,

  // Tool results
  searchResults,
  toolResult,
}

/// A structured content block that can be rendered natively by the WidgetRegistry.
///
/// Each block has a [type] that maps to a registered component builder,
/// [data] containing component-specific properties, optional [children]
/// for nesting, and optional [action]/[actionArgs] for interactive components.
class ContentBlock {
  final String id;
  final ComponentType type;
  final Map<String, dynamic> data;
  final List<ContentBlock> children;
  final String? action;
  final Map<String, dynamic>? actionArgs;

  const ContentBlock({
    required this.id,
    required this.type,
    this.data = const {},
    this.children = const [],
    this.action,
    this.actionArgs,
  });

  factory ContentBlock.fromJson(Map<String, dynamic> json) {
    return ContentBlock(
      id: json['id'] as String? ?? '',
      type: _parseType(json['type'] as String? ?? 'text'),
      data: (json['data'] as Map<String, dynamic>?) ?? {},
      children: (json['children'] as List<dynamic>?)
              ?.map((e) => ContentBlock.fromJson(e as Map<String, dynamic>))
              .toList() ??
          [],
      action: json['action'] as String?,
      actionArgs: json['actionArgs'] as Map<String, dynamic>?,
    );
  }

  Map<String, dynamic> toJson() => {
        'id': id,
        'type': type.name,
        'data': data,
        if (children.isNotEmpty) 'children': children.map((c) => c.toJson()).toList(),
        if (action != null) 'action': action,
        if (actionArgs != null) 'actionArgs': actionArgs,
      };

  ContentBlock copyWith({
    String? id,
    ComponentType? type,
    Map<String, dynamic>? data,
    List<ContentBlock>? children,
    String? action,
    Map<String, dynamic>? actionArgs,
    bool clearAction = false,
  }) {
    return ContentBlock(
      id: id ?? this.id,
      type: type ?? this.type,
      data: data ?? this.data,
      children: children ?? this.children,
      action: clearAction ? null : (action ?? this.action),
      actionArgs: clearAction ? null : (actionArgs ?? this.actionArgs),
    );
  }

  static ComponentType _parseType(String name) {
    return ComponentType.values.firstWhere(
      (t) => t.name == name,
      orElse: () => ComponentType.text,
    );
  }

  @override
  String toString() => 'ContentBlock(${type.name}: $id)';
}

/// Creates a [ContentBlock] for markdown-rendered text.
ContentBlock textBlock(String id, String text) => ContentBlock(
      id: id,
      type: ComponentType.markdown,
      data: {'text': text},
    );

/// Creates a [ContentBlock] for a system message.
ContentBlock systemBlock(String id, String text) => ContentBlock(
      id: id,
      type: ComponentType.text,
      data: {'text': text},
    );

/// Creates a [ContentBlock] for an error message.
ContentBlock errorBlock(String id, String text) => ContentBlock(
      id: id,
      type: ComponentType.card,
      data: {
        'title': 'Error',
        'content': [
          {
            'id': '${id}_text',
            'type': 'text',
            'data': {'text': text},
          }
        ],
      },
    );
