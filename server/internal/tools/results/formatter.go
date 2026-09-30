package results

import (
	"github.com/rhythm493/pocket-assistant/server/internal/llm"
)

type Formatter interface {
	CanFormat(toolName string) bool
	Format(toolName string, result interface{}, args map[string]interface{}) []llm.ComponentBlock
}

var formatters []Formatter

func Register(f Formatter) {
	formatters = append(formatters, f)
}

func FormatResult(toolName string, result interface{}, args map[string]interface{}) []llm.ComponentBlock {
	for _, f := range formatters {
		if f.CanFormat(toolName) {
			if blocks := f.Format(toolName, result, args); len(blocks) > 0 {
				return blocks
			}
		}
	}
	return nil
}
