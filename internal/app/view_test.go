package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"btrman/internal/document"
	"btrman/internal/manual"
	"btrman/internal/search"
)

func TestPreviewContentFitsHeightWithMoreMarker(t *testing.T) {
	m := New(Config{})
	m.previewDoc = document.Parse(manual.PageRef{Name: "long", Section: "1"}, strings.Join([]string{
		"LONG(1)",
		"",
		"NAME",
		"       long - a long manual page",
		"DESCRIPTION",
		"       first",
		"       second",
		"       third",
		"       fourth",
		"       fifth",
	}, "\n"))

	const height = 6
	content := m.previewContent(40, height)
	if got := lipgloss.Height(content); got > height {
		t.Fatalf("preview content height = %d, want <= %d\n%s", got, height, content)
	}
	if !strings.Contains(content, "-- More --") {
		t.Fatalf("expected more marker in overflowing preview:\n%s", content)
	}
}

func TestCommandBrowserFitsTerminalHeight(t *testing.T) {
	m := New(Config{})
	m.mode = modeCommandSearch
	m.width = 90
	m.height = 14
	m.pages = []manual.Page{{Ref: manual.PageRef{Name: "long", Section: "1"}, Description: "long manual page"}}
	m.pageResults = m.pages
	m.previewDoc = document.Parse(m.pages[0].Ref, strings.Join([]string{
		"LONG(1)",
		"",
		"NAME",
		"       long - a long manual page",
		"DESCRIPTION",
		"       first",
		"       second",
		"       third",
		"       fourth",
		"       fifth",
		"       sixth",
		"       seventh",
	}, "\n"))

	view := m.viewCommandBrowser()
	if got := lipgloss.Height(view); got > m.height {
		t.Fatalf("browser height = %d, want <= %d\n%s", got, m.height, view)
	}

	lines := strings.Split(view, "\n")
	lastBorder := -1
	footerLine := -1
	for i, line := range lines {
		if strings.Contains(line, "╰") {
			lastBorder = i
		}
		if strings.Contains(line, "ENTER") && strings.Contains(line, "Open") {
			footerLine = i
		}
	}
	if lastBorder == -1 {
		t.Fatalf("expected bottom border in browser view:\n%s", view)
	}
	if footerLine <= lastBorder {
		t.Fatalf("expected keybindings below border, border line %d footer line %d\n%s", lastBorder, footerLine, view)
	}
}

func TestCommandListKeepsResultCountOnOneLine(t *testing.T) {
	m := New(Config{})
	m.pages = make([]manual.Page, 5935)
	m.pageResults = make([]manual.Page, 55)
	m.pageIndex = search.NewPageIndex(m.pages)

	view := m.viewCommandList(45, 10)
	if strings.Contains(view, "55 /\n5935") {
		t.Fatalf("result count wrapped onto multiple lines:\n%s", view)
	}
	if !strings.Contains(view, "55 / 5935") {
		t.Fatalf("expected result count on one line:\n%s", view)
	}
}

func TestDocumentCommandLineSticksToBottomAndFullWidth(t *testing.T) {
	m := New(Config{})
	m.width = 48
	m.height = 12
	m.resize()
	m = m.handlePageLoaded(pageLoadedMsg{raw: manual.RawPage{
		Ref:  manual.PageRef{Name: "short", Section: "1"},
		Text: "SHORT(1)\n\nNAME\n       short - small page\n",
	}})
	m.appendScrollCount(7)

	view := m.viewDocument()
	if got := lipgloss.Height(view); got != m.height {
		t.Fatalf("document view height = %d, want %d\n%s", got, m.height, view)
	}
	lines := strings.Split(view, "\n")
	lastLine := lines[len(lines)-1]
	if got := lipgloss.Width(lastLine); got != m.width {
		t.Fatalf("command line width = %d, want %d\n%s", got, m.width, view)
	}
	if !strings.Contains(lastLine, "7") {
		t.Fatalf("expected numeric prefix on command line:\n%s", view)
	}
}

func TestDocumentViewDividesSectionsAndDocument(t *testing.T) {
	m := New(Config{})
	m.width = 100
	m.height = 12
	m.resize()
	m = m.handlePageLoaded(pageLoadedMsg{raw: manual.RawPage{
		Ref: manual.PageRef{Name: "short", Section: "1"},
		Text: strings.Join([]string{
			"SHORT(1)",
			"",
			"NAME",
			"       short - small page",
			"DESCRIPTION",
			"       more text",
		}, "\n"),
	}})

	view := m.viewDocument()
	if !strings.Contains(view, "│") {
		t.Fatalf("expected divider between sections and document:\n%s", view)
	}
	if got := lipgloss.Height(view); got != m.height {
		t.Fatalf("document view height = %d, want %d\n%s", got, m.height, view)
	}
}

func TestInPageSearchRendersInBottomCommandLine(t *testing.T) {
	m := New(Config{})
	m.width = 52
	m.height = 12
	m.resize()
	m = m.handlePageLoaded(pageLoadedMsg{raw: manual.RawPage{
		Ref:  manual.PageRef{Name: "short", Section: "1"},
		Text: "SHORT(1)\n\nNAME\n       short - small page\n",
	}})
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(Model)

	view := m.viewInPageSearch()
	if got := lipgloss.Height(view); got != m.height {
		t.Fatalf("search view height = %d, want %d\n%s", got, m.height, view)
	}
	lines := strings.Split(view, "\n")
	lastLine := lines[len(lines)-1]
	if got := lipgloss.Width(lastLine); got != m.width {
		t.Fatalf("search command line width = %d, want %d\n%s", got, m.width, view)
	}
	if !strings.Contains(lastLine, "/ s") || !strings.Contains(lastLine, "Match") {
		t.Fatalf("expected search prompt and match status in bottom command line:\n%s", view)
	}
	if strings.Contains(lastLine, "enter keep") || strings.Contains(lastLine, "esc close") {
		t.Fatalf("expected compact match status in bottom command line:\n%s", view)
	}
	if !strings.HasSuffix(lastLine, "Match 1/3") {
		t.Fatalf("expected match status at right edge:\n%s", view)
	}
}
