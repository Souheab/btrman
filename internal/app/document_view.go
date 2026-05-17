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
	return m.frame(title, body, m.viewDocumentCommandLine(status))
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
	status := ""
	if len(m.searchMatches) > 0 && m.currentMatch >= 0 {
		status = fmt.Sprintf("Match %d/%d", m.currentMatch+1, len(m.searchMatches))
	} else if m.status == "No matches." {
		status = m.status
	}
	body := m.viewport.View()
	if m.width >= sidebarThreshold && len(m.doc.Sections) > 0 {
		body = lipgloss.JoinHorizontal(lipgloss.Top, m.viewSidebar(), body)
	}
	return m.frame(titleStyle.Render(m.doc.Title), body, m.viewSearchCommandLine(status))
}

func (m Model) viewDocumentCommandLine(status string) string {
	content := status
	if m.hasScrollCount {
		content = fmt.Sprintf("%d", m.scrollCount)
	}
	return commandLineStyle.Width(max(1, m.width)).MaxWidth(max(1, m.width)).Render(content)
}

func (m Model) viewSearchCommandLine(status string) string {
	input := m.findInput
	availableWidth := max(1, m.width-commandLineStyle.GetHorizontalFrameSize())
	input.PromptStyle = commandLineStyle
	input.TextStyle = commandLineStyle
	input.Cursor.Style = commandLineStyle
	input.PlaceholderStyle = commandLineStyle

	right := truncate(status, availableWidth)
	rightWidth := lipgloss.Width(right)
	gapWidth := 0
	if status != "" {
		gapWidth = 1
	}

	leftWidth := max(1, availableWidth-rightWidth-gapWidth)
	input.Width = max(1, leftWidth-lipgloss.Width(input.Prompt))
	left := commandLineStyle.Width(leftWidth).MaxWidth(leftWidth).Render(input.View())

	content := left
	if right != "" {
		content += commandLineStyle.Render(strings.Repeat(" ", max(0, availableWidth-lipgloss.Width(left)-rightWidth)))
		content += commandLineStyle.Render(right)
	}
	return content
}
