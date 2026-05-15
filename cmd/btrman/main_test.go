package main

import (
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
