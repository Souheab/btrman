package main

import (
	"os"
	"testing"

	"btrman/internal/manual"
)

func TestParseInitialRef(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want manual.PageRef
	}{
		{name: "none", args: nil, want: manual.PageRef{}},
		{name: "name", args: []string{"ls"}, want: manual.PageRef{Name: "ls"}},
		{name: "reference", args: []string{"printf(1)"}, want: manual.PageRef{Name: "printf", Section: "1"}},
		{name: "section name", args: []string{"1", "printf"}, want: manual.PageRef{Name: "printf", Section: "1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseInitialRef(tt.args)
			if err != nil {
				t.Fatalf("parseInitialRef returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestParseInitialRefRejectsExtraArgs(t *testing.T) {
	if _, err := parseInitialRef([]string{"one", "two", "three"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestInferInputRef(t *testing.T) {
	ref := inferInputRef("PRINTF(1)                 User Commands                PRINTF(1)\n\nNAME\n")
	if ref.String() != "PRINTF(1)" {
		t.Fatalf("ref = %q, want PRINTF(1)", ref.String())
	}
}

func TestReadPagerInputReadsNonTerminalFile(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("LL\bS(1)\r\nNAME\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatal(err)
	}

	raw, inputTTY, err := readPagerInput(file)
	if err != nil {
		t.Fatalf("readPagerInput returned error: %v", err)
	}
	if !inputTTY {
		t.Fatal("inputTTY = false, want true for non-terminal stdin")
	}
	if raw == nil {
		t.Fatal("raw = nil, want pager input")
	}
	if raw.Ref.String() != "LS(1)" {
		t.Fatalf("ref = %q, want LS(1)", raw.Ref.String())
	}
	if raw.Text != "LS(1)\nNAME\n" {
		t.Fatalf("text = %q", raw.Text)
	}
}
