package llm

import "fmt"

// UIGenerationPrompt returns instructions appended to the system prompt
// telling the LLM it can return structured UI components.
func UIGenerationPrompt() string {
	return fmt.Sprintf(`You can enhance responses with structured UI components by wrapping JSON in %s code fences. These render as native UI components instead of plain text.

Available component types and their data formats:

1. metric — Key performance indicator with optional trend
   { "type": "metric", "label": "Temperature", "value": "72", "unit": "°F", "trend": "up" }

2. card — Grouped content with title and optional actions
   { "type": "card", "title": "Weather", "subtitle": "Today", "content": [...child components...] }

3. table — Tabular data with headers and rows
   { "type": "table", "headers": ["Item", "Price"], "rows": [["Milk", "$3.50"], ["Bread", "$2.00"]], "caption": "Grocery List" }

4. chart — Visual data (bar, line, pie)
   { "type": "chart", "chartType": "bar", "title": "Sales", "data": [{"label": "Jan", "value": 100}, {"label": "Feb", "value": 150}] }

5. image — Display an image from URL or base64
   { "type": "image", "source": "url", "url": "https://example.com/image.png", "alt": "Description", "caption": "Chart" }

6. progress — Show a progress bar
   { "type": "progress", "value": 0.65, "label": "Downloading...", "variant": "linear" }

7. list — Ordered or unordered items
   { "type": "list", "items": [{"title": "Step 1", "subtitle": "Do this"}, {"title": "Step 2"}], "numbered": true }

8. code — Syntax-highlighted code block
   { "type": "code", "language": "python", "code": "print('hello')", "showLineNumbers": true }

9. badge — Status badge
   { "type": "badge", "text": "Active", "color": "green" }

10. divider — Section separator
    { "type": "divider", "label": "Results" }

Layout components can nest children:

11. row — Horizontal layout
    { "type": "row", "children": [...], "mainAxisAlignment": "spaceBetween" }

12. column — Vertical layout
    { "type": "column", "children": [...], "crossAxisAlignment": "center" }

Interactive components trigger tool actions:

13. button — Action trigger
    { "type": "button", "label": "Play", "action": "radio_play", "args": {"query": "song name"}, "variant": "filled" }

14. link — URL or action
    { "type": "link", "text": "View Details", "url": "https://...", "action": "some_tool" }

IMPORTANT RULES:
- Use components to make data visually scannable (metrics, tables, charts)
- Use plain text for conversational responses
- Multiple component blocks can be included in one response
- Action arguments should use proper types (numbers not strings for numeric fields)
- Keep component data concise — avoid deeply nested structures`, "```component\n{...}\n```")
}
