package manual

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

type runnerFunc func(context.Context, string, []string, []string) ([]byte, error)

func (f runnerFunc) Run(ctx context.Context, name string, args []string, env []string) ([]byte, error) {
	return f(ctx, name, args, env)
}

func TestParsePageRef(t *testing.T) {
	tests := []struct {
		input string
		want  PageRef
	}{
		{input: "printf", want: PageRef{Name: "printf"}},
		{input: "printf(1)", want: PageRef{Name: "printf", Section: "1"}},
		{input: "systemd.service(5)", want: PageRef{Name: "systemd.service", Section: "5"}},
	}
	for _, tt := range tests {
		got, ok := ParsePageRef(tt.input)
		if !ok {
			t.Fatalf("ParsePageRef(%q) returned !ok", tt.input)
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Fatalf("ParsePageRef(%q) = %#v, want %#v", tt.input, got, tt.want)
		}
	}
}

func TestParseAproposLineMultipleNames(t *testing.T) {
	pages := ParseAproposLine("printf, fprintf (3) - formatted output conversion")
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	if pages[0].Ref.String() != "printf(3)" || pages[1].Ref.String() != "fprintf(3)" {
		t.Fatalf("unexpected refs: %#v", pages)
	}
	if pages[0].Description != "formatted output conversion" {
		t.Fatalf("unexpected description: %q", pages[0].Description)
	}
}

func TestParseAproposOutputDeduplicatesAndSorts(t *testing.T) {
	pages := ParseAproposOutput("zcat (1) - compress files\ncat (1) - concatenate files\ncat (1) - duplicate\n")
	refs := []string{pages[0].Ref.String(), pages[1].Ref.String()}
	want := []string{"cat(1)", "zcat(1)"}
	if !reflect.DeepEqual(refs, want) {
		t.Fatalf("refs = %#v, want %#v", refs, want)
	}
}

func TestCleanManOutput(t *testing.T) {
	input := "_\bls\nB\bBO\bOL\bLD\bD\x1b[0m\r\n"
	got := CleanManOutput(input)
	want := "ls\nBOLD\n"
	if got != want {
		t.Fatalf("CleanManOutput() = %q, want %q", got, want)
	}
}

func TestSystemProviderOpenPageUsesManAndCleansOutput(t *testing.T) {
	var gotName string
	var gotArgs []string
	var gotEnv []string
	provider := &SystemProvider{Runner: runnerFunc(func(ctx context.Context, name string, args []string, env []string) ([]byte, error) {
		gotName = name
		gotArgs = append([]string(nil), args...)
		gotEnv = append([]string(nil), env...)
		return []byte("P\bPR\bRINTF\n"), nil
	})}

	raw, err := provider.OpenPage(context.Background(), PageRef{Name: "printf", Section: "1"}, 88)
	if err != nil {
		t.Fatalf("OpenPage returned error: %v", err)
	}
	if gotName != "man" {
		t.Fatalf("command name = %q, want man", gotName)
	}
	wantArgs := []string{"-P", "cat", "1", "printf"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}
	if !contains(gotEnv, "MANWIDTH=88") || !contains(gotEnv, "MANPAGER=cat") {
		t.Fatalf("env missing pager/width values: %#v", gotEnv)
	}
	if raw.Text != "PRINTF\n" {
		t.Fatalf("raw text = %q", raw.Text)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if strings.EqualFold(value, want) {
			return true
		}
	}
	return false
}
