package llm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"
)

// SlotConfig defines a provider+model combination for the rotating provider.
type SlotConfig struct {
	Provider string `yaml:"provider" json:"provider"`
	Model    string `yaml:"model" json:"model"`
}

// defaultSlots is the priority-ordered list of provider+model pairs used when
// no custom slots are supplied to NewRotatingProvider.
var defaultSlots = []SlotConfig{
	{Provider: "groq", Model: "meta-llama/llama-4-scout-17b-16e-instruct"},
	{Provider: "cerebras", Model: "llama3.1-8b"},
	{Provider: "groq", Model: "llama-3.3-70b-versatile"},
	{Provider: "groq", Model: "moonshotai/kimi-k2-instruct"},
	{Provider: "cerebras", Model: "qwen-3-235b-a22b-instruct-2507"},
	{Provider: "groq", Model: "openai/gpt-oss-120b"},
	{Provider: "groq", Model: "qwen/qwen3-32b"},
	{Provider: "groq", Model: "llama-3.1-8b-instant"},
}

// providerBaseURLs maps provider names to their OpenAI-compatible base URLs.
var providerBaseURLs = map[string]string{
	"groq":     "https://api.groq.com/openai/v1",
	"cerebras": "https://api.cerebras.ai/v1",
}

// providerAPIKeyEnvs maps provider names to the environment variable that
// holds the corresponding API key.
var providerAPIKeyEnvs = map[string]string{
	"groq":     "GROQ_API_KEY",
	"cerebras": "CEREBRAS_API_KEY",
}

// slot wraps a Provider with rate-limit cooldown tracking.
type slot struct {
	provider    Provider
	cooldownAt  time.Time
	cooldownDur time.Duration
}

// RotatingProvider implements Provider by delegating to a prioritised list of
// provider+model slots. When a slot returns a 429 (RateLimitError), it is put
// on cooldown and the next available slot is tried automatically.
type RotatingProvider struct {
	mu          sync.Mutex
	slots       []slot
	activeIndex int
}

// NewRotatingProvider creates a RotatingProvider from the given slot configs.
// If slots is nil or empty the defaultSlots list is used. Each slot config is
// resolved to an OpenAICompatibleProvider; slots whose API key cannot be found
// in the environment are skipped with a warning.
func NewRotatingProvider(slots []SlotConfig) (*RotatingProvider, error) {
	if len(slots) == 0 {
		slots = defaultSlots
	}

	var built []slot
	for _, sc := range slots {
		baseURL, ok := providerBaseURLs[sc.Provider]
		if !ok {
			slog.Warn("Unknown provider in rotating slot, skipping",
				"provider", sc.Provider, "model", sc.Model)
			continue
		}

		envVar, ok := providerAPIKeyEnvs[sc.Provider]
		if !ok {
			slog.Warn("No API key env var defined for provider, skipping",
				"provider", sc.Provider, "model", sc.Model)
			continue
		}

		apiKey := os.Getenv(envVar)
		if apiKey == "" {
			slog.Warn("API key not set for provider, skipping slot",
				"provider", sc.Provider, "env_var", envVar, "model", sc.Model)
			continue
		}

		p := NewOpenAIProvider(sc.Provider, baseURL, apiKey, sc.Model, 60*time.Second)
		built = append(built, slot{provider: p})
	}

	if len(built) == 0 {
		return nil, fmt.Errorf("rotating provider: no valid slots configured (check API key env vars)")
	}

	slog.Info("Rotating provider initialized", "slots", len(built))
	return &RotatingProvider{
		slots:       built,
		activeIndex: 0,
	}, nil
}

// Name returns "rotating".
func (r *RotatingProvider) Name() string {
	return "rotating"
}

// GetModel returns the model of the currently active slot.
func (r *RotatingProvider) GetModel() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.slots[r.activeIndex].provider.GetModel()
}

// SetModel is a no-op for the rotating provider because the model is
// determined by the slot configuration.
func (r *RotatingProvider) SetModel(model string) {
	slog.Debug("SetModel called on rotating provider (no-op)", "model", model)
}

// SupportsTools returns true because all cloud providers support tool calling.
func (r *RotatingProvider) SupportsTools() bool {
	return true
}

// Chat sends the request to the highest-priority available slot. If a slot
// returns a RateLimitError it is placed on cooldown and the next slot is tried.
// Non-rate-limit errors are returned immediately without trying other slots.
func (r *RotatingProvider) Chat(ctx context.Context, messages []Message, tools []Tool) (*Response, error) {
	// Determine which slots are available (not on cooldown).
	r.mu.Lock()
	now := time.Now()
	var available []int
	var shortestWait time.Duration
	for i := range r.slots {
		remaining := r.slots[i].cooldownDur - now.Sub(r.slots[i].cooldownAt)
		if remaining <= 0 {
			available = append(available, i)
		} else {
			if shortestWait == 0 || remaining < shortestWait {
				shortestWait = remaining
			}
		}
	}
	r.mu.Unlock()

	if len(available) == 0 {
		// If shortest wait is brief, wait and retry once
		if shortestWait > 0 && shortestWait <= 15*time.Second {
			slog.Info("All LLM slots rate-limited, waiting for cooldown", "wait", shortestWait.Round(time.Second))
			select {
			case <-time.After(shortestWait + 500*time.Millisecond):
				// Re-collect available slots after waiting
				r.mu.Lock()
				available = nil
				now = time.Now()
				for i := range r.slots {
					remaining := r.slots[i].cooldownDur - now.Sub(r.slots[i].cooldownAt)
					if remaining <= 0 {
						available = append(available, i)
					}
				}
				r.mu.Unlock()
				if len(available) == 0 {
					return nil, fmt.Errorf("all LLM slots still rate-limited after waiting %s", shortestWait.Round(time.Second))
				}
				// Fall through to try available slots
			case <-ctx.Done():
				return nil, fmt.Errorf("all LLM slots rate-limited and request cancelled")
			}
		} else {
			return nil, fmt.Errorf("all LLM slots rate-limited, shortest wait: %s", shortestWait.Round(time.Second))
		}
	}

	// Try each available slot in priority order.
	for _, idx := range available {
		s := &r.slots[idx]
		slog.Info("Trying LLM slot",
			"provider", s.provider.Name(),
			"model", s.provider.GetModel())

		resp, err := s.provider.Chat(ctx, messages, tools)
		if err != nil {
			var rle *RateLimitError
			if errors.As(err, &rle) {
				r.mu.Lock()
				s.cooldownAt = time.Now()
				s.cooldownDur = rle.RetryAfter
				r.mu.Unlock()

				slog.Warn("Slot rate-limited, trying next",
					"provider", s.provider.Name(),
					"model", s.provider.GetModel(),
					"retry_after", rle.RetryAfter.Round(time.Second))
				continue
			}
			// Non-rate-limit error: return immediately.
			return nil, err
		}

		// Success: record active index.
		r.mu.Lock()
		r.activeIndex = idx
		r.mu.Unlock()
		return resp, nil
	}

	return nil, fmt.Errorf("all LLM slots rate-limited during request")
}

// ListModels aggregates models from all unique providers across the slots.
// Each provider is queried at most once.
func (r *RotatingProvider) ListModels(ctx context.Context) ([]Model, error) {
	r.mu.Lock()
	slots := make([]slot, len(r.slots))
	copy(slots, r.slots)
	r.mu.Unlock()

	seen := make(map[string]bool)
	var all []Model

	for _, s := range slots {
		name := s.provider.Name()
		if seen[name] {
			continue
		}
		seen[name] = true

		models, err := s.provider.ListModels(ctx)
		if err != nil {
			slog.Warn("Failed to list models from provider",
				"provider", name, "error", err)
			continue
		}
		all = append(all, models...)
	}

	return all, nil
}
