package app

import (
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"btrman/internal/history"
	"btrman/internal/manual"
)

type fakeProvider struct {
	pages []manual.Page
	text  map[string]string
}

func (f fakeProvider) ListPages(context.Context) ([]manual.Page, error) {
	return f.pages, nil
}

func (f fakeProvider) OpenPage(_ context.Context, ref manual.PageRef, _ int) (manual.RawPage, error) {
	if f.text == nil {
		return manual.RawPage{}, errors.New("missing fake page")
	}
	text, ok := f.text[ref.Key()]
	if !ok {
		return manual.RawPage{}, errors.New("missing fake page")
	}
	return manual.RawPage{Ref: ref, Text: text}, nil
}

type memoryHistory struct {
	entries []history.Entry
	added   []manual.PageRef
}

func (h *memoryHistory) Entries() ([]history.Entry, error) {
	return h.entries, nil
}

func (h *memoryHistory) Add(ref manual.PageRef, title string) error {
	h.added = append(h.added, ref)
	h.entries = append([]history.Entry{{Ref: ref, Title: title}}, h.entries...)
	return nil
}

type memoryCopier struct {
	text string
}

func (c *memoryCopier) Write(text string) error {
	c.text = text
	return nil
}

func TestPagesLoadedOpensCommandSearch(t *testing.T) {
	m := New(Config{Provider: fakeProvider{}, Copier: &memoryCopier{}})
	updated, _ := m.Update(pagesLoadedMsg{pages: []manual.Page{{Ref: manual.PageRef{Name: "ls", Section: "1"}}}})
	got := updated.(Model)
	if got.mode != modeCommandSearch {
		t.Fatalf("mode = %v, want command search", got.mode)
	}
	if len(got.pageResults) != 1 || got.pageResults[0].Ref.String() != "ls(1)" {
		t.Fatalf("unexpected page results: %#v", got.pageResults)
	}
}

func TestCommandSearchEnterLoadsSelectedPage(t *testing.T) {
	provider := fakeProvider{
		pages: []manual.Page{{Ref: manual.PageRef{Name: "ls", Section: "1"}}},
		text: map[string]string{
			"ls(1)": "LS(1)\n\nNAME\n       ls - list files\n",
		},
	}
	hist := &memoryHistory{}
	m := New(Config{Provider: provider, History: hist, Copier: &memoryCopier{}})
	updated, _ := m.Update(pagesLoadedMsg{pages: provider.pages})
	m = updated.(Model)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("expected load command")
	}
	updated, _ = m.Update(cmd())
	m = updated.(Model)
	if m.mode != modeDocument {
		t.Fatalf("mode = %v, want document", m.mode)
	}
	if m.currentRef.String() != "ls(1)" {
		t.Fatalf("current ref = %q", m.currentRef.String())
	}
	if len(hist.added) != 1 || hist.added[0].String() != "ls(1)" {
		t.Fatalf("history not updated: %#v", hist.added)
	}
}

func TestDocumentSearchAndCopy(t *testing.T) {
	copier := &memoryCopier{}
	m := New(Config{Provider: fakeProvider{}, Copier: copier})
	updated, _ := m.Update(pageLoadedMsg{raw: manual.RawPage{Ref: manual.PageRef{Name: "printf", Section: "1"}, Text: "PRINTF(1)\n\nNAME\n       printf - format data\n\nEXAMPLES\n       printf hello\n"}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = updated.(Model)
	if len(m.searchMatches) == 0 {
		t.Fatal("expected search matches")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(Model)
	if copier.text == "" {
		t.Fatal("expected copied text")
	}
}
