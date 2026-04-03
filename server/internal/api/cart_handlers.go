package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/rhythm493/pocket-assistant/server/internal/cart"
	"github.com/rhythm493/pocket-assistant/server/internal/quickcom"
)

// handleGetCart returns the compact cart summary.
func (s *Server) handleGetCart(w http.ResponseWriter, r *http.Request) {
	convID := r.PathValue("convId")
	if convID == "" {
		http.Error(w, `{"error":"conversation ID required"}`, http.StatusBadRequest)
		return
	}

	c := s.cartManager.GetCart(convID)
	if c == nil {
		http.Error(w, `{"error":"no cart found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(c.Summary())
}

// handleGetCartFull returns the full cart with all product details (images, brands, etc).
func (s *Server) handleGetCartFull(w http.ResponseWriter, r *http.Request) {
	convID := r.PathValue("convId")
	if convID == "" {
		http.Error(w, `{"error":"conversation ID required"}`, http.StatusBadRequest)
		return
	}

	c := s.cartManager.GetCart(convID)
	if c == nil {
		http.Error(w, `{"error":"no cart found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(c.FullDetail())
}

// handleSwapCartItem swaps an item's preferred provider.
func (s *Server) handleSwapCartItem(w http.ResponseWriter, r *http.Request) {
	convID := r.PathValue("convId")
	if convID == "" {
		http.Error(w, `{"error":"conversation ID required"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		Query    string `json:"query"`
		Provider string `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	c := s.cartManager.GetCart(convID)
	if c == nil {
		http.Error(w, `{"error":"no cart found"}`, http.StatusNotFound)
		return
	}

	ok := c.SwapItem(req.Query, req.Provider, nil)
	if !ok {
		http.Error(w, `{"error":"item not found in cart"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"cart":    c.FullDetail(),
	})
}

// handleAddCartItem adds a product to the cart from inline search (bypasses LLM).
func (s *Server) handleAddCartItem(w http.ResponseWriter, r *http.Request) {
	convID := r.PathValue("convId")
	if convID == "" {
		http.Error(w, `{"error":"conversation ID required"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		Query    string `json:"query"`
		Provider string `json:"provider,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Query == "" {
		http.Error(w, `{"error":"query is required"}`, http.StatusBadRequest)
		return
	}

	if s.quickcomClient == nil {
		http.Error(w, `{"error":"search not available"}`, http.StatusServiceUnavailable)
		return
	}

	// Search QuickCom for the product
	ctx, cancel := context.WithTimeout(r.Context(), 30*secondTimeout)
	defer cancel()

	resp, err := s.quickcomClient.Search(ctx, quickcom.SearchRequest{Query: req.Query, Limit: 5})
	if err != nil {
		http.Error(w, `{"error":"search failed"}`, http.StatusBadGateway)
		return
	}

	// Build cart item with best match per provider
	item := cart.CartItem{
		Query:      req.Query,
		Selections: make(map[string]*cart.CartProduct),
	}

	for provider, pr := range resp.Results {
		best, ok := cart.BestProduct(req.Query, pr.Products)
		if !ok {
			continue
		}
		item.Selections[provider] = productToCartProduct(best, provider)
	}

	if len(item.Selections) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "no products found"})
		return
	}

	// Set preferred provider if specified
	if req.Provider != "" {
		item.PreferredProvider = req.Provider
	}

	c := s.cartManager.GetOrCreateCart(convID)
	c.AddItem(item)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"cart":    c.FullDetail(),
	})
}

// handleRemoveCartItem removes an item from the cart.
func (s *Server) handleRemoveCartItem(w http.ResponseWriter, r *http.Request) {
	convID := r.PathValue("convId")
	query := r.PathValue("query")
	if convID == "" || query == "" {
		http.Error(w, `{"error":"conversation ID and query required"}`, http.StatusBadRequest)
		return
	}

	c := s.cartManager.GetCart(convID)
	if c == nil {
		http.Error(w, `{"error":"no cart found"}`, http.StatusNotFound)
		return
	}

	ok := c.RemoveItem(query)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": ok,
		"cart":    c.FullDetail(),
	})
}

// handleSearch proxies search to QuickCom for the inline search UI.
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, `{"error":"q parameter required"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*secondTimeout)
	defer cancel()

	resp, err := s.quickcomClient.Search(ctx, quickcom.SearchRequest{Query: query, Limit: 10})
	if err != nil {
		http.Error(w, `{"error":"search failed"}`, http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// productToCartProduct converts a QuickCom product to a CartProduct.
func productToCartProduct(p quickcom.Product, provider string) *cart.CartProduct {
	return &cart.CartProduct{
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

const secondTimeout = 1_000_000_000 // 1 second in nanoseconds for context.WithTimeout
