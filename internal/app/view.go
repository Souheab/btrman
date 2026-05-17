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
	parts := make([]string, 0, 2)
	if header != "" {
		rendered := lipgloss.NewStyle().Width(availableWidth).Render(header)
		parts = append(parts, rendered)
	}
	if body != "" {
		parts = append(parts, body)
	}
	content := strings.Join(parts, "\n")
	if footer != "" {
		rendered := statusStyle.Width(availableWidth).Render(footer)
		footerHeight := lipgloss.Height(rendered)
		spacerHeight := m.height - lipgloss.Height(content) - footerHeight
		if content == "" {
			return strings.Repeat("\n", max(0, spacerHeight)) + rendered
		}
		return content + strings.Repeat("\n", max(1, spacerHeight+1)) + rendered
	}
	return content
}
