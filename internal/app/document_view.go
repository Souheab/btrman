package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"btrman/internal/search"
)

func (m Model) viewDocument() string {
	title := titleStyle.Render(m.doc.Title)
	if !m.currentRef.IsZero() {
		title += subtleStyle.Render("  " + m.currentRef.String())
	}
	body := m.viewport.View()
	body = m.viewDocumentBody(body)
	status := m.status
	if status == "" {
		status = "o open · / search · ctrl+f swiper · [ ] sections · r related · h history · y copy · q quit"
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

func (m Model) viewDocumentBody(document string) string {
	if m.width < sidebarThreshold || len(m.doc.Sections) == 0 {
		return document
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, m.viewSidebar(), m.viewDocumentDivider(), document)
}

func (m Model) viewDocumentDivider() string {
	height := max(1, m.viewport.Height)
	lines := make([]string, height)
	for i := range lines {
		lines[i] = "│"
	}
	return dividerStyle.Render(strings.Join(lines, "\n"))
}

func (m Model) viewInPageSearch() string {
	status := ""
	if len(m.searchMatches) > 0 && m.currentMatch >= 0 {
		status = fmt.Sprintf("Match %d/%d", m.currentMatch+1, len(m.searchMatches))
	} else if m.status == "No matches." {
		status = m.status
	}
	body := m.viewport.View()
	body = m.viewDocumentBody(body)
	return m.frame(titleStyle.Render(m.doc.Title), body, m.viewSearchCommandLine(status))
}

func (m Model) viewSwiperSearch() string {
	overlay := m.viewSwiperOverlay()
	overlayHeight := lipgloss.Height(overlay)
	bodyHeight := max(1, m.height-overlayHeight-2)

	viewModel := m
	viewModel.viewport.Height = bodyHeight
	body := viewModel.viewport.View()
	body = viewModel.viewDocumentBody(body)

	title := titleStyle.Render(m.doc.Title)
	if !m.currentRef.IsZero() {
		title += subtleStyle.Render("  " + m.currentRef.String())
	}
	body = lipgloss.JoinVertical(lipgloss.Left, body, overlay)
	return m.frame(title, body, m.viewDocumentCommandLine(m.swiperStatus()))
}

func (m Model) viewSwiperOverlay() string {
	width := max(20, m.width-4)
	innerWidth := max(1, width-inputBoxStyle.GetHorizontalFrameSize()-inputBoxStyle.GetHorizontalPadding())
	inputRow := m.viewSwiperInputRow(innerWidth)
	results := m.viewSwiperResults(innerWidth, m.swiperResultHeight())
	footer := m.viewSwiperFooter(innerWidth)
	content := lipgloss.JoinVertical(lipgloss.Left, inputRow, results, footer)
	return inputBoxStyle.Width(width).Render(content)
}

func (m Model) viewSwiperInputRow(width int) string {
	input := m.swiperInput
	count := m.swiperCount()
	countWidth := lipgloss.Width(count)
	gapWidth := 1
	input.Width = max(1, width-countWidth-gapWidth-lipgloss.Width(input.Prompt))
	left := lipgloss.NewStyle().Width(max(1, width-countWidth-gapWidth)).MaxWidth(max(1, width-countWidth-gapWidth)).Render(input.View())
	spacer := strings.Repeat(" ", max(0, width-lipgloss.Width(left)-countWidth))
	return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(left + spacer + subtleStyle.Render(count))
}

func (m Model) swiperCount() string {
	if strings.TrimSpace(m.swiperInput.Value()) == "" {
		return ""
	}
	if len(m.swiperResults) == 1 {
		return "1 match"
	}
	return fmt.Sprintf("%d matches", len(m.swiperResults))
}

func (m Model) swiperResultHeight() int {
	switch {
	case m.height >= 24:
		return 8
	case m.height >= 16:
		return 5
	default:
		return 3
	}
}

func (m Model) viewSwiperResults(width, height int) string {
	height = max(1, height)
	if strings.TrimSpace(m.swiperInput.Value()) == "" {
		return lipgloss.NewStyle().Width(width).Height(height).MaxHeight(height).Render(subtleStyle.Render("Type to search."))
	}
	if len(m.swiperResults) == 0 {
		return lipgloss.NewStyle().Width(width).Height(height).MaxHeight(height).Render(subtleStyle.Render("No matches."))
	}

	start := 0
	if m.selectedSwiper >= height {
		start = m.selectedSwiper - height + 1
	}
	end := min(len(m.swiperResults), start+height)
	lines := make([]string, 0, height)
	for i := start; i < end; i++ {
		row := m.viewSwiperResultRow(m.swiperResults[i], width, i == m.selectedSwiper)
		lines = append(lines, row)
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return lipgloss.NewStyle().Width(width).Height(height).MaxHeight(height).Render(strings.Join(lines, "\n"))
}

func (m Model) viewSwiperResultRow(result search.LineResult, width int, selected bool) string {
	section := result.SectionTitle
	if section != "" {
		section = truncate(section, min(20, max(8, width/4)))
	}
	sectionWidth := lipgloss.Width(section)
	gapWidth := 2
	textWidth := width
	if section != "" && width > sectionWidth+gapWidth+12 {
		textWidth = width - sectionWidth - gapWidth
	}
	text := strings.TrimSpace(result.Text)
	if text == "" {
		text = fmt.Sprintf("line %d", result.Line+1)
	}
	text = truncate(text, textWidth)
	if selected {
		text = highlightQuery(text, m.swiperInput.Value(), true)
	} else {
		text = highlightQuery(text, m.swiperInput.Value(), false)
	}
	row := lipgloss.NewStyle().Width(textWidth).MaxWidth(textWidth).Render(text)
	if section != "" && textWidth < width {
		row += strings.Repeat(" ", max(0, width-lipgloss.Width(row)-sectionWidth))
		row += subtleStyle.Render(section)
	}
	if selected {
		return selectedStyle.Width(width).Render(row)
	}
	return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(row)
}

func (m Model) viewSwiperFooter(width int) string {
	items := []string{
		keyStyle.Render("↑↓") + "/" + keyStyle.Render("Ctrl+j/k") + " Navigate",
		keyStyle.Render("Enter") + " Jump",
		keyStyle.Render("Esc") + " Cancel",
	}
	footer := strings.Join(items, "   ")
	if lipgloss.Width(footer) > width {
		items = []string{
			keyStyle.Render("↑↓") + " Navigate",
			keyStyle.Render("Enter") + " Jump",
			keyStyle.Render("Esc") + " Cancel",
		}
		footer = strings.Join(items, "   ")
	}
	if lipgloss.Width(footer) > width {
		footer = truncate("↑↓ Navigate   Enter Jump   Esc Cancel", width)
	}
	return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(footer)
}

func (m Model) swiperStatus() string {
	if len(m.swiperResults) > 0 && m.selectedSwiper >= 0 {
		return fmt.Sprintf("Match %d/%d", m.selectedSwiper+1, len(m.swiperResults))
	}
	return m.status
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
