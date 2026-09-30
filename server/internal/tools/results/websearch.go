package results

import (
	"github.com/rhythm493/pocket-assistant/server/internal/llm"
	"github.com/rhythm493/pocket-assistant/server/internal/websearch"
)

type webSearchFormatter struct{}

func init() {
	Register(&webSearchFormatter{})
}

func (f *webSearchFormatter) CanFormat(toolName string) bool {
	return toolName == "web_search"
}

func (f *webSearchFormatter) Format(toolName string, result interface{}, args map[string]interface{}) []llm.ComponentBlock {
	searchResults, ok := result.([]websearch.SearchResult)
	if !ok || len(searchResults) == 0 {
		return nil
	}

	query := ""
	if q, ok := args["query"].(string); ok {
		query = q
	}

	return []llm.ComponentBlock{
		{
			Type: "search_results",
			Data: map[string]interface{}{
				"tool":    toolName,
				"query":   query,
				"count":   len(searchResults),
				"results": searchResults,
			},
		},
	}
}
