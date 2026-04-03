package cart

import (
	"math"
	"sort"
	"time"
)

// ProviderCart is one provider's portion of the order.
type ProviderCart struct {
	Provider   string        `json:"provider"`
	Items      []CartProduct `json:"items"`
	TotalPaise int           `json:"totalPaise"`
	ItemCount  int           `json:"itemCount"`
}

// OptimizationResult is the output of the cart optimizer.
type OptimizationResult struct {
	SingleBest       ProviderCart   `json:"singleBest"`
	SplitCarts       []ProviderCart `json:"splitCarts"`
	SplitTotalPaise  int            `json:"splitTotalPaise"`
	SplitSavings     int            `json:"splitSavings"`     // paise saved vs single best
	SplitSavingsPct  float64        `json:"splitSavingsPct"`
	Recommendation   string         `json:"recommendation"` // "single" or "split"
	UnavailableItems []string       `json:"unavailableItems,omitempty"`
	Timestamp        time.Time      `json:"timestamp"`
}

// Optimize finds the cheapest way to fulfill the cart items.
// Pure function — no side effects.
func Optimize(items []CartItem) *OptimizationResult {
	if len(items) == 0 {
		return &OptimizationResult{Recommendation: "single", Timestamp: time.Now()}
	}

	// Collect all providers
	providerSet := map[string]bool{}
	for _, item := range items {
		for provider := range item.Selections {
			providerSet[provider] = true
		}
	}

	providers := make([]string, 0, len(providerSet))
	for p := range providerSet {
		providers = append(providers, p)
	}
	sort.Strings(providers)

	// --- Single-provider evaluation ---
	type providerScore struct {
		provider string
		total    int
		gaps     int // items not available on this provider
		cart     ProviderCart
	}

	var scores []providerScore
	for _, provider := range providers {
		pc := ProviderCart{Provider: provider}
		gaps := 0
		for _, item := range items {
			sel, ok := item.Selections[provider]
			if !ok || !sel.Available {
				gaps++
				continue
			}
			pc.Items = append(pc.Items, *sel)
			pc.TotalPaise += sel.PricePaise
			pc.ItemCount++
		}
		scores = append(scores, providerScore{
			provider: provider,
			total:    pc.TotalPaise,
			gaps:     gaps,
			cart:     pc,
		})
	}

	// Sort: fewest gaps first, then cheapest
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].gaps != scores[j].gaps {
			return scores[i].gaps < scores[j].gaps
		}
		return scores[i].total < scores[j].total
	})

	if len(scores) == 0 {
		var unavail []string
		for _, item := range items {
			unavail = append(unavail, item.Query)
		}
		return &OptimizationResult{
			Recommendation:   "single",
			UnavailableItems: unavail,
			Timestamp:        time.Now(),
		}
	}

	singleBest := scores[0].cart

	// --- Split optimization ---
	splitMap := map[string]*ProviderCart{}
	var unavailable []string

	for _, item := range items {
		var cheapest *CartProduct
		var cheapestProvider string

		// Respect user preference
		if item.PreferredProvider != "" {
			if sel, ok := item.Selections[item.PreferredProvider]; ok && sel.Available {
				cheapest = sel
				cheapestProvider = item.PreferredProvider
			}
		}

		// Find cheapest across all providers
		if cheapest == nil {
			for provider, sel := range item.Selections {
				if !sel.Available {
					continue
				}
				if cheapest == nil || sel.PricePaise < cheapest.PricePaise {
					cheapest = sel
					cheapestProvider = provider
				}
			}
		}

		if cheapest == nil {
			unavailable = append(unavailable, item.Query)
			continue
		}

		if _, ok := splitMap[cheapestProvider]; !ok {
			splitMap[cheapestProvider] = &ProviderCart{Provider: cheapestProvider}
		}
		pc := splitMap[cheapestProvider]
		pc.Items = append(pc.Items, *cheapest)
		pc.TotalPaise += cheapest.PricePaise
		pc.ItemCount++
	}

	splitCarts := make([]ProviderCart, 0, len(splitMap))
	splitTotal := 0
	for _, pc := range splitMap {
		splitCarts = append(splitCarts, *pc)
		splitTotal += pc.TotalPaise
	}
	// Sort by total descending for display
	sort.Slice(splitCarts, func(i, j int) bool {
		return splitCarts[i].TotalPaise > splitCarts[j].TotalPaise
	})

	// --- Compare ---
	savings := singleBest.TotalPaise - splitTotal
	if savings < 0 {
		savings = 0
	}
	var savingsPct float64
	if singleBest.TotalPaise > 0 {
		savingsPct = math.Round(float64(savings) / float64(singleBest.TotalPaise) * 100)
	}

	recommendation := "single"
	if savings > 0 && savingsPct > 5 && len(splitCarts) > 1 {
		recommendation = "split"
	}

	return &OptimizationResult{
		SingleBest:       singleBest,
		SplitCarts:       splitCarts,
		SplitTotalPaise:  splitTotal,
		SplitSavings:     savings,
		SplitSavingsPct:  savingsPct,
		Recommendation:   recommendation,
		UnavailableItems: unavailable,
		Timestamp:        time.Now(),
	}
}
