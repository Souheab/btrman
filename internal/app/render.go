package app

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"btrman/internal/document"
)

func styleDocumentLine(line document.Line, query string, currentMatch bool) string {
	text := line.Text
	if query != "" {
		text = highlightQuery(text, query, currentMatch)
	}
	switch line.Kind {
	case document.LineHeading:
		return headingStyle.Render(text)
	case document.LineOption:
		return optionStyle.Render(text)
	case document.LineExample:
		return exampleStyle.Render(text)
	default:
		return text
	}
}

func highlightQuery(text, query string, current bool) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return text
	}
	lowerText := strings.ToLower(text)
	needle := strings.ToLower(query)
	style := matchStyle
	if current {
		style = currentMatchStyle
	}
	var out strings.Builder
	pos := 0
	for {
		idx := strings.Index(lowerText[pos:], needle)
		if idx == -1 {
			out.WriteString(text[pos:])
			break
		}
		absolute := pos + idx
		end := absolute + len(query)
		if end > len(text) {
			out.WriteString(text[pos:])
			break
		}
		out.WriteString(text[pos:absolute])
		out.WriteString(style.Render(text[absolute:end]))
		pos = end
		if pos >= len(text) {
			break
		}
	}
	return out.String()
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
