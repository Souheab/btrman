package manual

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	ErrUnavailable = errors.New("manual page tooling is unavailable")
	ErrNoPages     = errors.New("no manual pages were discovered")
	ErrNotFound    = errors.New("manual page was not found")
)

// Provider reads official manual pages from a backing source.
type Provider interface {
	ListPages(ctx context.Context) ([]Page, error)
	OpenPage(ctx context.Context, ref PageRef, width int) (RawPage, error)
}

// Runner abstracts command execution so the provider can be tested without the host man database.
type Runner interface {
	Run(ctx context.Context, name string, args []string, env []string) ([]byte, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, name string, args []string, env []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), env...)
	return cmd.CombinedOutput()
}

type SystemProvider struct {
	Runner Runner
}

func NewSystemProvider() *SystemProvider {
	return &SystemProvider{Runner: ExecRunner{}}
}

func (p *SystemProvider) runner() Runner {
	if p.Runner != nil {
		return p.Runner
	}
	return ExecRunner{}
}

func (p *SystemProvider) ListPages(ctx context.Context) ([]Page, error) {
	out, err := p.runner().Run(ctx, "man", []string{"-k", "."}, nil)
	if err != nil {
		fallback, fallbackErr := p.runner().Run(ctx, "apropos", []string{"."}, nil)
		if fallbackErr != nil {
			if errors.Is(err, exec.ErrNotFound) || errors.Is(fallbackErr, exec.ErrNotFound) {
				return nil, ErrUnavailable
			}
			return nil, fmt.Errorf("discover manual pages: %w: %s", ErrNoPages, firstNonEmptyOutput(out, fallback))
		}
		out = fallback
	}

	pages := ParseAproposOutput(string(out))
	if len(pages) == 0 {
		return nil, ErrNoPages
	}
	return pages, nil
}

func (p *SystemProvider) OpenPage(ctx context.Context, ref PageRef, width int) (RawPage, error) {
	if ref.IsZero() {
		return RawPage{}, fmt.Errorf("open manual page: %w", ErrNotFound)
	}
	if width < 60 {
		width = 100
	}

	args := []string{"-P", "cat"}
	if ref.Section != "" {
		args = append(args, ref.Section)
	}
	args = append(args, ref.Name)

	env := []string{
		"MANWIDTH=" + strconv.Itoa(width),
		"MANPAGER=cat",
		"PAGER=cat",
		"LESS=FRX",
		"GROFF_NO_SGR=1",
	}

	out, err := p.runner().Run(ctx, "man", args, env)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return RawPage{}, ErrUnavailable
		}
		return RawPage{}, fmt.Errorf("open %s: %w: %s", ref.String(), ErrNotFound, strings.TrimSpace(string(out)))
	}

	return RawPage{Ref: ref, Text: CleanManOutput(string(out))}, nil
}

var aproposLineRE = regexp.MustCompile(`^\s*(.+?)\s+\(([^)]+)\)\s+-\s+(.*)\s*$`)

func ParseAproposOutput(output string) []Page {
	scanner := bufio.NewScanner(strings.NewReader(output))
	seen := map[string]struct{}{}
	var pages []Page
	for scanner.Scan() {
		for _, page := range ParseAproposLine(scanner.Text()) {
			if page.Ref.IsZero() {
				continue
			}
			key := page.Ref.Key()
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			pages = append(pages, page)
		}
	}
	sort.SliceStable(pages, func(i, j int) bool {
		if pages[i].Ref.Name == pages[j].Ref.Name {
			return pages[i].Ref.Section < pages[j].Ref.Section
		}
		return pages[i].Ref.Name < pages[j].Ref.Name
	})
	return pages
}

func ParseAproposLine(line string) []Page {
	match := aproposLineRE.FindStringSubmatch(line)
	if match == nil {
		return nil
	}
	names, section, description := match[1], strings.TrimSpace(match[2]), strings.TrimSpace(match[3])
	var pages []Page
	for _, name := range strings.Split(names, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		pages = append(pages, Page{Ref: PageRef{Name: name, Section: section}, Description: description})
	}
	return pages
}

var ansiEscapeRE = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

func CleanManOutput(input string) string {
	input = strings.ReplaceAll(input, "\r\n", "\n")
	input = strings.ReplaceAll(input, "\r", "")
	input = ansiEscapeRE.ReplaceAllString(input, "")

	var out []rune
	for _, r := range input {
		if r == '\b' {
			if len(out) > 0 {
				out = out[:len(out)-1]
			}
			continue
		}
		out = append(out, r)
	}
	return string(out)
}

func firstNonEmptyOutput(values ...[]byte) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(string(bytes.TrimSpace(value)))
		if trimmed != "" {
			return trimmed
		}
	}
	return "no output"
}
