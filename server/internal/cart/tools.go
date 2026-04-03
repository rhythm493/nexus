package cart

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"

	"github.com/rhythm493/pocket-assistant/server/internal/quickcom"
)

// ToolDefinition matches the radio tools pattern.
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]any
}

// ToolHandlers implements cart tool execution.
type ToolHandlers struct {
	manager  *Manager
	quickcom *quickcom.Client
}

// NewToolHandlers creates a new set of cart tool handlers.
func NewToolHandlers(manager *Manager, qc *quickcom.Client) *ToolHandlers {
	return &ToolHandlers{manager: manager, quickcom: qc}
}

// IsCartTool checks if a tool name belongs to the cart system.
func IsCartTool(name string) bool {
	switch name {
	case "cart_add", "cart_remove", "cart_view", "cart_optimize", "cart_clear", "price_check", "find_deals":
		return true
	}
	return false
}

// ExecuteTool dispatches to the correct handler.
func (h *ToolHandlers) ExecuteTool(ctx context.Context, conversationID string, name string, args map[string]any) (any, error) {
	switch name {
	case "cart_add":
		return h.CartAdd(ctx, conversationID, args)
	case "cart_remove":
		return h.CartRemove(ctx, conversationID, args)
	case "cart_view":
		return h.CartView(ctx, conversationID, args)
	case "cart_optimize":
		return h.CartOptimize(ctx, conversationID, args)
	case "cart_clear":
		return h.CartClear(ctx, conversationID, args)
	case "price_check":
		return h.PriceCheck(ctx, conversationID, args)
	case "find_deals":
		return h.FindDeals(ctx, conversationID, args)
	default:
		return nil, fmt.Errorf("unknown cart tool: %s", name)
	}
}

// ToolDefinitions returns LLM-compatible tool schemas.
func (h *ToolHandlers) ToolDefinitions() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "cart_add",
			Description: "Add items to the shopping cart. Each item is a search query — be specific (e.g. 'amul butter 100g' not 'butter'). Returns matched product names for verification.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"items": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "List of specific search queries. Include type, brand, and size. E.g. [\"toned milk 1L\", \"brown eggs 6 pack\", \"whole wheat bread\"]",
					},
				},
				"required": []string{"items"},
			},
		},
		{
			Name:        "cart_remove",
			Description: "Remove an item from the cart by name.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"item": map[string]any{
						"type":        "string",
						"description": "Item name to remove (partial match supported)",
					},
				},
				"required": []string{"item"},
			},
		},
		{
			Name:        "cart_view",
			Description: "View the current cart contents with prices per provider and totals.",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "cart_optimize",
			Description: "Run price optimization on the cart. Compares ordering everything from one provider vs splitting across providers to find the cheapest option.",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "cart_clear",
			Description: "Clear all items from the cart and start fresh.",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "price_check",
			Description: "Quick price lookup without modifying the cart. Shows prices across all providers.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Product to search for",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "find_deals",
			Description: "Search for discounted items, optionally filtered by category.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"category": map[string]any{
						"type":        "string",
						"description": "Optional category like 'dairy', 'snacks', 'cleaning'",
					},
				},
			},
		},
	}
}

// --- Tool Handlers ---

func (h *ToolHandlers) CartAdd(ctx context.Context, convID string, args map[string]any) (any, error) {
	// Extract items — handle both []interface{} and single string
	var queries []string
	switch v := args["items"].(type) {
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				queries = append(queries, s)
			}
		}
	case string:
		queries = append(queries, v)
	default:
		return nil, fmt.Errorf("items must be a string array")
	}

	if len(queries) == 0 {
		return map[string]any{"success": false, "error": "no items provided"}, nil
	}
	if len(queries) > 10 {
		queries = queries[:10] // Cap at 10
	}

	cart := h.manager.GetOrCreateCart(convID)

	// Search items concurrently (max 3 at a time)
	type searchResult struct {
		query string
		resp  *quickcom.SearchResponse
		err   error
	}

	results := make([]searchResult, len(queries))
	sem := make(chan struct{}, 3)
	var wg sync.WaitGroup

	for i, query := range queries {
		wg.Add(1)
		go func(idx int, q string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			resp, err := h.quickcom.Search(ctx, quickcom.SearchRequest{Query: q, Limit: 5})
			results[idx] = searchResult{query: q, resp: resp, err: err}
		}(i, query)
	}
	wg.Wait()

	// Process results
	type addedItem struct {
		Query   string            `json:"query"`
		Matches map[string]string `json:"matches"` // provider → "Product Name ₹Price"
		Found   bool              `json:"found"`
	}

	added := make([]addedItem, 0, len(queries))

	for _, r := range results {
		if r.err != nil {
			slog.Warn("Cart search failed", "query", r.query, "error", r.err)
			added = append(added, addedItem{Query: r.query, Found: false})
			continue
		}

		item := CartItem{
			Query:      r.query,
			Selections: make(map[string]*CartProduct),
		}

		matches := map[string]string{}

		for provider, pr := range r.resp.Results {
			if len(pr.Products) == 0 {
				continue
			}
			best, ok := BestProduct(r.query, pr.Products)
			if !ok {
				continue
			}
			cp := productToCartProduct(best, provider)
			item.Selections[provider] = cp
			matches[provider] = compactMatch(best)
		}

		if len(item.Selections) > 0 {
			cart.AddItem(item)
			added = append(added, addedItem{Query: r.query, Matches: matches, Found: true})
		} else {
			added = append(added, addedItem{Query: r.query, Found: false})
		}
	}

	foundCount := 0
	for _, a := range added {
		if a.Found {
			foundCount++
		}
	}

	return map[string]any{
		"success":   true,
		"added":     foundCount,
		"total":     len(cart.Items),
		"items":     added,
		"cartState": cart.State,
	}, nil
}

func (h *ToolHandlers) CartRemove(ctx context.Context, convID string, args map[string]any) (any, error) {
	item, _ := args["item"].(string)
	if item == "" {
		return map[string]any{"success": false, "error": "item name required"}, nil
	}

	cart := h.manager.GetCart(convID)
	if cart == nil {
		return map[string]any{"success": false, "error": "no cart exists"}, nil
	}

	removed := cart.RemoveItem(item)
	return map[string]any{
		"success":   removed,
		"remaining": len(cart.Items),
		"cartState": cart.State,
	}, nil
}

func (h *ToolHandlers) CartView(ctx context.Context, convID string, args map[string]any) (any, error) {
	cart := h.manager.GetCart(convID)
	if cart == nil {
		return map[string]any{"state": CartIdle, "itemCount": 0, "items": []any{}}, nil
	}
	return cart.Summary(), nil
}

func (h *ToolHandlers) CartOptimize(ctx context.Context, convID string, args map[string]any) (any, error) {
	cart := h.manager.GetCart(convID)
	if cart == nil || len(cart.Items) == 0 {
		return map[string]any{"success": false, "error": "cart is empty"}, nil
	}

	result := Optimize(cart.Items)
	cart.SetOptimization(result)

	return map[string]any{
		"success":      true,
		"optimization": result,
		"cartState":    cart.State,
	}, nil
}

func (h *ToolHandlers) CartClear(ctx context.Context, convID string, args map[string]any) (any, error) {
	cart := h.manager.GetCart(convID)
	if cart == nil {
		return map[string]any{"success": true, "message": "no cart to clear"}, nil
	}
	cart.Clear()
	return map[string]any{"success": true, "cartState": cart.State}, nil
}

func (h *ToolHandlers) PriceCheck(ctx context.Context, convID string, args map[string]any) (any, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	resp, err := h.quickcom.Search(ctx, quickcom.SearchRequest{Query: query, Limit: 3})
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Compact format: provider → top products
	type priceEntry struct {
		Name     string  `json:"name"`
		Price    string  `json:"price"`
		Quantity string  `json:"quantity"`
		Discount *string `json:"discount,omitempty"`
	}

	prices := map[string][]priceEntry{}
	for provider, pr := range resp.Results {
		for _, p := range pr.Products[:min(len(pr.Products), 3)] {
			prices[provider] = append(prices[provider], priceEntry{
				Name: p.Name, Price: p.PriceDisplay,
				Quantity: p.Quantity, Discount: p.Discount,
			})
		}
	}

	return map[string]any{"query": query, "prices": prices}, nil
}

func (h *ToolHandlers) FindDeals(ctx context.Context, convID string, args map[string]any) (any, error) {
	category, _ := args["category"].(string)
	query := category
	if query == "" {
		query = "deals"
	}

	resp, err := h.quickcom.Search(ctx, quickcom.SearchRequest{Query: query, Limit: 10})
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	type dealEntry struct {
		Name        string `json:"name"`
		Price       string `json:"price"`
		Discount    string `json:"discount"`
		DiscountPct int    `json:"discountPct"`
		Provider    string `json:"provider"`
		Quantity    string `json:"quantity"`
	}

	var deals []dealEntry
	for provider, pr := range resp.Results {
		for _, p := range pr.Products {
			if p.DiscountPct != nil && *p.DiscountPct > 0 {
				deals = append(deals, dealEntry{
					Name: p.Name, Price: p.PriceDisplay,
					Discount: fmt.Sprintf("%d%% OFF", *p.DiscountPct),
					DiscountPct: *p.DiscountPct,
					Provider: provider, Quantity: p.Quantity,
				})
			}
		}
	}

	// Sort by discount pct descending — show best deals first
	sort.Slice(deals, func(i, j int) bool {
		return deals[i].DiscountPct > deals[j].DiscountPct
	})
	if len(deals) > 10 {
		deals = deals[:10]
	}

	return map[string]any{
		"category": category,
		"deals":    deals,
		"count":    len(deals),
	}, nil
}

// --- Helpers ---

func productToCartProduct(p quickcom.Product, provider string) *CartProduct {
	return &CartProduct{
		ProductID:         p.ID,
		Name:              p.Name,
		Brand:             p.Brand,
		PricePaise:        p.PricePaise,
		PriceDisplay:      p.PriceDisplay,
		Quantity:          p.Quantity,
		QuantityValue:     p.QuantityValue,
		QuantityUnit:      p.QuantityUnit,
		PerUnitPricePaise: p.PerUnitPricePaise,
		DiscountPct:       p.DiscountPct,
		ProductURL:        p.ProductURL,
		ImageURL:          p.ImageURL,
		Available:         p.Available,
		Provider:          provider,
	}
}

func compactMatch(p quickcom.Product) string {
	name := p.Name
	if len(name) > 40 {
		name = name[:37] + "..."
	}
	return name + " " + p.PriceDisplay
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetToolHandler returns the handler function for a tool (for external use).
func (h *ToolHandlers) GetToolHandler(name string) func(context.Context, string, map[string]any) (any, error) {
	switch name {
	case "cart_add":
		return h.CartAdd
	case "cart_remove":
		return h.CartRemove
	case "cart_view":
		return h.CartView
	case "cart_optimize":
		return h.CartOptimize
	case "cart_clear":
		return h.CartClear
	case "price_check":
		return h.PriceCheck
	case "find_deals":
		return h.FindDeals
	default:
		return nil
	}
}

// FormatCartSummaryForLLM creates a concise text summary for the LLM context.
func FormatCartSummaryForLLM(s CartSummary) string {
	if s.ItemCount == 0 {
		return "Cart is empty."
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Cart (%s): %d items\n", s.State, s.ItemCount)

	for _, item := range s.Items {
		fmt.Fprintf(&b, "- %s: ", item.Query)
		parts := []string{}
		for provider, price := range item.Prices {
			marker := ""
			if provider == item.BestProvider {
				marker = " ✓"
			}
			parts = append(parts, fmt.Sprintf("%s %s%s", provider, price, marker))
		}
		fmt.Fprintf(&b, "%s\n", strings.Join(parts, " | "))
	}

	if s.Optimization != nil {
		fmt.Fprintf(&b, "\nOptimization: %s", s.Optimization.Recommendation)
		if s.Optimization.SplitSavings > 0 {
			fmt.Fprintf(&b, " (saves ₹%d)", s.Optimization.SplitSavings/100)
		}
	}

	return b.String()
}
