package app

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"btrman/internal/manual"
)

func loadPagesCmd(ctx context.Context, provider manual.Provider) tea.Cmd {
	return func() tea.Msg {
		pages, err := provider.ListPages(ctx)
		return pagesLoadedMsg{pages: pages, err: err}
	}
}

func loadPageCmd(ctx context.Context, provider manual.Provider, ref manual.PageRef, width int) tea.Cmd {
	return func() tea.Msg {
		raw, err := provider.OpenPage(ctx, ref, width)
		return pageLoadedMsg{ref: ref, raw: raw, err: err}
	}
}

func loadPreviewCmd(ctx context.Context, provider manual.Provider, ref manual.PageRef, width int) tea.Cmd {
	return func() tea.Msg {
		raw, err := provider.OpenPage(ctx, ref, width)
		return previewLoadedMsg{ref: ref, raw: raw, err: err}
	}
}

func loadHistoryCmd(store History) tea.Cmd {
	return func() tea.Msg {
		entries, err := store.Entries()
		return historyLoadedMsg{entries: entries, err: err}
	}
}
