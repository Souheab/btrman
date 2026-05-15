package app

import (
	"fmt"
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

func (m Model) viewDocument() string {
	title := titleStyle.Render(m.doc.Title)
	if !m.currentRef.IsZero() {
		title += subtleStyle.Render("  " + m.currentRef.String())
	}
	body := m.viewport.View()
	if m.width >= 90 && len(m.doc.Sections) > 0 {
		body = lipgloss.JoinHorizontal(lipgloss.Top, m.viewSidebar(), body)
	}
	status := m.status
	if status == "" {
		status = "o open · / search · [ ] sections · r related · h history · y copy · q quit"
	}
	return m.frame(title, body, status)
}

func (m Model) viewSidebar() string {
	var b strings.Builder
	b.WriteString(subtleStyle.Render("SECTIONS"))
	b.WriteString("\n")
	limit := m.viewport.Height - 1
	if limit < 1 {
		limit = len(m.doc.Sections)
	}
	start := 0
	if m.selectedSection >= limit {
		start = m.selectedSection - limit + 1
	}
	end := start + limit
	if end > len(m.doc.Sections) {
		end = len(m.doc.Sections)
	}
	for i := start; i < end; i++ {
		label := truncate(m.doc.Sections[i].Title, sidebarWidth-3)
		line := "  " + label
		if i == m.selectedSection {
			line = selectedStyle.Width(sidebarWidth - 1).Render("› " + label)
		}
		b.WriteString(line)
		if i < end-1 {
			b.WriteString("\n")
		}
	}
	return lipgloss.NewStyle().Width(sidebarWidth).PaddingRight(1).Render(b.String())
}

func (m Model) viewCommandSearch() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Open manual page"))
	b.WriteString("\n\n")
	b.WriteString(m.pageInput.View())
	b.WriteString("\n\n")
	if m.pageIndex == nil {
		b.WriteString(subtleStyle.Render("Loading page index…"))
	} else if len(m.pageResults) == 0 {
		b.WriteString(subtleStyle.Render("No matches."))
	} else {
		for i, page := range m.pageResults {
			prefix := "  "
			line := fmt.Sprintf("%s %-18s %s", prefix, page.Ref.String(), page.Description)
			line = truncate(line, max(20, m.width-2))
			if i == m.selectedPage {
				line = selectedStyle.Width(max(20, m.width-2)).Render("› " + strings.TrimSpace(line))
			}
			b.WriteString(line)
			if i < len(m.pageResults)-1 {
				b.WriteString("\n")
			}
		}
	}
	return m.frame("", b.String(), "enter open · esc back · q quit")
}

func (m Model) viewInPageSearch() string {
	status := m.status
	if len(m.searchMatches) > 0 && m.currentMatch >= 0 {
		status = fmt.Sprintf("Match %d/%d · enter keep · esc close", m.currentMatch+1, len(m.searchMatches))
	}
	body := m.viewport.View()
	if m.width >= 90 && len(m.doc.Sections) > 0 {
		body = lipgloss.JoinHorizontal(lipgloss.Top, m.viewSidebar(), body)
	}
	searchBar := m.findInput.View()
	return m.frame(titleStyle.Render(m.doc.Title), searchBar+"\n"+body, status)
}

func (m Model) viewRelated() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Related manual pages"))
	b.WriteString("\n\n")
	if len(m.related) == 0 {
		b.WriteString(subtleStyle.Render("No related pages detected."))
	} else {
		for i, ref := range m.related {
			line := "  " + ref.String()
			if i == m.selectedRel {
				line = selectedStyle.Width(max(20, m.width-2)).Render("› " + ref.String())
			}
			b.WriteString(line)
			if i < len(m.related)-1 {
				b.WriteString("\n")
			}
		}
	}
	return m.frame("", b.String(), "enter open · esc back")
}

func (m Model) viewHistory() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Recent manual pages"))
	b.WriteString("\n\n")
	if len(m.recents) == 0 {
		b.WriteString(subtleStyle.Render("No recent pages yet."))
	} else {
		for i, entry := range m.recents {
			label := entry.Ref.String()
			if entry.Title != "" {
				label += "  " + entry.Title
			}
			line := "  " + truncate(label, max(20, m.width-4))
			if i == m.selectedRecent {
				line = selectedStyle.Width(max(20, m.width-2)).Render("› " + truncate(label, max(20, m.width-4)))
			}
			b.WriteString(line)
			if i < len(m.recents)-1 {
				b.WriteString("\n")
			}
		}
	}
	return m.frame("", b.String(), "enter open · esc back")
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

func truncate(value string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if width == 1 {
		return "…"
	}
	return string(runes[:width-1]) + "…"
}
