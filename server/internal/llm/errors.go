package llm

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// RateLimitError is returned when the LLM API responds with HTTP 429.
type RateLimitError struct {
	RetryAfter time.Duration
	Message    string
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limited: %s (retry after %s)", e.Message, e.RetryAfter.Round(time.Second))
}

// parseRetryAfter parses the Retry-After header value.
// Handles integer seconds (Groq) and HTTP-date (RFC 7231).
// Returns 60s as default if parsing fails.
func parseRetryAfter(header string) time.Duration {
	if header == "" {
		return 60 * time.Second
	}
	// Try integer seconds first (most common: Groq sends "8.988s" or just "9")
	if secs, err := strconv.ParseFloat(header, 64); err == nil {
		return time.Duration(secs * float64(time.Second))
	}
	// Try HTTP-date format
	if t, err := http.ParseTime(header); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}
	return 60 * time.Second
}
