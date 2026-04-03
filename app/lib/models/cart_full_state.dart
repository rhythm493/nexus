// Full cart state with product details — mirrors Go Cart.FullDetail() response.

import 'cart_state.dart';

class CartFullState {
  final String id;
  final String conversationId;
  final String state;
  final int itemCount;
  final List<CartFullItem> items;
  final OptimizationResult? optimization;

  CartFullState({
    required this.id,
    required this.conversationId,
    required this.state,
    required this.itemCount,
    required this.items,
    this.optimization,
  });

  factory CartFullState.fromJson(Map<String, dynamic> json) {
    return CartFullState(
      id: json['id'] as String? ?? '',
      conversationId: json['conversationId'] as String? ?? '',
      state: json['state'] as String? ?? 'idle',
      itemCount: json['itemCount'] as int? ?? 0,
      items: (json['items'] as List?)
              ?.map((e) => CartFullItem.fromJson(e as Map<String, dynamic>))
              .toList() ??
          [],
      optimization: json['optimization'] != null
          ? OptimizationResult.fromJson(json['optimization'] as Map<String, dynamic>)
          : null,
    );
  }

  bool get isEmpty => itemCount == 0;
  bool get isOptimized => state == 'optimized';
  bool get isBuilding => state == 'building';

  int get cheapestTotal {
    int cheapest = 0;
    for (final item in items) {
      final best = item.bestProduct;
      if (best != null) cheapest += best.pricePaise;
    }
    return cheapest;
  }
}

class CartFullItem {
  final String query;
  final Map<String, CartProduct> selections;
  final String preferredProvider;

  CartFullItem({
    required this.query,
    required this.selections,
    required this.preferredProvider,
  });

  factory CartFullItem.fromJson(Map<String, dynamic> json) {
    final selections = <String, CartProduct>{};
    if (json['selections'] is Map) {
      (json['selections'] as Map).forEach((k, v) {
        if (v is Map<String, dynamic>) {
          selections[k.toString()] = CartProduct.fromJson(v);
        }
      });
    }

    return CartFullItem(
      query: json['query'] as String? ?? '',
      selections: selections,
      preferredProvider: json['preferredProvider'] as String? ?? '',
    );
  }

  /// Best product: preferred provider first, then cheapest available.
  CartProduct? get bestProduct {
    if (preferredProvider.isNotEmpty && selections.containsKey(preferredProvider)) {
      final p = selections[preferredProvider]!;
      if (p.available) return p;
    }
    CartProduct? best;
    for (final p in selections.values) {
      if (!p.available) continue;
      if (best == null || p.pricePaise < best.pricePaise) best = p;
    }
    return best;
  }

  /// Sorted providers: preferred first, then cheapest.
  List<String> get sortedProviders {
    final providers = selections.keys.toList();
    providers.sort((a, b) {
      if (a == preferredProvider) return -1;
      if (b == preferredProvider) return 1;
      final pa = selections[a]!.pricePaise;
      final pb = selections[b]!.pricePaise;
      return pa.compareTo(pb);
    });
    return providers;
  }

  /// Current provider (preferred or cheapest).
  String get currentProvider => bestProduct?.provider ?? '';
}

class CartProduct {
  final String productId;
  final String name;
  final String? brand;
  final int pricePaise;
  final String priceDisplay;
  final String quantity;
  final double? quantityValue;
  final String? quantityUnit;
  final int? perUnitPricePaise;
  final int? discountPct;
  final String? productUrl;
  final String imageUrl;
  final bool available;
  final String provider;

  CartProduct({
    required this.productId,
    required this.name,
    this.brand,
    required this.pricePaise,
    required this.priceDisplay,
    required this.quantity,
    this.quantityValue,
    this.quantityUnit,
    this.perUnitPricePaise,
    this.discountPct,
    this.productUrl,
    required this.imageUrl,
    required this.available,
    required this.provider,
  });

  factory CartProduct.fromJson(Map<String, dynamic> json) {
    return CartProduct(
      productId: json['productId'] as String? ?? json['ProductID'] as String? ?? '',
      name: json['name'] as String? ?? json['Name'] as String? ?? '',
      brand: json['brand'] as String? ?? json['Brand'] as String?,
      pricePaise: json['pricePaise'] as int? ?? json['PricePaise'] as int? ?? 0,
      priceDisplay: json['priceDisplay'] as String? ?? json['PriceDisplay'] as String? ?? '',
      quantity: json['quantity'] as String? ?? json['Quantity'] as String? ?? '',
      quantityValue: (json['quantityValue'] as num?)?.toDouble(),
      quantityUnit: json['quantityUnit'] as String?,
      perUnitPricePaise: json['perUnitPricePaise'] as int?,
      discountPct: json['discountPct'] as int?,
      productUrl: json['productUrl'] as String?,
      imageUrl: json['imageUrl'] as String? ?? json['ImageURL'] as String? ?? '',
      available: json['available'] as bool? ?? json['Available'] as bool? ?? false,
      provider: json['provider'] as String? ?? json['Provider'] as String? ?? '',
    );
  }

  static String formatPaise(int paise) {
    final rupees = paise / 100;
    if (rupees == rupees.roundToDouble()) return '\u20b9${rupees.toInt()}';
    return '\u20b9${rupees.toStringAsFixed(2)}';
  }
}
