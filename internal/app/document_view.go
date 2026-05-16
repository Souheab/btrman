package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewDocument() string {
	title := titleStyle.Render(m.doc.Title)
	if !m.currentRef.IsZero() {
		title += subtleStyle.Render("  " + m.currentRef.String())
	}
	body := m.viewport.View()
	if m.width >= sidebarThreshold && len(m.doc.Sections) > 0 {
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

func (m Model) viewInPageSearch() string {
	status := m.status
	if len(m.searchMatches) > 0 && m.currentMatch >= 0 {
		status = fmt.Sprintf("Match %d/%d · enter keep · esc close", m.currentMatch+1, len(m.searchMatches))
	}
	body := m.viewport.View()
	if m.width >= sidebarThreshold && len(m.doc.Sections) > 0 {
		body = lipgloss.JoinHorizontal(lipgloss.Top, m.viewSidebar(), body)
	}
	searchBar := m.findInput.View()
	return m.frame(titleStyle.Render(m.doc.Title), searchBar+"\n"+body, status)
}
