package app

import (
	"strings"
	"testing"

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
