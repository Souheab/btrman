package search

import (
	"testing"

	"btrman/internal/document"
	"btrman/internal/manual"
)

func TestPageIndexQueryRanksFuzzyMatches(t *testing.T) {
	idx := NewPageIndex([]manual.Page{
		{Ref: manual.PageRef{Name: "grep", Section: "1"}, Description: "print lines matching a pattern"},
		{Ref: manual.PageRef{Name: "printf", Section: "1"}, Description: "format and print data"},
		{Ref: manual.PageRef{Name: "find", Section: "1"}, Description: "search for files"},
	})
	results := idx.Query("pr", 2)
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if results[0].Ref.Name != "printf" {
		t.Fatalf("top result = %q, want printf", results[0].Ref.Name)
	}
}

func TestFindInDocument(t *testing.T) {
	doc := document.Document{Lines: []document.Line{
		{Text: "first printf call"},
		{Text: "second line"},
		{Text: "PRINTF again"},
	}}
	matches := FindInDocument(doc, "printf")
	if len(matches) != 2 {
		t.Fatalf("got %d matches, want 2", len(matches))
	}
	if matches[0].Line != 0 || matches[1].Line != 2 {
		t.Fatalf("unexpected match lines: %#v", matches)
	}
}

func TestMoveMatchWraps(t *testing.T) {
	matches := []LineMatch{{Line: 1}, {Line: 3}, {Line: 5}}
	if got := MoveMatch(matches, 2, 1); got != 0 {
		t.Fatalf("MoveMatch forward = %d, want 0", got)
	}
	if got := MoveMatch(matches, 0, -1); got != 2 {
		t.Fatalf("MoveMatch backward = %d, want 2", got)
	}
}
