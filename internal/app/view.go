package app

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	switch m.mode {
	case modeLoading:
		return m.frame(titleStyle.Render("btrman"), "", m.status)
	case modeCommandSearch:
		return m.viewCommandSearch()
	case modeInPageSearch:
		return m.viewInPageSearch()
	case modeRelated:
		return m.viewRelated()
	case modeHistory:
		return m.viewHistory()
	case modeError:
		return m.frame(titleStyle.Render("btrman"), errorStyle.Render(m.errorText), "q quit")
	case modeDocument:
		fallthrough
	default:
		return m.viewDocument()
	}
}

func (m Model) frame(header, body, footer string) string {
	availableWidth := max(20, m.width)
	parts := make([]string, 0, 3)
	if header != "" {
		parts = append(parts, lipgloss.NewStyle().Width(availableWidth).Render(header))
	}
	if body != "" {
		parts = append(parts, body)
	}
	if footer != "" {
		parts = append(parts, statusStyle.Width(availableWidth).Render(footer))
	}
	return strings.Join(parts, "\n")
}
