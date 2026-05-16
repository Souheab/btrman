package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	btrclipboard "btrman/internal/clipboard"
	"btrman/internal/document"
	"btrman/internal/history"
	"btrman/internal/manual"
	"btrman/internal/search"
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
	viewport   viewport.Model
	pageInput  textinput.Model
	findInput  textinput.Model
	status     string
	errorText  string
	initialRef manual.PageRef

	pages           []manual.Page
	pageIndex       *search.PageIndex
	pageResults     []manual.Page
	selectedPage    int
	doc             document.Document
	plainLines      []string
	currentRef      manual.PageRef
	selectedSection int
	searchMatches   []search.LineMatch
	currentMatch    int
	previewRef      manual.PageRef
	previewDoc      document.Document
	previewLoading  bool
	previewError    string
	related         []manual.PageRef
	selectedRel     int
	recents         []history.Entry
	selectedRecent  int
	backStack       []manual.PageRef
}

type pagesLoadedMsg struct {
	pages []manual.Page
	err   error
}

type pageLoadedMsg struct {
	ref manual.PageRef
	raw manual.RawPage
	err error
}

type historyLoadedMsg struct {
	entries []history.Entry
	err     error
}

type previewLoadedMsg struct {
	ref manual.PageRef
	raw manual.RawPage
	err error
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

	vp := viewport.New(80, 20)
	m := Model{
		ctx:          ctx,
		provider:     provider,
		history:      cfg.History,
		copier:       copier,
		version:      cfg.Version,
		mode:         modeLoading,
		width:        100,
		height:       30,
		viewport:     vp,
		pageInput:    pageInput,
		findInput:    findInput,
		status:       "Loading manual page index…",
		initialRef:   cfg.InitialRef,
		currentMatch: -1,
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

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		return m, nil
	case pagesLoadedMsg:
		return m.handlePagesLoaded(msg)
	case pageLoadedMsg:
		return m.handlePageLoaded(msg), nil
	case previewLoadedMsg:
		return m.handlePreviewLoaded(msg), nil
	case historyLoadedMsg:
		return m.handleHistoryLoaded(msg), nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m.handleKey(msg)
	default:
		return m, nil
	}
}

func (m Model) handlePagesLoaded(msg pagesLoadedMsg) (Model, tea.Cmd) {
	if msg.err != nil {
		m.status = fmt.Sprintf("Could not load page index: %v", msg.err)
		if m.currentRef.IsZero() && m.initialRef.IsZero() {
			m.mode = modeError
			m.errorText = m.status
		}
		return m, nil
	}
	m.pages = msg.pages
	m.pageIndex = search.NewPageIndex(msg.pages)
	m.recomputePageResults()
	if m.currentRef.IsZero() && m.initialRef.IsZero() && m.mode == modeLoading {
		m.mode = modeCommandSearch
		m.status = "Type to find a manual page."
	}
	return m.ensurePreview()
}

func (m Model) handlePageLoaded(msg pageLoadedMsg) Model {
	if msg.err != nil {
		m.mode = modeError
		m.errorText = msg.err.Error()
		m.status = "Failed to open " + msg.ref.String()
		return m
	}

	m.doc = document.Parse(msg.raw.Ref, msg.raw.Text)
	m.currentRef = msg.raw.Ref
	m.plainLines = m.doc.PlainLines()
	m.related = make([]manual.PageRef, 0, len(m.doc.Related))
	for _, link := range m.doc.Related {
		if link.Ref.Key() == m.currentRef.Key() {
			continue
		}
		m.related = append(m.related, link.Ref)
	}
	m.selectedRel = 0
	m.searchMatches = nil
	m.currentMatch = -1
	m.findInput.SetValue("")
	m.mode = modeDocument
	m.status = "Opened " + msg.raw.Ref.String()
	m.viewport.GotoTop()
	m.selectedSection = 0
	m.rebuildViewportContent()
	if m.history != nil {
		if err := m.history.Add(msg.raw.Ref, m.doc.Title); err != nil {
			m.status = "Opened " + msg.raw.Ref.String() + "; history unavailable: " + err.Error()
		}
	}
	return m
}

func (m Model) handleHistoryLoaded(msg historyLoadedMsg) Model {
	if msg.err != nil {
		m.mode = modeDocument
		m.status = "Could not load history: " + msg.err.Error()
		return m
	}
	m.recents = msg.entries
	m.selectedRecent = 0
	m.mode = modeHistory
	if len(m.recents) == 0 {
		m.status = "No recent pages yet."
	} else {
		m.status = "Recent pages."
	}
	return m
}

func (m Model) handlePreviewLoaded(msg previewLoadedMsg) Model {
	if msg.ref.Key() != m.previewRef.Key() {
		return m
	}
	m.previewLoading = false
	if msg.err != nil {
		m.previewDoc = document.Document{}
		m.previewError = msg.err.Error()
		return m
	}
	m.previewError = ""
	m.previewDoc = document.Parse(msg.raw.Ref, msg.raw.Text)
	return m
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeCommandSearch:
		return m.updateCommandSearch(msg)
	case modeInPageSearch:
		return m.updateInPageSearch(msg)
	case modeRelated:
		return m.updateRelated(msg)
	case modeHistory:
		return m.updateHistory(msg)
	case modeDocument:
		return m.updateDocument(msg)
	case modeError:
		if msg.String() == "q" {
			return m, tea.Quit
		}
		if msg.String() == "esc" && !m.currentRef.IsZero() {
			m.mode = modeDocument
			m.status = ""
			return m, nil
		}
		return m, nil
	case modeLoading:
		if msg.String() == "q" {
			return m, tea.Quit
		}
		return m, nil
	default:
		return m, nil
	}
}

func (m Model) updateCommandSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	previousSelection := ""
	if len(m.pageResults) > 0 && m.selectedPage >= 0 && m.selectedPage < len(m.pageResults) {
		previousSelection = m.pageResults[m.selectedPage].Ref.Key()
	}

	switch msg.String() {
	case "ctrl+q":
		return m, tea.Quit
	case "esc":
		if !m.currentRef.IsZero() {
			m.mode = modeDocument
			m.status = ""
			return m, nil
		}
		return m, tea.Quit
	case "up", "ctrl+k":
		m.selectedPage = clamp(m.selectedPage-1, 0, len(m.pageResults)-1)
		return m.ensurePreview()
	case "down", "ctrl+j":
		m.selectedPage = clamp(m.selectedPage+1, 0, len(m.pageResults)-1)
		return m.ensurePreview()
	case "enter":
		if len(m.pageResults) == 0 {
			m.status = "No matching manual page."
			return m, nil
		}
		return m.startLoad(m.pageResults[m.selectedPage].Ref, true)
	}

	var cmd tea.Cmd
	m.pageInput, cmd = m.pageInput.Update(msg)
	m.recomputePageResults()
	previewCmd := tea.Cmd(nil)
	if len(m.pageResults) > 0 && m.pageResults[m.selectedPage].Ref.Key() != previousSelection {
		m, previewCmd = m.ensurePreview()
	}
	return m, tea.Batch(cmd, previewCmd)
}

func (m Model) updateInPageSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.mode = modeDocument
		return m, nil
	case "up":
		m.gotoMatch(-1)
		return m, nil
	case "down":
		m.gotoMatch(1)
		return m, nil
	}

	var cmd tea.Cmd
	m.findInput, cmd = m.findInput.Update(msg)
	m.searchMatches = search.FindInDocument(m.doc, m.findInput.Value())
	if len(m.searchMatches) == 0 {
		m.currentMatch = -1
		if strings.TrimSpace(m.findInput.Value()) == "" {
			m.status = "Search within page."
		} else {
			m.status = "No matches."
		}
	} else {
		m.currentMatch = 0
		m.viewport.SetYOffset(m.searchMatches[0].Line)
		m.status = fmt.Sprintf("Match 1/%d", len(m.searchMatches))
	}
	m.rebuildViewportContent()
	m.syncSection()
	return m, cmd
}

func (m Model) updateRelated(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.mode = modeDocument
		return m, nil
	case "up", "k":
		m.selectedRel = clamp(m.selectedRel-1, 0, len(m.related)-1)
		return m, nil
	case "down", "j":
		m.selectedRel = clamp(m.selectedRel+1, 0, len(m.related)-1)
		return m, nil
	case "enter":
		if len(m.related) == 0 {
			return m, nil
		}
		return m.startLoad(m.related[m.selectedRel], true)
	}
	return m, nil
}

func (m Model) updateHistory(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.mode = modeDocument
		return m, nil
	case "up", "k":
		m.selectedRecent = clamp(m.selectedRecent-1, 0, len(m.recents)-1)
		return m, nil
	case "down", "j":
		m.selectedRecent = clamp(m.selectedRecent+1, 0, len(m.recents)-1)
		return m, nil
	case "enter":
		if len(m.recents) == 0 {
			return m, nil
		}
		return m.startLoad(m.recents[m.selectedRecent].Ref, true)
	}
	return m, nil
}

func (m Model) updateDocument(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "o":
		if m.pageIndex == nil {
			m.status = "Manual page index is still loading."
			return m, nil
		}
		m.mode = modeCommandSearch
		m.pageInput.SetValue("")
		m.pageInput.Focus()
		m.recomputePageResults()
		m.status = "Type to find a manual page."
		return m, nil
	case "/":
		m.mode = modeInPageSearch
		m.findInput.Focus()
		m.status = "Search within page."
		return m, nil
	case "n":
		m.gotoMatch(1)
		return m, nil
	case "N":
		m.gotoMatch(-1)
		return m, nil
	case "]", "tab":
		m.jumpSection(1)
		return m, nil
	case "[", "shift+tab":
		m.jumpSection(-1)
		return m, nil
	case "r", "g":
		if len(m.related) == 0 {
			m.status = "No related manual references detected."
			return m, nil
		}
		m.mode = modeRelated
		m.selectedRel = 0
		m.status = "Related pages."
		return m, nil
	case "h":
		if m.history == nil {
			m.status = "History is not configured."
			return m, nil
		}
		m.mode = modeLoading
		m.status = "Loading history…"
		return m, loadHistoryCmd(m.history)
	case "backspace", "alt+left":
		if len(m.backStack) == 0 {
			m.status = "No previous page."
			return m, nil
		}
		ref := m.backStack[len(m.backStack)-1]
		m.backStack = m.backStack[:len(m.backStack)-1]
		return m.startLoad(ref, false)
	case "y":
		m.copyCurrentLine()
		return m, nil
	case "Y":
		m.copyCurrentBlock()
		return m, nil
	case "j", "down":
		m.viewport.LineDown(1)
	case "k", "up":
		m.viewport.LineUp(1)
	case "pgdown", "ctrl+f":
		m.viewport.PageDown()
	case "pgup", "ctrl+b":
		m.viewport.PageUp()
	case "home":
		m.viewport.GotoTop()
	case "end":
		m.viewport.GotoBottom()
	default:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		m.syncSection()
		return m, cmd
	}
	m.syncSection()
	return m, nil
}

func (m Model) startLoad(ref manual.PageRef, push bool) (tea.Model, tea.Cmd) {
	if push && !m.currentRef.IsZero() && ref.Key() != m.currentRef.Key() {
		m.backStack = append(m.backStack, m.currentRef)
	}
	m.mode = modeLoading
	m.status = "Loading " + ref.String() + "…"
	return m, loadPageCmd(m.ctx, m.provider, ref, m.contentWidth())
}

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

func (m *Model) recomputePageResults() {
	if m.pageIndex == nil {
		m.pageResults = nil
		m.selectedPage = 0
		return
	}
	m.pageResults = m.pageIndex.Query(m.pageInput.Value(), max(20, m.height-8))
	m.selectedPage = clamp(m.selectedPage, 0, len(m.pageResults)-1)
}

func (m Model) ensurePreview() (Model, tea.Cmd) {
	if m.mode != modeCommandSearch || len(m.pageResults) == 0 {
		m.previewRef = manual.PageRef{}
		m.previewDoc = document.Document{}
		m.previewLoading = false
		m.previewError = ""
		return m, nil
	}
	ref := m.pageResults[m.selectedPage].Ref
	if ref.Key() == m.previewRef.Key() && (m.previewLoading || len(m.previewDoc.Lines) > 0 || m.previewError != "") {
		return m, nil
	}
	m.previewRef = ref
	m.previewDoc = document.Document{}
	m.previewLoading = true
	m.previewError = ""
	return m, loadPreviewCmd(m.ctx, m.provider, ref, m.previewWidth())
}

func (m Model) previewWidth() int {
	if m.width < 90 {
		return max(60, m.width-8)
	}
	return max(60, (m.width*3)/5-8)
}

func (m *Model) resize() {
	if m.width <= 0 {
		m.width = 100
	}
	if m.height <= 0 {
		m.height = 30
	}
	m.pageInput.Width = max(20, m.width-10)
	m.findInput.Width = max(20, m.width-10)
	m.viewport.Width = max(20, m.contentWidth())
	m.viewport.Height = max(5, m.height-4)
	m.rebuildViewportContent()
}

func (m Model) contentWidth() int {
	if m.width >= 90 {
		return m.width - sidebarWidth - 2
	}
	return m.width
}

func (m *Model) rebuildViewportContent() {
	if len(m.doc.Lines) == 0 {
		m.viewport.SetContent("")
		return
	}
	currentLine := -1
	if m.currentMatch >= 0 && m.currentMatch < len(m.searchMatches) {
		currentLine = m.searchMatches[m.currentMatch].Line
	}
	query := ""
	if len(m.searchMatches) > 0 {
		query = m.findInput.Value()
	}
	lines := make([]string, len(m.doc.Lines))
	for i, line := range m.doc.Lines {
		lines[i] = styleDocumentLine(line, query, i == currentLine)
	}
	m.viewport.SetContent(strings.Join(lines, "\n"))
}

func (m *Model) syncSection() {
	m.selectedSection = m.doc.SectionForLine(m.viewport.YOffset)
}

func (m *Model) jumpSection(delta int) {
	if len(m.doc.Sections) == 0 {
		return
	}
	current := m.doc.SectionForLine(m.viewport.YOffset)
	if current < 0 {
		current = 0
	}
	next := clamp(current+delta, 0, len(m.doc.Sections)-1)
	m.selectedSection = next
	m.viewport.SetYOffset(m.doc.SectionStart(next))
	m.status = "Section: " + m.doc.Sections[next].Title
}

func (m *Model) gotoMatch(delta int) {
	if len(m.searchMatches) == 0 {
		m.status = "No search matches."
		return
	}
	m.currentMatch = search.MoveMatch(m.searchMatches, m.currentMatch, delta)
	match := m.searchMatches[m.currentMatch]
	m.viewport.SetYOffset(match.Line)
	m.status = fmt.Sprintf("Match %d/%d", m.currentMatch+1, len(m.searchMatches))
	m.rebuildViewportContent()
	m.syncSection()
}

func (m *Model) copyCurrentLine() {
	if len(m.plainLines) == 0 {
		m.status = "Nothing to copy."
		return
	}
	line := clamp(m.viewport.YOffset, 0, len(m.plainLines)-1)
	text := strings.TrimRight(m.plainLines[line], " ")
	if strings.TrimSpace(text) == "" {
		m.status = "Current line is empty."
		return
	}
	if err := m.copier.Write(text); err != nil {
		m.status = "Copy failed: " + err.Error()
		return
	}
	m.status = "Copied current line."
}

func (m *Model) copyCurrentBlock() {
	if len(m.plainLines) == 0 {
		m.status = "Nothing to copy."
		return
	}
	line := clamp(m.viewport.YOffset, 0, len(m.plainLines)-1)
	start, end := line, line
	for start > 0 && strings.TrimSpace(m.plainLines[start-1]) != "" && m.doc.Lines[start-1].Kind != document.LineHeading {
		start--
	}
	for end+1 < len(m.plainLines) && strings.TrimSpace(m.plainLines[end+1]) != "" && m.doc.Lines[end+1].Kind != document.LineHeading {
		end++
	}
	text := strings.TrimSpace(strings.Join(m.plainLines[start:end+1], "\n"))
	if text == "" {
		m.status = "Current block is empty."
		return
	}
	if err := m.copier.Write(text); err != nil {
		m.status = "Copy failed: " + err.Error()
		return
	}
	m.status = "Copied current block."
}

const sidebarWidth = 24

var (
	accentColor       = lipgloss.Color("99")
	accentBrightColor = lipgloss.Color("141")
	blueColor         = lipgloss.Color("39")
	textColor         = lipgloss.Color("252")
	mutedColor        = lipgloss.Color("244")
	panelColor        = lipgloss.Color("60")
	titleStyle        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	subtleStyle       = lipgloss.NewStyle().Foreground(mutedColor)
	statusStyle       = lipgloss.NewStyle().Foreground(textColor)
	errorStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	headingStyle      = lipgloss.NewStyle().Bold(true).Foreground(accentBrightColor)
	optionStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	exampleStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("151"))
	matchStyle        = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("229"))
	currentMatchStyle = lipgloss.NewStyle().Background(lipgloss.Color("57")).Foreground(lipgloss.Color("230")).Bold(true)
	selectedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(accentColor).Bold(true)
	promptStyle       = lipgloss.NewStyle().Foreground(accentBrightColor).Bold(true)
	inputTextStyle    = lipgloss.NewStyle().Foreground(textColor)
	placeholderStyle  = lipgloss.NewStyle().Foreground(mutedColor)
	cursorStyle       = lipgloss.NewStyle().Foreground(accentBrightColor)
	labelStyle        = lipgloss.NewStyle().Foreground(accentBrightColor).Bold(true)
	linkStyle         = lipgloss.NewStyle().Foreground(blueColor).Bold(true)
	keyStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("225")).Background(lipgloss.Color("57")).Bold(true).Padding(0, 1)
	panelStyle        = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(panelColor)
	inputBoxStyle     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accentColor).Padding(0, 1)
)

func styleDocumentLine(line document.Line, query string, currentMatch bool) string {
	text := line.Text
	if query != "" {
		text = highlightQuery(text, query, currentMatch)
	}
	switch line.Kind {
	case document.LineHeading:
		return headingStyle.Render(text)
	case document.LineOption:
		return optionStyle.Render(text)
	case document.LineExample:
		return exampleStyle.Render(text)
	default:
		return text
	}
}

func highlightQuery(text, query string, current bool) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return text
	}
	lowerText := strings.ToLower(text)
	needle := strings.ToLower(query)
	style := matchStyle
	if current {
		style = currentMatchStyle
	}
	var out strings.Builder
	pos := 0
	for {
		idx := strings.Index(lowerText[pos:], needle)
		if idx == -1 {
			out.WriteString(text[pos:])
			break
		}
		absolute := pos + idx
		end := absolute + len(query)
		if end > len(text) {
			out.WriteString(text[pos:])
			break
		}
		out.WriteString(text[pos:absolute])
		out.WriteString(style.Render(text[absolute:end]))
		pos = end
		if pos >= len(text) {
			break
		}
	}
	return out.String()
}

func clamp(value, minValue, maxValue int) int {
	if maxValue < minValue {
		return minValue
	}
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
