package models

import (
	"strings"
)

var providerModelMappings = map[string]map[string]string{
	"gemini": {
		"models/gemini-2.5-flash":          "google/gemini-2.5-flash",
		"models/gemini-2.5-flash-lite":     "google/gemini-2.5-flash-lite",
		"models/gemini-2.5-pro":            "google/gemini-2.5-pro",
		"models/gemini-2.0-flash":          "google/gemini-2.0-flash",
		"models/gemini-3-flash-preview":    "google/gemini-3-flash-preview",
		"models/gemini-3-pro-preview":      "google/gemini-3-pro-preview",
		"models/gemma-4-26b-a4b-it":        "google/gemma-4-26b-a4b-it",
		"models/gemma-4-31b-it":            "google/gemma-4-31b-it",
		"models/gemma-3-27b-it":            "google/gemma-3-27b-it",
		"models/gemma-3-12b-it":            "google/gemma-3-12b-it",
		"models/gemini-2.5-flash-lite-preview-09-2025": "google/gemini-2.5-flash-lite-preview-09-2025",
	},
	"groq": {
		"llama-3.3-70b-versatile":            "meta-llama/llama-3.3-70b-instruct",
		"llama-3.1-8b-instant":               "meta-llama/llama-3.1-8b-instruct",
		"meta-llama/llama-4-scout-17b-16e-instruct": "meta-llama/llama-4-scout",
		"qwen/qwen3-32b":                     "qwen/qwen3-32b",
		"moonshotai/kimi-k2-instruct":        "moonshotai/kimi-k2",
	},
}

func EnrichModel(providerName, modelID string, cache *Cache) *ModelMeta {
	orID := resolveOpenRouterID(providerName, modelID)
	if orID == "" {
		return nil
	}

	meta, err := cache.Lookup(orID)
	if err != nil {
		return nil
	}
	if meta == nil {
		meta = fuzzySearch(cache, providerName, modelID)
	}
	return meta
}

func resolveOpenRouterID(provider, modelID string) string {
	mapping, ok := providerModelMappings[provider]
	if !ok {
		return ""
	}

	if target, ok := mapping[modelID]; ok {
		return target
	}

	return ""
}

func fuzzySearch(cache *Cache, provider, modelID string) *ModelMeta {
	var keywords []string

	switch provider {
	case "gemini":
		id := strings.TrimPrefix(modelID, "models/")
		keywords = []string{"google/" + id}
	case "groq":
		parts := strings.Split(modelID, "/")
		base := parts[len(parts)-1]
		base = strings.TrimSuffix(base, "-versatile")
		base = strings.TrimSuffix(base, "-instruct")
		base = strings.TrimSuffix(base, "-instant")
		keywords = []string{base}
	case "cerebras":
		keywords = []string{"cerebras/" + modelID, modelID}
	default:
		return nil
	}

	for _, kw := range keywords {
		results, err := cache.SearchByKeyword(kw)
		if err != nil || len(results) == 0 {
			continue
		}
		return results[0]
	}

	return nil
}
