package mcp

// RichContentItem represents content extracted from an MCP tool result
// with structured data preserved for UI rendering.
type RichContentItem struct {
	Type     string      `json:"type"`     // "text", "image", "component", etc.
	Text     string      `json:"text,omitempty"`
	Data     interface{} `json:"data,omitempty"`     // Structured UI component data
	MIMEType string      `json:"mimeType,omitempty"` // MIME type for images
	Source   string      `json:"source,omitempty"`   // "base64" or URL for images
}

// ExtractRichContent extracts all content items from a ToolCallResult,
// preserving structured data (component, image) alongside text.
func ExtractRichContent(result *ToolCallResult) []RichContentItem {
	var items []RichContentItem
	for _, c := range result.Content {
		item := RichContentItem{
			Type:     c.Type,
			Text:     c.Text,
			Data:     c.Data,
			MIMEType: c.MIMEType,
			Source:   c.Source,
		}
		items = append(items, item)
	}
	return items
}

// HasComponentContent returns true if any content item has type "component".
func HasComponentContent(items []RichContentItem) bool {
	for _, item := range items {
		if item.Type == "component" {
			return true
		}
	}
	return false
}

// HasImageContent returns true if any content item has type "image".
func HasImageContent(items []RichContentItem) bool {
	for _, item := range items {
		if item.Type == "image" {
			return true
		}
	}
	return false
}

// ExtractTexts returns all text content items concatenated.
func ExtractTexts(items []RichContentItem) string {
	var texts []string
	for _, item := range items {
		if item.Type == "text" && item.Text != "" {
			texts = append(texts, item.Text)
		}
	}
	if len(texts) == 0 {
		return ""
	}
	if len(texts) == 1 {
		return texts[0]
	}
	result := texts[0]
	for _, t := range texts[1:] {
		result += "\n" + t
	}
	return result
}
