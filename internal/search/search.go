package search

import (
	"sort"
	"strings"

	"github.com/sahilm/fuzzy"

	"btrman/internal/document"
	"btrman/internal/manual"
)

type PageIndex struct {
	pages    []manual.Page
	haystack []string
}

func NewPageIndex(pages []manual.Page) *PageIndex {
	copied := append([]manual.Page(nil), pages...)
	sort.SliceStable(copied, func(i, j int) bool {
		if copied[i].Ref.Name == copied[j].Ref.Name {
			return copied[i].Ref.Section < copied[j].Ref.Section
		}
		return copied[i].Ref.Name < copied[j].Ref.Name
	})

	haystack := make([]string, len(copied))
	for i, page := range copied {
		haystack[i] = strings.ToLower(page.Ref.String() + " " + page.Description)
	}
	return &PageIndex{pages: copied, haystack: haystack}
}

func (idx *PageIndex) Query(query string, limit int) []manual.Page {
	if idx == nil {
		return nil
	}
	if limit <= 0 || limit > len(idx.pages) {
		limit = len(idx.pages)
	}
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return append([]manual.Page(nil), idx.pages[:limit]...)
	}

	matches := fuzzy.Find(query, idx.haystack)
	if len(matches) < limit {
		limit = len(matches)
	}
	results := make([]manual.Page, 0, limit)
	for _, match := range matches[:limit] {
		results = append(results, idx.pages[match.Index])
	}
	return results
}

type LineMatch struct {
	Line  int
	Start int
	End   int
}

func FindInDocument(doc document.Document, query string) []LineMatch {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	needle := strings.ToLower(query)
	var matches []LineMatch
	for i, line := range doc.Lines {
		hay := strings.ToLower(line.Text)
		start := 0
		for {
			idx := strings.Index(hay[start:], needle)
			if idx == -1 {
				break
			}
			absolute := start + idx
			matches = append(matches, LineMatch{Line: i, Start: absolute, End: absolute + len(query)})
			start = absolute + len(needle)
			if start >= len(hay) {
				break
			}
		}
	}
	return matches
}

func MoveMatch(matches []LineMatch, current int, delta int) int {
	if len(matches) == 0 {
		return -1
	}
	if current < 0 || current >= len(matches) {
		current = 0
	}
	next := (current + delta) % len(matches)
	if next < 0 {
		next += len(matches)
	}
	return next
}
