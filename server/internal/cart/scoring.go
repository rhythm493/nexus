package cart

import (
	"strings"
	"unicode"

	"github.com/rhythm493/pocket-assistant/server/internal/quickcom"
)

// ScoreProduct scores how well a product matches the search query.
// Higher score = better match. Factors: word overlap, brand match, rating, size reasonableness.
func ScoreProduct(query string, p quickcom.Product) float64 {
	score := 0.0

	queryLower := strings.ToLower(query)
	queryWords := tokenize(queryLower)
	nameLower := strings.ToLower(p.Name)
	nameWords := tokenize(nameLower)

	// 1. Word match (0–6.0)
	matchCount := 0
	for _, qw := range queryWords {
		for _, nw := range nameWords {
			if nw == qw || strings.HasPrefix(nw, qw) || strings.HasPrefix(qw, nw) {
				matchCount++
				break
			}
		}
	}
	if len(queryWords) > 0 {
		score += float64(matchCount) / float64(len(queryWords)) * 6.0
	}

	// Bonus for exact substring match
	if strings.Contains(nameLower, queryLower) {
		score += 1.0
	}

	// 2. Brand match (0–2.0)
	if p.Brand != nil && *p.Brand != "" {
		brandLower := strings.ToLower(*p.Brand)
		for _, qw := range queryWords {
			if strings.Contains(brandLower, qw) || strings.Contains(qw, brandLower) {
				score += 2.0
				break
			}
		}
	}

	// 3. Rating tiebreaker (0–1.0)
	if p.Rating != nil && *p.Rating > 0 {
		score += (*p.Rating / 5.0)
	}

	// 4. Oversize penalty (-1.0)
	if !containsQuantityHint(queryLower) && p.QuantityValue != nil && p.QuantityUnit != nil {
		if isOversized(p.QuantityUnit, *p.QuantityValue) {
			score -= 1.0
		}
	}

	return score
}

// BestProduct returns the highest-scoring available product for a query.
// Returns false if no available products exist.
func BestProduct(query string, products []quickcom.Product) (quickcom.Product, bool) {
	bestScore := -1.0
	bestIdx := -1

	for i, p := range products {
		if !p.Available {
			continue
		}
		s := ScoreProduct(query, p)
		if s > bestScore {
			bestScore = s
			bestIdx = i
		}
	}

	if bestIdx < 0 {
		return quickcom.Product{}, false
	}
	return products[bestIdx], true
}

// tokenize splits a string into lowercase words, stripping punctuation.
func tokenize(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// containsQuantityHint returns true if the query mentions a quantity or unit.
func containsQuantityHint(query string) bool {
	units := []string{"kg", "gm", "g", "ml", "l", "ltr", "litre", "liter", "pack", "pcs", "piece", "dozen"}
	for _, u := range units {
		if strings.Contains(query, u) {
			return true
		}
	}
	// Check for any digit
	for _, r := range query {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

// isOversized returns true if a product quantity is unusually large for household grocery.
func isOversized(unit *string, value float64) bool {
	if unit == nil {
		return false
	}
	switch strings.ToLower(*unit) {
	case "kg":
		return value > 2
	case "g", "gm":
		return value > 2000
	case "l", "ltr", "litre", "liter":
		return value > 2
	case "ml":
		return value > 2000
	}
	return false
}
