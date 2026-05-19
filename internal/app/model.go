package app

import (
	"context"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	btrclipboard "btrman/internal/clipboard"
	"btrman/internal/history"
	"btrman/internal/manual"
)

type History interface {
	Entries() ([]history.Entry, error)
	Add(ref manual.PageRef, title string) error
}

type Config struct {
	Context    context.Context
	Provider   manual.Provider
	History    History
	Copier     btrclipboard.Copier
	InitialRef manual.PageRef
	Version    string
}

type mode int

const (
	modeLoading mode = iota
	modeDocument
	modeCommandSearch
	modeInPageSearch
	modeSwiperSearch
	modeRelated
	modeHistory
	modeError
)

type Model struct {
	ctx      context.Context
	provider manual.Provider
	history  History
	copier   btrclipboard.Copier
	version  string

	mode       mode
	width      int
	height     int
	status     string
	errorText  string
	initialRef manual.PageRef

	BrowserState
	DocumentState
	SearchState
	SwiperState
	PreviewState
	HistoryState
	RelatedState
	NavigationState
}

func New(cfg Config) Model {
	ctx := cfg.Context
	if ctx == nil {
		ctx = context.Background()
	}
	provider := cfg.Provider
	if provider == nil {
		provider = manual.NewSystemProvider()
	}
	copier := cfg.Copier
	if copier == nil {
		copier = btrclipboard.NoopCopier{}
	}

	pageInput := textinput.New()
	pageInput.Prompt = "›  "
	pageInput.Placeholder = "search man pages"
	pageInput.Focus()
	pageInput.CharLimit = 256
	pageInput.PromptStyle = promptStyle
	pageInput.TextStyle = inputTextStyle
	pageInput.PlaceholderStyle = placeholderStyle
	pageInput.Cursor.Style = cursorStyle

	findInput := textinput.New()
	findInput.Prompt = "/ "
	findInput.Placeholder = "search within page"
	findInput.CharLimit = 256
	findInput.PromptStyle = promptStyle
	findInput.TextStyle = inputTextStyle
	findInput.PlaceholderStyle = placeholderStyle
	findInput.Cursor.Style = cursorStyle

	swiperInput := textinput.New()
	swiperInput.Prompt = "Search (swiper): "
	swiperInput.Placeholder = "search within page"
	swiperInput.CharLimit = 256
	swiperInput.PromptStyle = promptStyle
	swiperInput.TextStyle = inputTextStyle
	swiperInput.PlaceholderStyle = placeholderStyle
	swiperInput.Cursor.Style = cursorStyle

	vp := viewport.New(80, 20)
	m := Model{
		ctx:        ctx,
		provider:   provider,
		history:    cfg.History,
		copier:     copier,
		version:    cfg.Version,
		mode:       modeLoading,
		width:      100,
		height:     30,
		status:     "Loading manual page index…",
		initialRef: cfg.InitialRef,
		BrowserState: BrowserState{
			pageInput: pageInput,
		},
		DocumentState: DocumentState{
			viewport: vp,
		},
		SearchState: SearchState{
			findInput:    findInput,
			currentMatch: -1,
		},
		SwiperState: SwiperState{
			swiperInput:    swiperInput,
			selectedSwiper: -1,
		},
	}
	m.resize()
	return m
}

func Run(ctx context.Context, cfg Config) error {
	cfg.Context = ctx
	program := tea.NewProgram(New(cfg), tea.WithAltScreen())
	_, err := program.Run()
	return err
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{loadPagesCmd(m.ctx, m.provider)}
	if !m.initialRef.IsZero() {
		cmds = append(cmds, loadPageCmd(m.ctx, m.provider, m.initialRef, m.contentWidth()))
	}
	return tea.Batch(cmds...)
}
