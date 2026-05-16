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
	if m.width >= 80 {
		return m.viewCommandBrowser()
	}

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

func (m Model) viewCommandBrowser() string {
	outerWidth := max(40, m.width)
	outerHeight := max(12, m.height)
	innerWidth := max(38, outerWidth-4)
	footerHeight := 1
	panelHeight := max(9, outerHeight-footerHeight-1)
	panelInnerHeight := max(7, panelHeight-2)
	contentHeight := max(5, panelInnerHeight-1)
	gap := 1
	leftWidth := max(34, (innerWidth*39)/100)
	rightWidth := innerWidth - leftWidth - gap
	if rightWidth < 34 {
		rightWidth = 34
		leftWidth = max(30, innerWidth-rightWidth-gap)
	}

	header := lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).Render(
		titleStyle.Render("btrman") + subtleStyle.Render(" - Browse manual pages"),
	)
	columns := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.viewCommandList(leftWidth, contentHeight),
		strings.Repeat(" ", gap),
		m.viewCommandPreview(rightWidth, contentHeight),
	)
	footer := m.viewCommandFooter(innerWidth)

	body := lipgloss.JoinVertical(lipgloss.Left, header, columns)
	panel := lipgloss.NewStyle().
		Width(innerWidth).
		Height(panelInnerHeight).
		MaxHeight(panelHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238")).
		Padding(0, 1).
		Render(body)
	return lipgloss.JoinVertical(lipgloss.Left, panel, footer)
}

func (m Model) viewCommandList(width, height int) string {
	innerWidth := max(20, width-4)
	inputContentWidth := max(1, innerWidth-inputBoxStyle.GetHorizontalFrameSize())
	inputRowWidth := max(1, inputContentWidth-inputBoxStyle.GetHorizontalPadding())
	count := m.resultCount()
	countWidth := lipgloss.Width(count)
	inputWidth := max(1, inputRowWidth-countWidth-1)
	m.pageInput.Width = inputWidth
	inputView := lipgloss.NewStyle().Width(inputWidth).MaxWidth(inputWidth).Render(m.pageInput.View())
	spacer := strings.Repeat(" ", max(0, inputRowWidth-inputWidth-countWidth))
	inputRow := lipgloss.NewStyle().Width(inputRowWidth).MaxWidth(inputRowWidth).Render(
		lipgloss.JoinHorizontal(lipgloss.Top, inputView, spacer, subtleStyle.Render(count)),
	)

	var b strings.Builder
	b.WriteString(labelStyle.Render("Search"))
	b.WriteString("\n")
	b.WriteString(inputBoxStyle.Width(inputContentWidth).MaxHeight(3).Render(inputRow))
	b.WriteString("\n\n")
	b.WriteString(labelStyle.Render("TOP RESULTS"))
	b.WriteString("\n")

	availableResults := max(1, height-7)
	if m.pageIndex == nil {
		b.WriteString(subtleStyle.Render("Loading page index…"))
	} else if len(m.pageResults) == 0 {
		b.WriteString(subtleStyle.Render("No matches."))
	} else {
		start := 0
		if m.selectedPage >= availableResults {
			start = m.selectedPage - availableResults + 1
		}
		end := min(len(m.pageResults), start+availableResults)
		for i := start; i < end; i++ {
			page := m.pageResults[i]
			refWidth := min(16, max(8, innerWidth/3))
			descWidth := max(8, innerWidth-refWidth-5)
			row := fmt.Sprintf("  %-*s %s", refWidth, truncate(page.Ref.String(), refWidth), subtleStyle.Render(truncate(page.Description, descWidth)))
			if i == m.selectedPage {
				row = selectedStyle.Width(innerWidth).Render("› " + fmt.Sprintf("%-*s %s", refWidth, truncate(page.Ref.String(), refWidth), truncate(page.Description, descWidth)))
			}
			b.WriteString(row)
			if i < end-1 {
				b.WriteString("\n")
			}
		}
	}

	return lipgloss.NewStyle().Width(width).Height(height).MaxHeight(height).Render(b.String())
}

func (m Model) resultCount() string {
	if m.pageIndex == nil {
		return ""
	}
	return fmt.Sprintf("  %d / %d", len(m.pageResults), len(m.pages))
}

func (m Model) viewCommandPreview(width, height int) string {
	innerWidth := max(20, width-4)
	innerHeight := max(5, height-2)

	var content string
	switch {
	case len(m.pageResults) == 0:
		content = subtleStyle.Render("No manual page selected.")
	case m.previewLoading:
		content = subtleStyle.Render("Loading preview for " + m.pageResults[m.selectedPage].Ref.String() + "…")
	case m.previewError != "":
		content = errorStyle.Render("Preview unavailable") + "\n\n" + subtleStyle.Render(truncate(m.previewError, innerWidth))
	case len(m.previewDoc.Lines) > 0:
		content = m.previewContent(innerWidth, innerHeight)
	default:
		content = subtleStyle.Render("Select a result to preview it.")
	}

	previewStyle := panelStyle.Padding(0, 2)
	return previewStyle.
		Width(max(1, width-previewStyle.GetHorizontalFrameSize())).
		Height(max(1, height-previewStyle.GetVerticalFrameSize())).
		MaxHeight(max(1, height)).
		Render(content)
}

func (m Model) previewContent(width, height int) string {
	leftTitle := linkStyle.Render(truncate(m.previewDoc.Ref.String(), max(1, width/3)))
	middleTitle := subtleStyle.Render(truncate("General Commands Manual", max(1, width-(width/3)*2)))
	rightTitle := linkStyle.Render(truncate(m.previewDoc.Ref.String(), max(1, width/3)))
	header := lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().Width(width/3).Render(leftTitle),
		lipgloss.NewStyle().Width(width-(width/3)*2).Align(lipgloss.Center).Render(middleTitle),
		lipgloss.NewStyle().Width(width/3).Align(lipgloss.Right).Render(rightTitle),
	)

	lines := []string{header, ""}
	limit := max(0, height-3)
	for _, line := range m.previewDoc.Lines {
		if len(lines) >= 2+limit {
			break
		}
		styled := styleDocumentLine(line, "", false)
		lines = append(lines, truncateStyledLine(styled, line.Text, width))
	}
	if len(m.previewDoc.Lines) > limit && len(lines) < height {
		lines = append(lines, linkStyle.Render("-- More --"))
	}
	return strings.Join(lines, "\n")
}

func (m Model) viewCommandFooter(width int) string {
	items := []string{
		keyStyle.Render("ENTER") + " Open",
		keyStyle.Render("↑") + " " + keyStyle.Render("↓") + " Navigate",
		keyStyle.Render("ESC") + " Back",
	}
	footer := strings.Join(items, "   ")
	return lipgloss.NewStyle().Width(width).Padding(0, 1).Render(footer)
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

func truncateStyledLine(styled, plain string, width int) string {
	if lipgloss.Width(plain) <= width {
		return styled
	}
	return truncate(plain, width)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
