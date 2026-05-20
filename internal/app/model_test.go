package app

import (
	"context"
	"errors"
	"strings"
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

func TestInitialRawInputOpensDocument(t *testing.T) {
	raw := manual.RawPage{Ref: manual.PageRef{Name: "ls", Section: "1"}, Text: "LS(1)\n\nNAME\n       ls - list files\n"}
	m := New(Config{InitialRaw: &raw, Copier: &memoryCopier{}})

	cmd := m.Init()
	if cmd == nil {
		t.Fatal("expected initial raw command")
	}
	updated, _ := m.Update(cmd())
	m = updated.(Model)

	if m.mode != modeDocument {
		t.Fatalf("mode = %v, want document", m.mode)
	}
	if m.currentRef.String() != "ls(1)" {
		t.Fatalf("current ref = %q", m.currentRef.String())
	}
	if m.doc.Title != "LS(1)" {
		t.Fatalf("title = %q, want LS(1)", m.doc.Title)
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

func TestCommandSearchLettersUpdateQuery(t *testing.T) {
	pages := []manual.Page{
		{Ref: manual.PageRef{Name: "jq", Section: "1"}},
		{Ref: manual.PageRef{Name: "kill", Section: "1"}},
	}
	m := New(Config{Provider: fakeProvider{}, Copier: &memoryCopier{}})
	updated, _ := m.Update(pagesLoadedMsg{pages: pages})
	m = updated.(Model)

	for _, r := range []rune{'q', 'j', 'k'} {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}

	if got := m.pageInput.Value(); got != "qjk" {
		t.Fatalf("query = %q, want qjk", got)
	}
}

func TestCommandSearchCtrlJKNavigateResults(t *testing.T) {
	pages := []manual.Page{
		{Ref: manual.PageRef{Name: "alpha", Section: "1"}},
		{Ref: manual.PageRef{Name: "beta", Section: "1"}},
	}
	m := New(Config{Provider: fakeProvider{}, Copier: &memoryCopier{}})
	updated, _ := m.Update(pagesLoadedMsg{pages: pages})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})
	m = updated.(Model)
	if m.selectedPage != 1 {
		t.Fatalf("selected page after ctrl+j = %d, want 1", m.selectedPage)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	m = updated.(Model)
	if m.selectedPage != 0 {
		t.Fatalf("selected page after ctrl+k = %d, want 0", m.selectedPage)
	}
}

func TestCommandSearchCtrlQQuits(t *testing.T) {
	m := New(Config{Provider: fakeProvider{}, Copier: &memoryCopier{}})
	updated, _ := m.Update(pagesLoadedMsg{pages: []manual.Page{{Ref: manual.PageRef{Name: "ls", Section: "1"}}}})
	m = updated.(Model)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlQ})
	if cmd == nil {
		t.Fatal("expected ctrl+q to return quit command")
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

func TestSwiperSearchOpensWithCtrlFAndDoesNotPageDown(t *testing.T) {
	m := newSwiperTestModel()
	m.viewport.SetYOffset(3)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlF})
	m = updated.(Model)

	if m.mode != modeSwiperSearch {
		t.Fatalf("mode = %v, want swiper search", m.mode)
	}
	if got := m.viewport.YOffset; got != 3 {
		t.Fatalf("viewport offset = %d, want 3", got)
	}
	if m.swiperOriginOffset != 3 {
		t.Fatalf("swiper origin = %d, want 3", m.swiperOriginOffset)
	}
}

func TestSwiperSearchTypingFiltersOneRowPerLineAndJumps(t *testing.T) {
	m := newSwiperTestModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlF})
	m = updated.(Model)

	for _, r := range []rune("printf") {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}

	if len(m.swiperResults) != 4 {
		t.Fatalf("swiper results = %d, want 4", len(m.swiperResults))
	}
	if len(m.swiperResults[1].Matches) != 2 {
		t.Fatalf("second result matches = %d, want 2", len(m.swiperResults[1].Matches))
	}
	if m.selectedSwiper != 0 {
		t.Fatalf("selected swiper = %d, want 0", m.selectedSwiper)
	}
	if got, want := m.viewport.YOffset, m.swiperResults[0].Line; got != want {
		t.Fatalf("viewport offset = %d, want %d", got, want)
	}
}

func TestSwiperSearchNavigationMovesViewport(t *testing.T) {
	m := newSwiperTestModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlF})
	m = updated.(Model)
	for _, r := range []rune("printf") {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})
	m = updated.(Model)
	if m.selectedSwiper != 1 || m.viewport.YOffset != m.swiperResults[1].Line {
		t.Fatalf("after ctrl+j selected=%d offset=%d results=%#v", m.selectedSwiper, m.viewport.YOffset, m.swiperResults)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.selectedSwiper != 2 || m.viewport.YOffset != m.swiperResults[2].Line {
		t.Fatalf("after down selected=%d offset=%d results=%#v", m.selectedSwiper, m.viewport.YOffset, m.swiperResults)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	m = updated.(Model)
	if m.selectedSwiper != 1 || m.viewport.YOffset != m.swiperResults[1].Line {
		t.Fatalf("after ctrl+k selected=%d offset=%d results=%#v", m.selectedSwiper, m.viewport.YOffset, m.swiperResults)
	}
}

func TestSwiperSearchEnterKeepsJumpAndEscRestoresOrigin(t *testing.T) {
	m := newSwiperTestModel()
	m.viewport.SetYOffset(3)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlF})
	m = updated.(Model)
	for _, r := range []rune("hello") {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}
	jumpOffset := m.viewport.YOffset
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.mode != modeDocument || m.viewport.YOffset != jumpOffset {
		t.Fatalf("enter mode=%v offset=%d want offset %d", m.mode, m.viewport.YOffset, jumpOffset)
	}

	m.viewport.SetYOffset(3)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlF})
	m = updated.(Model)
	for _, r := range []rune("hello") {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}
	if m.viewport.YOffset == 3 {
		t.Fatal("expected swiper search to jump before cancel")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.mode != modeDocument || m.viewport.YOffset != 3 {
		t.Fatalf("esc mode=%v offset=%d, want document offset 3", m.mode, m.viewport.YOffset)
	}
	if len(m.swiperResults) != 0 || m.swiperInput.Value() != "" {
		t.Fatalf("expected swiper state to clear, got input=%q results=%d", m.swiperInput.Value(), len(m.swiperResults))
	}
}

func TestDocumentNumericPrefixScrollsLines(t *testing.T) {
	m := New(Config{Provider: fakeProvider{}, Copier: &memoryCopier{}})
	lines := []string{"LONG(1)", "", "NAME", "       long - test page", "", "DESCRIPTION"}
	for i := 0; i < 80; i++ {
		lines = append(lines, "       line")
	}
	updated, _ := m.Update(pageLoadedMsg{raw: manual.RawPage{
		Ref:  manual.PageRef{Name: "long", Section: "1"},
		Text: strings.Join(lines, "\n"),
	}})
	m = updated.(Model)

	for _, r := range []rune{'1', '2', 'j'} {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}
	if got := m.viewport.YOffset; got != 12 {
		t.Fatalf("viewport offset after 12j = %d, want 12", got)
	}

	for _, r := range []rune{'3', 'k'} {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}
	if got := m.viewport.YOffset; got != 9 {
		t.Fatalf("viewport offset after 3k = %d, want 9", got)
	}
}

func newSwiperTestModel() Model {
	m := New(Config{Provider: fakeProvider{}, Copier: &memoryCopier{}})
	updated := m.handlePageLoaded(pageLoadedMsg{raw: manual.RawPage{
		Ref: manual.PageRef{Name: "printf", Section: "1"},
		Text: strings.Join([]string{
			"PRINTF(1)",
			"",
			"NAME",
			"       printf printf - format data",
			"",
			"DESCRIPTION",
			"       write formatted output with printf",
			"       no match here",
			"",
			"EXAMPLES",
			"       printf hello",
			"       tail one",
			"       tail two",
			"       tail three",
			"       tail four",
			"       tail five",
			"       tail six",
		}, "\n"),
	}})
	updated.height = 10
	updated.resize()
	return updated
}

func TestDocumentCommandLineShowsNumericPrefix(t *testing.T) {
	m := New(Config{Provider: fakeProvider{}, Copier: &memoryCopier{}})
	updated, _ := m.Update(pageLoadedMsg{raw: manual.RawPage{
		Ref:  manual.PageRef{Name: "long", Section: "1"},
		Text: "LONG(1)\n\nNAME\n       long - test page\n",
	}})
	m = updated.(Model)

	for _, r := range []rune{'1', '2'} {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}

	view := m.viewDocument()
	lines := strings.Split(view, "\n")
	if len(lines) == 0 || !strings.Contains(lines[len(lines)-1], "12") {
		t.Fatalf("expected numeric prefix in document command line:\n%s", view)
	}
}
