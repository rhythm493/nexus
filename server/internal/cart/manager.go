package cart

import (
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CartState represents the cart's lifecycle state.
type CartState string

const (
	CartIdle      CartState = "idle"
	CartBuilding  CartState = "building"
	CartReview    CartState = "review"
	CartOptimized CartState = "optimized"
	CartApproved  CartState = "approved"
)

// CartEventType identifies what happened in a cart event.
type CartEventType string

const (
	EventItemAdded   CartEventType = "item_added"
	EventItemRemoved CartEventType = "item_removed"
	EventItemSwapped CartEventType = "item_swapped"
	EventOptimized   CartEventType = "optimized"
	EventApproved    CartEventType = "approved"
	EventCleared     CartEventType = "cleared"
)

// CartEvent is an append-only log entry.
type CartEvent struct {
	Type      CartEventType          `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// CartProduct holds pricing info for one product from one provider.
type CartProduct struct {
	ProductID         string   `json:"productId"`
	Name              string   `json:"name"`
	Brand             *string  `json:"brand,omitempty"`
	PricePaise        int      `json:"pricePaise"`
	PriceDisplay      string   `json:"priceDisplay"`
	Quantity          string   `json:"quantity"`
	QuantityValue     *float64 `json:"quantityValue,omitempty"`
	QuantityUnit      *string  `json:"quantityUnit,omitempty"`
	PerUnitPricePaise *int     `json:"perUnitPricePaise,omitempty"`
	DiscountPct       *int     `json:"discountPct,omitempty"`
	ProductURL        *string  `json:"productUrl,omitempty"`
	ImageURL          string   `json:"imageUrl"`
	Available         bool     `json:"available"`
	Provider          string   `json:"provider"`
}

// CartItem represents one item the user wants, with options from each provider.
type CartItem struct {
	Query             string                  `json:"query"`
	Selections        map[string]*CartProduct `json:"selections"`        // provider → best match
	PreferredProvider string                  `json:"preferredProvider"` // user override (empty = auto)
}

// BestSelection returns the cheapest available product, respecting user preference.
func (ci *CartItem) BestSelection() *CartProduct {
	// User preference first
	if ci.PreferredProvider != "" {
		if p, ok := ci.Selections[ci.PreferredProvider]; ok && p.Available {
			return p
		}
	}
	// Cheapest available
	var best *CartProduct
	for _, p := range ci.Selections {
		if !p.Available {
			continue
		}
		if best == nil || p.PricePaise < best.PricePaise {
			best = p
		}
	}
	return best
}

// Cart is the per-conversation virtual shopping cart.
type Cart struct {
	ID             string              `json:"id"`
	ConversationID string              `json:"conversationId"`
	State          CartState           `json:"state"`
	Items          []CartItem          `json:"items"`
	Optimization   *OptimizationResult `json:"optimization,omitempty"`
	Events         []CartEvent         `json:"events"`
	CreatedAt      time.Time           `json:"createdAt"`
	UpdatedAt      time.Time           `json:"updatedAt"`
	mu             sync.Mutex
}

func (c *Cart) addEvent(typ CartEventType, data map[string]interface{}) {
	c.Events = append(c.Events, CartEvent{Type: typ, Timestamp: time.Now(), Data: data})
	c.UpdatedAt = time.Now()
}

// AddItem adds an item to the cart. Invalidates optimization if present.
func (c *Cart) AddItem(item CartItem) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if item already exists (by query, case-insensitive)
	for i, existing := range c.Items {
		if strings.EqualFold(existing.Query, item.Query) {
			c.Items[i] = item // Replace with updated selections
			c.addEvent(EventItemAdded, map[string]interface{}{"query": item.Query, "replaced": true})
			c.invalidateOptimization()
			return
		}
	}

	c.Items = append(c.Items, item)
	c.addEvent(EventItemAdded, map[string]interface{}{"query": item.Query})
	if c.State == CartIdle {
		c.State = CartBuilding
	}
	c.invalidateOptimization()
}

// RemoveItem removes an item by query (case-insensitive partial match).
func (c *Cart) RemoveItem(query string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	q := strings.ToLower(query)
	for i, item := range c.Items {
		if strings.EqualFold(item.Query, query) || strings.Contains(strings.ToLower(item.Query), q) {
			c.Items = append(c.Items[:i], c.Items[i+1:]...)
			c.addEvent(EventItemRemoved, map[string]interface{}{"query": item.Query})
			c.invalidateOptimization()
			return true
		}
	}
	return false
}

// SwapItem changes the preferred provider for an item and optionally sets a new product.
func (c *Cart) SwapItem(query string, provider string, product *CartProduct) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	q := strings.ToLower(query)
	for i, item := range c.Items {
		if strings.EqualFold(item.Query, query) || strings.Contains(strings.ToLower(item.Query), q) {
			if product != nil {
				c.Items[i].Selections[provider] = product
			}
			c.Items[i].PreferredProvider = provider
			c.addEvent(EventItemSwapped, map[string]interface{}{
				"query": item.Query, "provider": provider,
			})
			c.invalidateOptimization()
			return true
		}
	}
	return false
}

func (c *Cart) invalidateOptimization() {
	if c.State == CartOptimized {
		c.State = CartBuilding
		c.Optimization = nil
	}
}

// SetOptimization stores the optimization result and transitions state.
func (c *Cart) SetOptimization(result *OptimizationResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Optimization = result
	c.State = CartOptimized
	c.addEvent(EventOptimized, map[string]interface{}{
		"recommendation": result.Recommendation,
		"savings":        result.SplitSavings,
	})
}

// Approve transitions the cart to approved state.
func (c *Cart) Approve() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.State != CartOptimized {
		return false
	}
	c.State = CartApproved
	c.addEvent(EventApproved, nil)
	return true
}

// Clear resets the cart.
func (c *Cart) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Items = nil
	c.Optimization = nil
	c.State = CartIdle
	c.addEvent(EventCleared, nil)
}

// FullDetail returns the full cart state with all product details.
func (c *Cart) FullDetail() map[string]interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()

	items := make([]map[string]interface{}, 0, len(c.Items))
	for _, item := range c.Items {
		selections := make(map[string]interface{})
		for provider, p := range item.Selections {
			selections[provider] = p
		}
		items = append(items, map[string]interface{}{
			"query":             item.Query,
			"selections":        selections,
			"preferredProvider": item.PreferredProvider,
		})
	}

	result := map[string]interface{}{
		"id":             c.ID,
		"conversationId": c.ConversationID,
		"state":          c.State,
		"itemCount":      len(c.Items),
		"items":          items,
	}
	if c.Optimization != nil {
		result["optimization"] = c.Optimization
	}
	return result
}

// --- Summary types (compact representation for tool results / SSE) ---

type CartItemSummary struct {
	Query        string            `json:"query"`
	BestPrice    int               `json:"bestPrice"`
	BestProvider string            `json:"bestProvider"`
	Prices       map[string]string `json:"prices"` // provider → "₹75 (500ml)"
}

type CartSummary struct {
	State           CartState           `json:"state"`
	ItemCount       int                 `json:"itemCount"`
	Items           []CartItemSummary   `json:"items"`
	TotalsByProvider map[string]int     `json:"totalsByProvider"` // paise
	Optimization    *OptimizationResult `json:"optimization,omitempty"`
}

// Summary returns a compact representation of the cart.
func (c *Cart) Summary() CartSummary {
	c.mu.Lock()
	defer c.mu.Unlock()

	summary := CartSummary{
		State:            c.State,
		ItemCount:        len(c.Items),
		Items:            make([]CartItemSummary, 0, len(c.Items)),
		TotalsByProvider: make(map[string]int),
		Optimization:     c.Optimization,
	}

	for _, item := range c.Items {
		is := CartItemSummary{
			Query:  item.Query,
			Prices: make(map[string]string),
		}

		for provider, p := range item.Selections {
			if !p.Available {
				continue
			}
			is.Prices[provider] = p.PriceDisplay + " (" + p.Quantity + ")"
			summary.TotalsByProvider[provider] += p.PricePaise

			if is.BestPrice == 0 || p.PricePaise < is.BestPrice {
				is.BestPrice = p.PricePaise
				is.BestProvider = provider
			}
		}

		// If user has a preferred provider, use that as "best"
		if item.PreferredProvider != "" {
			if p, ok := item.Selections[item.PreferredProvider]; ok && p.Available {
				is.BestPrice = p.PricePaise
				is.BestProvider = item.PreferredProvider
			}
		}

		summary.Items = append(summary.Items, is)
	}

	return summary
}

// --- Manager ---

// Manager stores carts per conversation.
type Manager struct {
	carts sync.Map
}

// NewManager creates a new cart manager.
func NewManager() *Manager {
	return &Manager{}
}

// GetOrCreateCart returns the cart for a conversation, creating if needed.
func (m *Manager) GetOrCreateCart(conversationID string) *Cart {
	if c, ok := m.carts.Load(conversationID); ok {
		return c.(*Cart)
	}
	c := &Cart{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		State:          CartIdle,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	actual, _ := m.carts.LoadOrStore(conversationID, c)
	return actual.(*Cart)
}

// GetCart returns the cart for a conversation, or nil if none exists.
func (m *Manager) GetCart(conversationID string) *Cart {
	if c, ok := m.carts.Load(conversationID); ok {
		return c.(*Cart)
	}
	return nil
}

// DeleteCart removes a cart.
func (m *Manager) DeleteCart(conversationID string) {
	m.carts.Delete(conversationID)
}

// CleanupStaleCarts removes carts older than maxAge.
func (m *Manager) CleanupStaleCarts(maxAge time.Duration) {
	cutoff := time.Now().Add(-maxAge)
	m.carts.Range(func(key, value interface{}) bool {
		c := value.(*Cart)
		c.mu.Lock()
		stale := c.UpdatedAt.Before(cutoff)
		c.mu.Unlock()
		if stale {
			m.carts.Delete(key)
		}
		return true
	})
}
