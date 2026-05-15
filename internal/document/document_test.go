package document

import (
	"testing"

	"btrman/internal/manual"
)

const sampleManPage = `LS(1) User Commands

NAME
       ls - list directory contents

SYNOPSIS
       ls [OPTION]... [FILE]...

DESCRIPTION
       List information about files.

OPTIONS
       -a, --all
              do not ignore entries starting with .

EXAMPLES
       ls -la /tmp

SEE ALSO
       stat(1), chmod(1), printf(3)
`

func TestParseSectionsAndRelatedLinks(t *testing.T) {
	doc := Parse(manual.PageRef{Name: "ls", Section: "1"}, sampleManPage)
	wantSections := []string{"NAME", "SYNOPSIS", "DESCRIPTION", "OPTIONS", "EXAMPLES", "SEE ALSO"}
	if len(doc.Sections) != len(wantSections) {
		t.Fatalf("got %d sections, want %d", len(doc.Sections), len(wantSections))
	}
	for i, want := range wantSections {
		if doc.Sections[i].Title != want {
			t.Fatalf("section %d = %q, want %q", i, doc.Sections[i].Title, want)
		}
	}
	if len(doc.Related) != 3 {
		t.Fatalf("got %d related refs, want 3", len(doc.Related))
	}
	if doc.Related[0].Ref.String() != "stat(1)" {
		t.Fatalf("first related ref = %q", doc.Related[0].Ref.String())
	}
}

func TestLineClassification(t *testing.T) {
	doc := Parse(manual.PageRef{Name: "ls", Section: "1"}, sampleManPage)
	var foundOption, foundExample bool
	for _, line := range doc.Lines {
		if line.Text == "       -a, --all" && line.Kind == LineOption {
			foundOption = true
		}
		if line.Text == "       ls -la /tmp" && line.Kind == LineExample {
			foundExample = true
		}
	}
	if !foundOption {
		t.Fatal("option line was not classified as LineOption")
	}
	if !foundExample {
		t.Fatal("example line was not classified as LineExample")
	}
}

func TestSectionForLine(t *testing.T) {
	doc := Parse(manual.PageRef{Name: "ls", Section: "1"}, sampleManPage)
	line := doc.SectionStart(3)
	if got := doc.SectionForLine(line); got != 3 {
		t.Fatalf("SectionForLine(%d) = %d, want 3", line, got)
	}
}
