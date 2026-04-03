// Cart state models matching Go server's JSON response.

class CartState {
  final String state;
  final int itemCount;
  final List<CartItemSummary> items;
  final Map<String, int> totalsByProvider;
  final OptimizationResult? optimization;

  CartState({
    required this.state,
    required this.itemCount,
    required this.items,
    required this.totalsByProvider,
    this.optimization,
  });

  factory CartState.fromJson(Map<String, dynamic> json) {
    final totals = <String, int>{};
    if (json['totalsByProvider'] is Map) {
      (json['totalsByProvider'] as Map).forEach((k, v) {
        totals[k.toString()] = (v is int) ? v : (v as num).toInt();
      });
    }

    return CartState(
      state: json['state'] as String? ?? 'idle',
      itemCount: json['itemCount'] as int? ?? 0,
      items: (json['items'] as List?)
              ?.map((e) => CartItemSummary.fromJson(e as Map<String, dynamic>))
              .toList() ??
          [],
      totalsByProvider: totals,
      optimization: json['optimization'] != null
          ? OptimizationResult.fromJson(json['optimization'] as Map<String, dynamic>)
          : null,
    );
  }

  bool get isEmpty => itemCount == 0;
  bool get isOptimized => state == 'optimized';
  bool get isBuilding => state == 'building';

  /// Cheapest single-provider total in paise.
  int get cheapestTotal {
    if (totalsByProvider.isEmpty) return 0;
    return totalsByProvider.values.reduce((a, b) => a < b ? a : b);
  }

  /// Format paise to rupees string.
  static String formatPaise(int paise) {
    final rupees = paise / 100;
    if (rupees == rupees.roundToDouble()) return '\u20b9${rupees.toInt()}';
    return '\u20b9${rupees.toStringAsFixed(2)}';
  }
}

class CartItemSummary {
  final String query;
  final int bestPrice;
  final String bestProvider;
  final Map<String, String> prices;

  CartItemSummary({
    required this.query,
    required this.bestPrice,
    required this.bestProvider,
    required this.prices,
  });

  factory CartItemSummary.fromJson(Map<String, dynamic> json) {
    final prices = <String, String>{};
    if (json['prices'] is Map) {
      (json['prices'] as Map).forEach((k, v) {
        prices[k.toString()] = v.toString();
      });
    }

    return CartItemSummary(
      query: json['query'] as String? ?? '',
      bestPrice: json['bestPrice'] as int? ?? 0,
      bestProvider: json['bestProvider'] as String? ?? '',
      prices: prices,
    );
  }

  /// List of providers sorted with best first.
  List<String> get sortedProviders {
    final providers = prices.keys.toList();
    providers.sort((a, b) {
      if (a == bestProvider) return -1;
      if (b == bestProvider) return 1;
      return a.compareTo(b);
    });
    return providers;
  }
}

class OptimizationResult {
  final ProviderCart singleBest;
  final List<ProviderCart> splitCarts;
  final int splitTotalPaise;
  final int splitSavings;
  final double splitSavingsPct;
  final String recommendation;
  final List<String> unavailableItems;

  OptimizationResult({
    required this.singleBest,
    required this.splitCarts,
    required this.splitTotalPaise,
    required this.splitSavings,
    required this.splitSavingsPct,
    required this.recommendation,
    required this.unavailableItems,
  });

  factory OptimizationResult.fromJson(Map<String, dynamic> json) {
    return OptimizationResult(
      singleBest: ProviderCart.fromJson(
          json['singleBest'] as Map<String, dynamic>? ?? {}),
      splitCarts: (json['splitCarts'] as List?)
              ?.map((e) => ProviderCart.fromJson(e as Map<String, dynamic>))
              .toList() ??
          [],
      splitTotalPaise: json['splitTotalPaise'] as int? ?? 0,
      splitSavings: json['splitSavings'] as int? ?? 0,
      splitSavingsPct: (json['splitSavingsPct'] as num?)?.toDouble() ?? 0,
      recommendation: json['recommendation'] as String? ?? 'single',
      unavailableItems: (json['unavailableItems'] as List?)
              ?.map((e) => e.toString())
              .toList() ??
          [],
    );
  }

  bool get isSplit => recommendation == 'split';
}

class ProviderCart {
  final String provider;
  final int totalPaise;
  final int itemCount;

  ProviderCart({
    required this.provider,
    required this.totalPaise,
    required this.itemCount,
  });

  factory ProviderCart.fromJson(Map<String, dynamic> json) {
    return ProviderCart(
      provider: json['provider'] as String? ?? '',
      totalPaise: json['totalPaise'] as int? ?? 0,
      itemCount: json['itemCount'] as int? ?? 0,
    );
  }
}
