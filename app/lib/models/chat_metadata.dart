class ChatMetadata {
  final String? model;
  final String? enrichedName;
  final String? finishReason;
  final int? promptTokens;
  final int? completionTokens;
  final int? totalTokens;
  final int? latencyMs;
  final String? promptPrice;
  final String? completionPrice;
  final String? estimatedCost;

  ChatMetadata({
    this.model,
    this.enrichedName,
    this.finishReason,
    this.promptTokens,
    this.completionTokens,
    this.totalTokens,
    this.latencyMs,
    this.promptPrice,
    this.completionPrice,
    this.estimatedCost,
  });

  bool get isTruncated => finishReason == 'length';

  String get displayLabel {
    if (enrichedName != null && enrichedName!.isNotEmpty) {
      return enrichedName!;
    }
    return model ?? 'Unknown';
  }

  String? get tokenSummary {
    if (totalTokens == null || totalTokens == 0) return null;
    final parts = <String>[];
    if (promptTokens != null) parts.add('$promptTokens in');
    if (completionTokens != null) parts.add('$completionTokens out');
    return parts.join(' · ');
  }

  String? get costSummary {
    if (estimatedCost != null && estimatedCost!.isNotEmpty) {
      final cost = double.tryParse(estimatedCost!);
      if (cost != null && cost > 0) {
        if (cost < 0.001) return '<\$0.001';
        return '\$${cost.toStringAsFixed(4)}';
      }
    }
    return null;
  }

  String? get latencySummary {
    if (latencyMs == null) return null;
    if (latencyMs! >= 1000) {
      return '${(latencyMs! / 1000).toStringAsFixed(1)}s';
    }
    return '${latencyMs}ms';
  }

  String? get finishReasonLabel {
    switch (finishReason) {
      case 'stop':
        return null;
      case 'length':
        return 'Truncated';
      case 'content_filter':
        return 'Blocked by filter';
      default:
        return finishReason;
    }
  }

  factory ChatMetadata.fromJson(Map<String, dynamic> json) {
    return ChatMetadata(
      model: json['model'] as String?,
      enrichedName: json['enriched_name'] as String?,
      finishReason: json['finish_reason'] as String?,
      promptTokens: json['prompt_tokens'] as int?,
      completionTokens: json['completion_tokens'] as int?,
      totalTokens: json['total_tokens'] as int?,
      latencyMs: json['latency_ms'] as int?,
      promptPrice: json['prompt_price'] as String?,
      completionPrice: json['completion_price'] as String?,
      estimatedCost: json['estimated_cost'] as String?,
    );
  }
}
