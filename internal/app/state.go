package app

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"

	"btrman/internal/document"
	"btrman/internal/history"
	"btrman/internal/manual"
	"btrman/internal/search"
)

type BrowserState struct {
	pageInput    textinput.Model
	pages        []manual.Page
	pageIndex    *search.PageIndex
	pageResults  []manual.Page
	selectedPage int
}

type DocumentState struct {
	viewport        viewport.Model
	doc             document.Document
	plainLines      []string
	currentRef      manual.PageRef
	selectedSection int
	scrollCount     int
	hasScrollCount  bool
}

type SearchState struct {
	findInput     textinput.Model
	searchMatches []search.LineMatch
	currentMatch  int
}

type PreviewState struct {
	previewRef     manual.PageRef
	previewDoc     document.Document
	previewLoading bool
	previewError   string
}

type HistoryState struct {
	recents        []history.Entry
	selectedRecent int
}

type RelatedState struct {
	related     []manual.PageRef
	selectedRel int
}

type NavigationState struct {
	backStack []manual.PageRef
}
