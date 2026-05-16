package app

import "strings"

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
