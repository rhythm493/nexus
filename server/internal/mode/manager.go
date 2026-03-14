package mode

import (
	"fmt"
	"strings"

	"github.com/rhythm493/pocket-assistant/server/config"
	"github.com/rhythm493/pocket-assistant/server/internal/mcp"
)

// Manager handles mode configuration and tool filtering
type Manager struct {
	modes       []config.ModeConfig
	defaultMode *config.ModeConfig
}

// NewManager creates a new mode manager
func NewManager(modes []config.ModeConfig) (*Manager, error) {
	if len(modes) == 0 {
		return nil, fmt.Errorf("at least one mode required")
	}

	m := &Manager{modes: modes}

	// Find default mode
	for i := range modes {
		if modes[i].DefaultMode {
			m.defaultMode = &modes[i]
			break
		}
	}

	// If no default specified, use first mode
	if m.defaultMode == nil {
		m.defaultMode = &modes[0]
	}

	return m, nil
}

// GetMode returns mode by ID, or default if not found
func (m *Manager) GetMode(id string) config.ModeConfig {
	if id == "" {
		return *m.defaultMode
	}

	for _, mode := range m.modes {
		if mode.ID == id {
			return mode
		}
	}

	return *m.defaultMode
}

// ListModes returns all available modes
func (m *Manager) ListModes() []config.ModeConfig {
	return m.modes
}

// FilterTools returns tools matching mode's tool filters
func (m *Manager) FilterTools(modeID string, allTools []mcp.Tool) []mcp.Tool {
	mode := m.GetMode(modeID)

	if len(mode.ToolFilters) == 0 {
		return allTools // No filters = return all
	}

	var filtered []mcp.Tool
	for _, tool := range allTools {
		if m.matchesFilter(tool.Name, mode.ToolFilters) {
			filtered = append(filtered, tool)
		}
	}

	return filtered
}

// matchesFilter checks if tool name matches any filter pattern
func (m *Manager) matchesFilter(toolName string, filters []string) bool {
	for _, filter := range filters {
		if strings.HasSuffix(filter, "*") {
			// Prefix match: "grocery_*" matches "grocery_search"
			prefix := strings.TrimSuffix(filter, "*")
			if strings.HasPrefix(toolName, prefix) {
				return true
			}
		} else if filter == toolName {
			// Exact match
			return true
		}
	}
	return false
}

// MatchesFilter is a public version for use by other packages
func (m *Manager) MatchesFilter(toolName string, mode config.ModeConfig) bool {
	return m.matchesFilter(toolName, mode.ToolFilters)
}

// GetSystemPrompt returns mode-specific system prompt
func (m *Manager) GetSystemPrompt(modeID string) string {
	mode := m.GetMode(modeID)
	if mode.SystemPrompt != "" {
		return mode.SystemPrompt
	}
	// Fallback to generic prompt
	return "You are a helpful assistant."
}
