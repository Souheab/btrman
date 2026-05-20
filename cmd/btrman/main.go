package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"btrman/internal/app"
	btrclipboard "btrman/internal/clipboard"
	"btrman/internal/history"
	"btrman/internal/manual"
)

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) > 0 {
		switch args[0] {
		case "-h", "--help", "help":
			fmt.Print(helpText())
			return 0
		case "-v", "--version", "version":
			fmt.Printf("btrman %s\n", version)
			return 0
		}
	}

	raw, inputTTY, err := readPagerInput(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read pager input:", err)
		return 1
	}

	var ref manual.PageRef
	if raw == nil {
		ref, err = parseInitialRef(args)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
	}

	historyStore, err := history.NewStore()
	if err != nil {
		fmt.Fprintln(os.Stderr, "history unavailable:", err)
	}

	cfg := app.Config{
		Provider:   manual.NewSystemProvider(),
		History:    historyStore,
		Copier:     btrclipboard.SystemCopier{},
		InitialRef: ref,
		InitialRaw: raw,
		InputTTY:   inputTTY,
		Version:    version,
	}
	if err := app.Run(context.Background(), cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func readPagerInput(stdin *os.File) (*manual.RawPage, bool, error) {
	info, err := stdin.Stat()
	if err != nil {
		return nil, false, err
	}
	if info.Mode()&os.ModeCharDevice != 0 {
		return nil, false, nil
	}

	data, err := io.ReadAll(stdin)
	if err != nil {
		return nil, false, err
	}
	text := manual.CleanManOutput(string(data))
	raw := manual.RawPage{
		Ref:  inferInputRef(text),
		Text: text,
	}
	return &raw, true, nil
}

func inferInputRef(text string) manual.PageRef {
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if ref, ok := manual.ParsePageRef(fields[0]); ok && ref.Section != "" {
			return ref
		}
	}
	return manual.PageRef{}
}

func parseInitialRef(args []string) (manual.PageRef, error) {
	if len(args) == 0 {
		return manual.PageRef{}, nil
	}
	if len(args) == 1 {
		ref, ok := manual.ParsePageRef(args[0])
		if !ok {
			return manual.PageRef{}, fmt.Errorf("invalid manual page reference %q", args[0])
		}
		return ref, nil
	}
	if len(args) == 2 && looksLikeSection(args[0]) {
		return manual.PageRef{Name: args[1], Section: args[0]}, nil
	}
	return manual.PageRef{}, fmt.Errorf("usage: btrman [page|page(section)|section page]")
}

func looksLikeSection(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 12 {
		return false
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			return false
		}
	}
	return true
}

func helpText() string {
	return `btrman — a modern TUI for Linux manual pages

Usage:
  btrman                 Open fuzzy manual-page search
  btrman <page>          Open a page, such as ls
  btrman <page(section)> Open a specific page reference, such as printf(1)
  btrman <section> <page> Open a section/page pair, such as 1 printf

Keybindings:
  q, ctrl+c       quit
  j/k, arrows     scroll or move selection
  [ / ]           previous/next section
  o               open fuzzy command search
  /               search within the current page
  ctrl+f          swiper-style search within the current page
  n / N           next/previous in-page match
  r or g          related manual pages
  h               recent history
  backspace       go back
  y / Y           copy current line/block
`
}
