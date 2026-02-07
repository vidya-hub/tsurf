package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/vidyasagar/tsurf/internal/browser"
	"github.com/vidyasagar/tsurf/internal/feeds"
	"github.com/vidyasagar/tsurf/internal/storage"
	"github.com/vidyasagar/tsurf/internal/theme"
	"github.com/vidyasagar/tsurf/internal/ui"
)

// Mode represents the current input mode.
type Mode int

const (
	ModeNormal  Mode = iota
	ModeInsert       // URL bar focused
	ModeCommand      // command bar active
	ModeFollow       // link follow mode
	ModeSearch       // search mode
	ModeHistory      // history panel active
	ModeLeader       // leader key palette active
)

// tabState holds per-tab state.
type tabState struct {
	viewport   ui.PageViewport
	history    *browser.History
	page       *browser.RenderedPage
	feedLinks  []browser.Link // links from feed/search/storage pages
	loading    bool
	cancelFunc context.CancelFunc
}

// Model is the top-level bubbletea model for tsurf.
type Model struct {
	// UI components
	tabBar     ui.TabBar
	urlBar     ui.URLBar
	statusBar  ui.StatusBar
	commandBar ui.CommandBar
	splitPane  ui.SplitPane

	// Per-tab state
	tabStates map[int]*tabState

	// Shared state
	fetcher   *browser.Fetcher
	pageCache *lru.Cache[string, *browser.RenderedPage] // LRU cache for rendered pages
	keys      KeyMap
	mode      Mode
	width     int
	height    int
	lastGKey  bool // for "gg" detection
	ready     bool
	startURL  string

	// Feeds
	hnClient     *feeds.HNClient
	redditClient *feeds.RedditClient
	rssClient    *feeds.RSSClient
	githubClient *feeds.GitHubClient

	// Storage
	db        *storage.DB
	bookmarks *storage.BookmarkStore
	readLater *storage.ReadLaterStore
	config    *storage.Config

	// History
	historyPanel ui.HistoryPanel
	historyStore *storage.HistoryStore

	// Leader key
	leaderPanel ui.LeaderPanel
}

// pageLoadedMsg is sent when a page finishes loading.
type pageLoadedMsg struct {
	tabID int
	page  *browser.RenderedPage
	url   string
	err   error
}

// feedLoadedMsg is sent when a feed finishes loading.
type feedLoadedMsg struct {
	tabID   int
	content string
	title   string
	links   []browser.Link
	err     error
}

// leaderTimeoutMsg is sent when the leader key palette times out.
type leaderTimeoutMsg struct{}

// New creates a new tsurf Model.
func New(startURL string) Model {
	tb := ui.NewTabBar()
	initialTab := tb.ActiveTab()

	// Initialize page cache (stores up to 50 rendered pages for instant back/forward).
	pageCache, _ := lru.New[string, *browser.RenderedPage](50)

	m := Model{
		tabBar:     tb,
		urlBar:     ui.NewURLBar(),
		statusBar:  ui.NewStatusBar(),
		commandBar: ui.NewCommandBar(),
		splitPane:  ui.NewSplitPane(),
		tabStates:  make(map[int]*tabState),
		fetcher:    browser.NewFetcher(),
		pageCache:  pageCache,
		keys:       DefaultKeyMap(),
		mode:       ModeNormal,
		startURL:   startURL,

		// Feeds
		hnClient:     feeds.NewHNClient(),
		redditClient: feeds.NewRedditClient(),
		rssClient:    feeds.NewRSSClient(),
		githubClient: feeds.NewGitHubClient(),
	}

	// Initialize storage (best-effort, non-fatal on error).
	dataDir, err := storage.DataDir()
	if err == nil {
		db, dbErr := storage.OpenDB(dataDir)
		if dbErr == nil {
			m.db = db
			m.bookmarks = storage.NewBookmarkStore(db)
			m.readLater = storage.NewReadLaterStore(db)
			m.historyStore = storage.NewHistoryStore(db)
		}
	}
	m.config, _ = storage.LoadConfig()
	m.historyPanel = ui.NewHistoryPanel()
	m.leaderPanel = ui.NewLeaderPanel()

	// Initialize first tab state.
	m.tabStates[initialTab.ID] = &tabState{
		viewport: ui.NewPageViewport(),
		history:  browser.NewHistory(),
	}

	return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	if m.startURL != "" {
		return m.navigateTo(m.startURL)
	}
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.layout()
		return m, nil

	case pageLoadedMsg:
		return m.handlePageLoaded(msg)

	case feedLoadedMsg:
		return m.handleFeedLoaded(msg)

	case leaderTimeoutMsg:
		if m.mode == ModeLeader {
			m.leaderPanel.Hide()
			m.mode = ModeNormal
			m.statusBar.SetMode("NORMAL")
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	// Forward to active components.
	cmds = append(cmds, m.updateComponents(msg)...)

	return m, tea.Batch(cmds...)
}

// View implements tea.Model.
func (m Model) View() string {
	if !m.ready {
		return "\n  Loading tsurf..."
	}

	// Layout:
	// [tab bar]
	// [url bar]
	// [viewport]
	// [status bar]
	// [command bar] (if active)

	var sections []string

	// Tab bar.
	sections = append(sections, m.tabBar.View())

	// URL bar.
	sections = append(sections, m.urlBar.View())

	// Viewport (with optional history panel on the left).
	ts := m.activeTabState()
	if ts != nil {
		if m.historyPanel.IsVisible() {
			t := theme.Current
			dividerStyle := lipgloss.NewStyle().
				Foreground(t.Border).
				Background(t.Background)

			// Calculate divider height.
			tabBarHeight := 1
			urlBarHeight := 3
			statusBarHeight := 1
			commandBarHeight := 0
			if m.commandBar.IsActive() {
				commandBarHeight = 1
			}
			dividerHeight := m.height - tabBarHeight - urlBarHeight - statusBarHeight - commandBarHeight
			if dividerHeight < 1 {
				dividerHeight = 1
			}

			var dividerLines []string
			for i := 0; i < dividerHeight; i++ {
				dividerLines = append(dividerLines, "│")
			}
			divider := dividerStyle.Render(strings.Join(dividerLines, "\n"))

			content := lipgloss.JoinHorizontal(lipgloss.Top,
				m.historyPanel.View(),
				divider,
				ts.viewport.View(),
			)
			sections = append(sections, content)
		} else {
			sections = append(sections, ts.viewport.View())
		}
	} else {
		sections = append(sections, "")
	}

	// Status bar.
	sections = append(sections, m.statusBar.View())

	// Command bar (if active).
	if m.commandBar.IsActive() {
		sections = append(sections, m.commandBar.View())
	}

	result := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Overlay the leader palette if active.
	if m.leaderPanel.IsVisible() {
		overlay := m.leaderPanel.View()
		result = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(theme.Current.Background),
		)
	}

	return result
}

// layout recalculates dimensions for all components.
func (m *Model) layout() {
	m.tabBar.SetWidth(m.width)
	m.urlBar.SetWidth(m.width)
	m.statusBar.SetWidth(m.width)
	m.commandBar.SetWidth(m.width)
	m.splitPane.SetSize(m.width, m.height)

	// Calculate viewport height.
	tabBarHeight := 1
	urlBarHeight := 3 // border adds height
	statusBarHeight := 1
	commandBarHeight := 0
	if m.commandBar.IsActive() {
		commandBarHeight = 1
	}
	viewportHeight := m.height - tabBarHeight - urlBarHeight - statusBarHeight - commandBarHeight
	if viewportHeight < 1 {
		viewportHeight = 1
	}

	// Calculate viewport width (narrower when history panel is shown).
	viewportWidth := m.width
	if m.historyPanel.IsVisible() {
		panelWidth := m.width * 30 / 100
		if panelWidth < 20 {
			panelWidth = 20
		}
		m.historyPanel.SetSize(panelWidth, viewportHeight)
		viewportWidth = m.width - panelWidth - 1 // -1 for divider
	}

	// Set viewport size for all tabs.
	for _, ts := range m.tabStates {
		ts.viewport.SetSize(viewportWidth, viewportHeight)
	}
}

// handleKeyMsg processes key events based on current mode.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Always allow Ctrl+C to quit.
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.mode {
	case ModeInsert:
		return m.handleInsertMode(msg)
	case ModeCommand, ModeSearch, ModeFollow:
		return m.handleCommandMode(msg)
	case ModeHistory:
		return m.handleHistoryMode(msg)
	case ModeLeader:
		return m.handleLeaderMode(msg)
	default:
		return m.handleNormalMode(msg)
	}
}

// handleNormalMode processes keys in normal (browsing) mode.
func (m Model) handleNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	ts := m.activeTabState()

	switch {
	// Quit.
	case key.Matches(msg, m.keys.Quit) && msg.String() != "ctrl+c":
		if msg.String() == "q" {
			return m, tea.Quit
		}

	// Leader key (Space) — open shortcut palette.
	case msg.String() == " ":
		m.lastGKey = false
		m.leaderPanel.SetSize(m.width, m.height)
		m.leaderPanel.Show()
		m.mode = ModeLeader
		m.statusBar.SetMode("LEADER")
		return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return leaderTimeoutMsg{}
		})

	// gg detection: first "g" sets flag, second "g" goes to top.
	case msg.String() == "g":
		if m.lastGKey {
			m.lastGKey = false
			if ts != nil {
				ts.viewport.GotoTop()
				m.syncStatusBar()
			}
			return m, nil
		}
		m.lastGKey = true
		return m, nil

	// gt detection: "t" after "g" switches to next tab.
	case msg.String() == "t":
		if m.lastGKey {
			m.lastGKey = false
			m.tabBar.NextTab()
			m.syncTabUI()
			return m, nil
		}

	// gT detection: "T" after "g" switches to prev tab.
	case msg.String() == "T":
		if m.lastGKey {
			m.lastGKey = false
			m.tabBar.PrevTab()
			m.syncTabUI()
			return m, nil
		}

	// Scroll down.
	case key.Matches(msg, m.keys.ScrollDown):
		m.lastGKey = false
		if ts != nil {
			ts.viewport.LineDown(1)
			m.syncStatusBar()
		}
		return m, nil

	// Scroll up.
	case key.Matches(msg, m.keys.ScrollUp):
		m.lastGKey = false
		if ts != nil {
			ts.viewport.LineUp(1)
			m.syncStatusBar()
		}
		return m, nil

	// Half page down.
	case key.Matches(msg, m.keys.HalfPageDown):
		m.lastGKey = false
		if ts != nil {
			ts.viewport.HalfPageDown()
			m.syncStatusBar()
		}
		return m, nil

	// Half page up.
	case key.Matches(msg, m.keys.HalfPageUp):
		m.lastGKey = false
		if ts != nil {
			ts.viewport.HalfPageUp()
			m.syncStatusBar()
		}
		return m, nil

	// Go to bottom.
	case key.Matches(msg, m.keys.GotoBottom):
		m.lastGKey = false
		if ts != nil {
			ts.viewport.GotoBottom()
			m.syncStatusBar()
		}
		return m, nil

	// Open URL.
	case key.Matches(msg, m.keys.OpenURL):
		m.lastGKey = false
		m.mode = ModeInsert
		m.urlBar.Reset()
		m.statusBar.SetMode("INSERT")
		cmd := m.urlBar.Focus()
		return m, cmd

	// Back.
	case key.Matches(msg, m.keys.Back):
		m.lastGKey = false
		if ts != nil {
			if url, ok := ts.history.Back(); ok {
				return m, m.loadPage(url, false)
			}
		}
		return m, nil

	// Forward.
	case key.Matches(msg, m.keys.Forward):
		m.lastGKey = false
		if ts != nil {
			if url, ok := ts.history.Forward(); ok {
				return m, m.loadPage(url, false)
			}
		}
		return m, nil

	// Reload.
	case key.Matches(msg, m.keys.Reload):
		m.lastGKey = false
		if ts != nil {
			current := ts.history.Current()
			if current != "" {
				return m, m.loadPage(current, false)
			}
		}
		return m, nil

	// Follow link.
	case key.Matches(msg, m.keys.FollowLink):
		m.lastGKey = false
		m.mode = ModeFollow
		m.statusBar.SetMode("FOLLOW")
		cmd := m.commandBar.Open(ui.CommandFollow)
		return m, cmd

	// New tab.
	case key.Matches(msg, m.keys.NewTab):
		m.lastGKey = false
		m.tabBar.NewTab()
		tab := m.tabBar.ActiveTab()
		m.tabStates[tab.ID] = &tabState{
			viewport: ui.NewPageViewport(),
			history:  browser.NewHistory(),
		}
		m.layout()
		m.syncTabUI()
		return m, nil

	// Close tab.
	case key.Matches(msg, m.keys.CloseTab):
		m.lastGKey = false
		tab := m.tabBar.ActiveTab()
		if m.tabBar.CloseCurrentTab() {
			// Cancel any pending load.
			if ts, ok := m.tabStates[tab.ID]; ok {
				if ts.cancelFunc != nil {
					ts.cancelFunc()
				}
				delete(m.tabStates, tab.ID)
			}
			m.syncTabUI()
		} else {
			// Last tab - quit.
			return m, tea.Quit
		}
		return m, nil

	// Next tab.
	case key.Matches(msg, m.keys.NextTab):
		m.lastGKey = false
		m.tabBar.NextTab()
		m.syncTabUI()
		return m, nil

	// Prev tab.
	case key.Matches(msg, m.keys.PrevTab):
		m.lastGKey = false
		m.tabBar.PrevTab()
		m.syncTabUI()
		return m, nil

	// Command mode.
	case key.Matches(msg, m.keys.CommandMode):
		m.lastGKey = false
		m.mode = ModeCommand
		m.statusBar.SetMode("COMMAND")
		cmd := m.commandBar.Open(ui.CommandEx)
		return m, cmd

	// Search mode.
	case key.Matches(msg, m.keys.SearchMode):
		m.lastGKey = false
		m.mode = ModeSearch
		m.statusBar.SetMode("SEARCH")
		cmd := m.commandBar.Open(ui.CommandSearch)
		return m, cmd

	// Help.
	case key.Matches(msg, m.keys.Help):
		m.lastGKey = false
		m.showHelp()
		return m, nil

	// Split vertical.
	case key.Matches(msg, m.keys.SplitVertical):
		m.lastGKey = false
		m.splitPane.Split(ui.SplitVertical)
		m.layout()
		return m, nil

	// Split close.
	case key.Matches(msg, m.keys.SplitClose):
		m.lastGKey = false
		m.splitPane.Unsplit()
		m.layout()
		return m, nil

	// Split toggle.
	case key.Matches(msg, m.keys.SplitToggle):
		m.lastGKey = false
		m.splitPane.Toggle()
		return m, nil

	// Bookmark current page.
	case key.Matches(msg, m.keys.Bookmark):
		m.lastGKey = false
		if m.bookmarks != nil && ts != nil {
			tab := m.tabBar.ActiveTab()
			if tab != nil && tab.URL != "" {
				if m.bookmarks.Add(tab.URL, tab.Title) {
					m.statusBar.SetMessage(fmt.Sprintf("Bookmarked: %s", tab.Title))
				} else {
					m.statusBar.SetMessage("Already bookmarked")
				}
			} else {
				m.statusBar.SetMessage("No page to bookmark")
			}
		}
		return m, nil

	// Read later.
	case key.Matches(msg, m.keys.ReadLater):
		m.lastGKey = false
		if m.readLater != nil && ts != nil {
			tab := m.tabBar.ActiveTab()
			if tab != nil && tab.URL != "" {
				if m.readLater.Add(tab.URL, tab.Title) {
					m.statusBar.SetMessage(fmt.Sprintf("Added to read later: %s", tab.Title))
				} else {
					m.statusBar.SetMessage("Already in read later")
				}
			} else {
				m.statusBar.SetMessage("No page to save")
			}
		}
		return m, nil

	// Toggle history panel.
	case key.Matches(msg, m.keys.HistoryToggle):
		m.lastGKey = false
		if m.historyPanel.IsVisible() {
			m.historyPanel.Hide()
			m.mode = ModeNormal
			m.statusBar.SetMode("NORMAL")
		} else {
			if m.historyStore != nil {
				entries := m.historyStore.List()
				m.historyPanel.SetEntries(entries)
			}
			m.historyPanel.Show()
			m.mode = ModeHistory
			m.statusBar.SetMode("HISTORY")
		}
		m.layout()
		return m, nil
	}

	// Reset g key if another key was pressed.
	m.lastGKey = false

	// Forward to viewport for mouse scroll, etc.
	if ts != nil {
		vp, cmd := ts.viewport.Update(msg)
		ts.viewport = *vp
		m.syncStatusBar()
		return m, cmd
	}

	return m, nil
}

// handleHistoryMode processes keys when the history panel is active.
func (m Model) handleHistoryMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		m.historyPanel.ResetGKey()
		m.historyPanel.CursorDown()
		return m, nil

	case "k", "up":
		m.historyPanel.ResetGKey()
		m.historyPanel.CursorUp()
		return m, nil

	case "g":
		m.historyPanel.HandleGKey()
		return m, nil

	case "G":
		m.historyPanel.ResetGKey()
		m.historyPanel.GotoBottom()
		return m, nil

	case "ctrl+d":
		m.historyPanel.ResetGKey()
		m.historyPanel.HalfPageDown()
		return m, nil

	case "ctrl+u":
		m.historyPanel.ResetGKey()
		m.historyPanel.HalfPageUp()
		return m, nil

	case "d":
		m.historyPanel.ResetGKey()
		idx := m.historyPanel.SelectedIndex()
		m.historyPanel.RemoveSelected()
		if m.historyStore != nil {
			m.historyStore.Remove(idx)
		}
		return m, nil

	case "enter":
		m.historyPanel.ResetGKey()
		entry := m.historyPanel.SelectedEntry()
		if entry != nil {
			// Open in a new tab.
			m.tabBar.NewTab()
			tab := m.tabBar.ActiveTab()
			m.tabStates[tab.ID] = &tabState{
				viewport: ui.NewPageViewport(),
				history:  browser.NewHistory(),
			}
			// Close history panel and return to normal mode.
			m.historyPanel.Hide()
			m.mode = ModeNormal
			m.statusBar.SetMode("NORMAL")
			m.layout()
			m.syncTabUI()
			return m, m.navigateTo(entry.URL)
		}
		return m, nil

	case "esc", "ctrl+h":
		m.historyPanel.ResetGKey()
		m.historyPanel.Hide()
		m.mode = ModeNormal
		m.statusBar.SetMode("NORMAL")
		m.layout()
		return m, nil
	}

	// Reset g key on any other key press.
	m.historyPanel.ResetGKey()
	return m, nil
}

// handleLeaderMode processes keys when the leader palette is active.
// Each key maps to a specific action, then returns to normal mode.
func (m Model) handleLeaderMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Always dismiss the palette first.
	m.leaderPanel.Hide()
	m.mode = ModeNormal
	m.statusBar.SetMode("NORMAL")

	ts := m.activeTabState()

	switch msg.String() {
	// ── Navigate ──
	case "o": // Open URL
		m.mode = ModeInsert
		m.urlBar.Reset()
		m.statusBar.SetMode("INSERT")
		return m, m.urlBar.Focus()

	case "b": // Back
		if ts != nil {
			if url, ok := ts.history.Back(); ok {
				return m, m.loadPage(url, false)
			}
		}
		return m, nil

	case "f": // Forward
		if ts != nil {
			if url, ok := ts.history.Forward(); ok {
				return m, m.loadPage(url, false)
			}
		}
		return m, nil

	case "l": // Follow link
		m.mode = ModeFollow
		m.statusBar.SetMode("FOLLOW")
		return m, m.commandBar.Open(ui.CommandFollow)

	case "r": // Reload
		if ts != nil {
			current := ts.history.Current()
			if current != "" {
				return m, m.loadPage(current, false)
			}
		}
		return m, nil

	// ── Tabs ──
	case "t": // New tab
		m.tabBar.NewTab()
		tab := m.tabBar.ActiveTab()
		m.tabStates[tab.ID] = &tabState{
			viewport: ui.NewPageViewport(),
			history:  browser.NewHistory(),
		}
		m.layout()
		m.syncTabUI()
		return m, nil

	case "w": // Close tab
		tab := m.tabBar.ActiveTab()
		if m.tabBar.CloseCurrentTab() {
			if ts, ok := m.tabStates[tab.ID]; ok {
				if ts.cancelFunc != nil {
					ts.cancelFunc()
				}
				delete(m.tabStates, tab.ID)
			}
			m.syncTabUI()
		} else {
			return m, tea.Quit
		}
		return m, nil

	case "n": // Next tab
		m.tabBar.NextTab()
		m.syncTabUI()
		return m, nil

	case "p": // Prev tab
		m.tabBar.PrevTab()
		m.syncTabUI()
		return m, nil

	// ── Feeds ──
	case "h": // Hacker News
		m.statusBar.SetLoading(true)
		m.statusBar.SetMessage("Loading Hacker News...")
		return m, m.fetchHN("top")

	case "e": // Reddit
		m.mode = ModeCommand
		m.statusBar.SetMode("COMMAND")
		cmd := m.commandBar.Open(ui.CommandEx)
		m.commandBar.SetValue("reddit ")
		return m, cmd

	case "s": // Search
		m.mode = ModeCommand
		m.statusBar.SetMode("COMMAND")
		cmd := m.commandBar.Open(ui.CommandEx)
		m.commandBar.SetValue("search ")
		return m, cmd

	case "a": // RSS feed
		m.mode = ModeCommand
		m.statusBar.SetMode("COMMAND")
		cmd := m.commandBar.Open(ui.CommandEx)
		m.commandBar.SetValue("rss ")
		return m, cmd

	// ── Tools ──
	case "B": // Bookmarks
		return m.executeCommand("bookmarks")

	case "R": // Read later
		return m.executeCommand("readlater")

	case "/": // Search page
		m.mode = ModeSearch
		m.statusBar.SetMode("SEARCH")
		return m, m.commandBar.Open(ui.CommandSearch)

	case ":": // Command mode
		m.mode = ModeCommand
		m.statusBar.SetMode("COMMAND")
		return m, m.commandBar.Open(ui.CommandEx)

	// ── Views ──
	case "H": // History panel
		if m.historyStore != nil {
			entries := m.historyStore.List()
			m.historyPanel.SetEntries(entries)
		}
		m.historyPanel.Show()
		m.mode = ModeHistory
		m.statusBar.SetMode("HISTORY")
		m.layout()
		return m, nil

	case "v": // Split vertical
		m.splitPane.Split(ui.SplitVertical)
		m.layout()
		return m, nil

	case "x": // Close split
		m.splitPane.Unsplit()
		m.layout()
		return m, nil

	case "T": // Theme cycle
		return m.cycleTheme()

	case "?": // Help
		m.showHelp()
		return m, nil

	case "esc", " ":
		// Already dismissed above.
		return m, nil
	}

	// Unknown key — just dismiss.
	return m, nil
}

// cycleTheme switches to the next available theme.
func (m Model) cycleTheme() (tea.Model, tea.Cmd) {
	themes := theme.List()
	current := theme.Current.Name
	for i, t := range themes {
		if t == current {
			next := themes[(i+1)%len(themes)]
			theme.Set(next)
			m.statusBar.SetMessage(fmt.Sprintf("Theme: %s", next))
			return m, nil
		}
	}
	// Fallback: set first theme.
	if len(themes) > 0 {
		theme.Set(themes[0])
		m.statusBar.SetMessage(fmt.Sprintf("Theme: %s", themes[0]))
	}
	return m, nil
}

// handleInsertMode processes keys when the URL bar is focused.
func (m Model) handleInsertMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = ModeNormal
		m.urlBar.Blur()
		m.statusBar.SetMode("NORMAL")
		return m, nil

	case tea.KeyEnter:
		url := m.urlBar.Value()
		m.mode = ModeNormal
		m.urlBar.Blur()
		m.statusBar.SetMode("NORMAL")
		if url != "" {
			return m, m.navigateTo(url)
		}
		return m, nil
	}

	ub, cmd := m.urlBar.Update(msg)
	m.urlBar = *ub
	return m, cmd
}

// handleCommandMode processes keys in command/search/follow mode.
func (m Model) handleCommandMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.commandBar.Close()
		m.mode = ModeNormal
		m.statusBar.SetMode("NORMAL")
		return m, nil

	case tea.KeyEnter:
		result := m.commandBar.Submit()
		m.mode = ModeNormal
		m.statusBar.SetMode("NORMAL")
		return m.handleCommandResult(result)
	}

	cb, cmd := m.commandBar.Update(msg)
	m.commandBar = *cb
	return m, cmd
}

// handleCommandResult processes a submitted command.
func (m Model) handleCommandResult(result ui.CommandResult) (tea.Model, tea.Cmd) {
	switch result.Type {
	case ui.CommandEx:
		return m.executeCommand(result.Value)
	case ui.CommandSearch:
		m.statusBar.SetMessage(fmt.Sprintf("Search: %s (not yet implemented)", result.Value))
		return m, nil
	case ui.CommandFollow:
		return m.followLink(result.Value)
	}
	return m, nil
}

// executeCommand handles :commands.
func (m Model) executeCommand(cmd string) (tea.Model, tea.Cmd) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return m, nil
	}

	switch parts[0] {
	case "q", "quit":
		return m, tea.Quit
	case "o", "open":
		if len(parts) > 1 {
			url := strings.Join(parts[1:], " ")
			return m, m.navigateTo(url)
		}
		m.statusBar.SetMessage("Usage: :open <url>")
	case "theme":
		if len(parts) > 1 {
			if theme.Set(parts[1]) {
				m.statusBar.SetMessage(fmt.Sprintf("Theme: %s", parts[1]))
			} else {
				m.statusBar.SetMessage(fmt.Sprintf("Unknown theme: %s (available: %s)", parts[1], strings.Join(theme.List(), ", ")))
			}
		} else {
			m.statusBar.SetMessage(fmt.Sprintf("Current: %s | Available: %s", theme.Current.Name, strings.Join(theme.List(), ", ")))
		}
	case "tab", "tabnew":
		m.tabBar.NewTab()
		tab := m.tabBar.ActiveTab()
		m.tabStates[tab.ID] = &tabState{
			viewport: ui.NewPageViewport(),
			history:  browser.NewHistory(),
		}
		m.layout()
		m.syncTabUI()
		if len(parts) > 1 {
			url := strings.Join(parts[1:], " ")
			return m, m.navigateTo(url)
		}
	case "tabclose", "tc":
		tab := m.tabBar.ActiveTab()
		if m.tabBar.CloseCurrentTab() {
			delete(m.tabStates, tab.ID)
			m.syncTabUI()
		}
	case "split", "vs", "vsplit":
		m.splitPane.Split(ui.SplitVertical)
		m.layout()
	case "sp", "hsplit":
		m.splitPane.Split(ui.SplitHorizontal)
		m.layout()
	case "unsplit":
		m.splitPane.Unsplit()
		m.layout()
	case "help":
		m.showHelp()
	case "hn":
		category := "top"
		if len(parts) > 1 {
			category = parts[1]
		}
		m.statusBar.SetLoading(true)
		m.statusBar.SetMessage("Loading Hacker News...")
		return m, m.fetchHN(category)
	case "reddit":
		subreddit := "programming"
		if len(parts) > 1 {
			subreddit = parts[1]
		}
		m.statusBar.SetLoading(true)
		m.statusBar.SetMessage(fmt.Sprintf("Loading r/%s...", subreddit))
		return m, m.fetchReddit(subreddit)
	case "rss":
		if len(parts) > 1 {
			feedURL := parts[1]
			m.statusBar.SetLoading(true)
			m.statusBar.SetMessage("Loading feed...")
			return m, m.fetchRSS(feedURL)
		}
		m.statusBar.SetMessage("Usage: :rss <url>")
	case "search":
		if len(parts) > 1 {
			query := strings.Join(parts[1:], " ")
			m.statusBar.SetLoading(true)
			m.statusBar.SetMessage(fmt.Sprintf("Searching: %s...", query))
			return m, m.fetchSearch(query)
		}
		m.statusBar.SetMessage("Usage: :search <query>")
	case "bookmarks", "bm":
		if m.bookmarks != nil {
			content, links := storage.RenderBookmarks(m.bookmarks.List())
			ts := m.activeTabState()
			if ts != nil {
				ts.page = nil
				ts.feedLinks = links
				ts.viewport.SetContent(content)
				m.tabBar.SetActiveTitle("Bookmarks")
				m.statusBar.SetTitle("Bookmarks")
				m.statusBar.SetLinkCount(len(links))
			}
		} else {
			m.statusBar.SetMessage("Bookmarks not available")
		}
	case "readlater", "rl":
		if m.readLater != nil {
			content, links := storage.RenderReadLater(m.readLater.ListAll())
			ts := m.activeTabState()
			if ts != nil {
				ts.page = nil
				ts.feedLinks = links
				ts.viewport.SetContent(content)
				m.tabBar.SetActiveTitle("Read Later")
				m.statusBar.SetTitle("Read Later")
				m.statusBar.SetLinkCount(len(links))
			}
		} else {
			m.statusBar.SetMessage("Read later not available")
		}
	case "bookmark":
		if m.bookmarks != nil {
			tab := m.tabBar.ActiveTab()
			if tab != nil && tab.URL != "" {
				if m.bookmarks.Add(tab.URL, tab.Title) {
					m.statusBar.SetMessage(fmt.Sprintf("Bookmarked: %s", tab.Title))
				} else {
					m.statusBar.SetMessage("Already bookmarked")
				}
			} else {
				m.statusBar.SetMessage("No page to bookmark")
			}
		}
	case "history":
		if m.historyStore != nil {
			entries := m.historyStore.List()
			m.historyPanel.SetEntries(entries)
			m.historyPanel.Show()
			m.mode = ModeHistory
			m.statusBar.SetMode("HISTORY")
			m.layout()
		} else {
			m.statusBar.SetMessage("History not available")
		}
	case "clearhistory":
		if m.historyStore != nil {
			m.historyStore.Clear()
			m.statusBar.SetMessage("History cleared")
		}
	default:
		m.statusBar.SetMessage(fmt.Sprintf("Unknown command: %s", parts[0]))
	}

	return m, nil
}

// followLink navigates to a link by its index number.
func (m Model) followLink(input string) (tea.Model, tea.Cmd) {
	ts := m.activeTabState()
	if ts == nil {
		m.statusBar.SetMessage("No page loaded")
		return m, nil
	}

	num, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil {
		m.statusBar.SetMessage(fmt.Sprintf("Invalid link number: %s", input))
		return m, nil
	}

	// Check page links first (normal web pages).
	if ts.page != nil {
		for _, link := range ts.page.Links {
			if link.Index == num {
				return m, m.navigateTo(link.URL)
			}
		}
	}

	// Check feed links (HN, Reddit, RSS, Search, Bookmarks, Read Later).
	if len(ts.feedLinks) > 0 {
		for _, link := range ts.feedLinks {
			if link.Index == num {
				return m, m.navigateTo(link.URL)
			}
		}
	}

	m.statusBar.SetMessage(fmt.Sprintf("Link [%d] not found", num))
	return m, nil
}

// navigateTo loads a URL in the active tab and pushes to history.
func (m Model) navigateTo(url string) tea.Cmd {
	return m.loadPage(url, true)
}

// loadPage fetches and renders a page. If pushHistory is true, adds to history.
func (m Model) loadPage(url string, pushHistory bool) tea.Cmd {
	ts := m.activeTabState()
	if ts == nil {
		return nil
	}

	tab := m.tabBar.ActiveTab()
	tabID := tab.ID

	// Cancel previous load if any.
	if ts.cancelFunc != nil {
		ts.cancelFunc()
	}

	// Check page cache first (for instant back/forward navigation).
	if m.pageCache != nil {
		if cachedPage, ok := m.pageCache.Get(url); ok {
			// Return cached page immediately.
			ts.loading = false
			m.statusBar.SetLoading(false)
			m.urlBar.SetValue(url)
			m.tabBar.SetActiveURL(url)
			if pushHistory {
				ts.history.Push(url)
			}
			return func() tea.Msg {
				return pageLoadedMsg{tabID: tabID, page: cachedPage, url: url}
			}
		}
	}

	ts.loading = true
	m.statusBar.SetLoading(true)
	m.statusBar.SetMessage("")
	m.urlBar.SetValue(url)
	m.tabBar.SetActiveTitle("Loading...")
	m.tabBar.SetActiveURL(url)

	if pushHistory {
		ts.history.Push(url)
	}

	// Intercept Reddit URLs and use .json API instead of HTML fetching.
	redditInfo := feeds.ParseRedditURL(url)
	if redditInfo != nil && redditInfo.Type != feeds.RedditURLNone {
		client := m.redditClient
		return func() tea.Msg {
			content, title, links, err := client.FetchURL(redditInfo)
			if err != nil {
				return feedLoadedMsg{tabID: tabID, err: err}
			}
			return feedLoadedMsg{tabID: tabID, content: content, title: title, links: links}
		}
	}

	// Intercept GitHub URLs and use GitHub API for rich rendering.
	githubInfo := feeds.ParseGitHubURL(url)
	if githubInfo != nil && githubInfo.Type != feeds.GitHubURLNone {
		client := m.githubClient
		width := m.width
		if width <= 0 {
			width = 80
		}
		return func() tea.Msg {
			content, title, links, err := client.FetchURL(githubInfo, width)
			if err != nil {
				return feedLoadedMsg{tabID: tabID, err: err}
			}
			return feedLoadedMsg{tabID: tabID, content: content, title: title, links: links}
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	ts.cancelFunc = cancel

	fetcher := m.fetcher
	pageCache := m.pageCache
	// Capture width for the goroutine (use actual terminal width, constrained for readability).
	renderWidth := m.width
	if renderWidth <= 0 {
		renderWidth = 80
	}

	return func() tea.Msg {
		result, err := fetcher.FetchWithContext(ctx, url)
		if err != nil {
			return pageLoadedMsg{tabID: tabID, err: err, url: url}
		}

		article, err := browser.Extract(result)
		if err != nil {
			return pageLoadedMsg{tabID: tabID, err: err, url: url}
		}

		page := browser.Render(article, renderWidth)

		// Store in cache for future back/forward navigation.
		if pageCache != nil {
			pageCache.Add(result.FinalURL, page)
		}

		return pageLoadedMsg{tabID: tabID, page: page, url: result.FinalURL}
	}
}

// handlePageLoaded processes a completed page load.
func (m Model) handlePageLoaded(msg pageLoadedMsg) (tea.Model, tea.Cmd) {
	ts, ok := m.tabStates[msg.tabID]
	if !ok {
		return m, nil
	}

	ts.loading = false
	ts.cancelFunc = nil

	if msg.err != nil {
		m.statusBar.SetLoading(false)
		m.statusBar.SetMessage(fmt.Sprintf("Error: %s", msg.err))

		errStyle := lipgloss.NewStyle().
			Foreground(theme.Current.Error).
			Bold(true).
			Padding(2, 4)
		detailStyle := lipgloss.NewStyle().
			Foreground(theme.Current.TextDim).
			Padding(0, 4)

		errContent := errStyle.Render("Failed to load page") + "\n\n" +
			detailStyle.Render(fmt.Sprintf("URL: %s\nError: %s", msg.url, msg.err))

		ts.viewport.SetContent(errContent)
		m.tabBar.SetActiveTitle("Error")
		return m, nil
	}

	ts.page = msg.page
	ts.viewport.SetContent(msg.page.Content)

	m.tabBar.SetActiveTitle(msg.page.Title)
	m.tabBar.SetActiveURL(msg.url)
	m.urlBar.SetValue(msg.url)
	m.statusBar.SetLoading(false)
	m.statusBar.SetTitle(msg.page.Title)
	m.statusBar.SetURL(msg.url)
	m.statusBar.SetLinkCount(len(msg.page.Links))
	m.syncStatusBar()

	// Record in global history.
	if m.historyStore != nil {
		m.historyStore.Add(msg.url, msg.page.Title)
	}

	return m, nil
}

// updateComponents forwards messages to sub-components.
func (m *Model) updateComponents(msg tea.Msg) []tea.Cmd {
	var cmds []tea.Cmd

	ts := m.activeTabState()
	if ts != nil {
		vp, cmd := ts.viewport.Update(msg)
		ts.viewport = *vp
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return cmds
}

// activeTabState returns the state for the currently active tab.
func (m *Model) activeTabState() *tabState {
	tab := m.tabBar.ActiveTab()
	if tab == nil {
		return nil
	}
	return m.tabStates[tab.ID]
}

// syncTabUI updates the URL bar, status bar, and link count to reflect the active tab.
func (m *Model) syncTabUI() {
	tab := m.tabBar.ActiveTab()
	if tab != nil {
		m.urlBar.SetValue(tab.URL)
	}
	m.syncStatusBar()
}

// syncStatusBar updates the status bar with current state.
func (m *Model) syncStatusBar() {
	ts := m.activeTabState()
	if ts == nil {
		return
	}
	m.statusBar.SetScrollInfo(ts.viewport.ScrollInfo())

	tab := m.tabBar.ActiveTab()
	if tab != nil {
		m.statusBar.SetURL(tab.URL)
		m.statusBar.SetTitle(tab.Title)
	}

	if ts.page != nil {
		m.statusBar.SetLinkCount(len(ts.page.Links))
	} else if len(ts.feedLinks) > 0 {
		m.statusBar.SetLinkCount(len(ts.feedLinks))
	}
}

// handleFeedLoaded processes a completed feed/search load.
func (m Model) handleFeedLoaded(msg feedLoadedMsg) (tea.Model, tea.Cmd) {
	ts, ok := m.tabStates[msg.tabID]
	if !ok {
		return m, nil
	}

	ts.loading = false
	m.statusBar.SetLoading(false)

	if msg.err != nil {
		m.statusBar.SetMessage(fmt.Sprintf("Error: %s", msg.err))

		errStyle := lipgloss.NewStyle().
			Foreground(theme.Current.Error).
			Bold(true).
			Padding(2, 4)
		detailStyle := lipgloss.NewStyle().
			Foreground(theme.Current.TextDim).
			Padding(0, 4)

		errContent := errStyle.Render("Failed to load feed") + "\n\n" +
			detailStyle.Render(fmt.Sprintf("Error: %s", msg.err))

		ts.viewport.SetContent(errContent)
		m.tabBar.SetActiveTitle("Error")
		return m, nil
	}

	ts.page = nil // clear page state since this is feed content
	ts.feedLinks = msg.links
	ts.viewport.SetContent(msg.content)
	m.tabBar.SetActiveTitle(msg.title)
	m.statusBar.SetTitle(msg.title)
	m.statusBar.SetMessage("")
	m.statusBar.SetLinkCount(len(msg.links))
	m.syncStatusBar()

	// Record feed page in global history.
	if m.historyStore != nil {
		tab := m.tabBar.ActiveTab()
		if tab != nil && tab.URL != "" {
			m.historyStore.Add(tab.URL, msg.title)
		}
	}

	return m, nil
}

// fetchHN creates a tea.Cmd that fetches HN stories asynchronously.
func (m Model) fetchHN(category string) tea.Cmd {
	tab := m.tabBar.ActiveTab()
	if tab == nil {
		return nil
	}
	tabID := tab.ID
	client := m.hnClient

	return func() tea.Msg {
		var stories []feeds.HNStory
		var err error
		var title string

		switch category {
		case "new":
			title = "Hacker News - New Stories"
			stories, err = client.NewStories(30)
		case "best":
			title = "Hacker News - Best Stories"
			stories, err = client.BestStories(30)
		case "ask":
			title = "Hacker News - Ask HN"
			stories, err = client.AskStories(30)
		case "show":
			title = "Hacker News - Show HN"
			stories, err = client.ShowStories(30)
		default:
			title = "Hacker News - Top Stories"
			stories, err = client.TopStories(30)
		}

		if err != nil {
			return feedLoadedMsg{tabID: tabID, err: err}
		}

		content, links := feeds.RenderHNStories(stories, title)
		return feedLoadedMsg{tabID: tabID, content: content, title: title, links: links}
	}
}

// fetchReddit creates a tea.Cmd that fetches a subreddit asynchronously.
func (m Model) fetchReddit(subreddit string) tea.Cmd {
	tab := m.tabBar.ActiveTab()
	if tab == nil {
		return nil
	}
	tabID := tab.ID
	client := m.redditClient

	return func() tea.Msg {
		posts, err := client.FetchSubreddit(subreddit, "hot", 25)
		if err != nil {
			return feedLoadedMsg{tabID: tabID, err: err}
		}

		title := fmt.Sprintf("r/%s - Hot", subreddit)
		content, links := feeds.RenderRedditPosts(posts, title)
		return feedLoadedMsg{tabID: tabID, content: content, title: title, links: links}
	}
}

// fetchRSS creates a tea.Cmd that fetches an RSS feed asynchronously.
func (m Model) fetchRSS(feedURL string) tea.Cmd {
	tab := m.tabBar.ActiveTab()
	if tab == nil {
		return nil
	}
	tabID := tab.ID
	client := m.rssClient

	return func() tea.Msg {
		feed, err := client.Fetch(feedURL)
		if err != nil {
			return feedLoadedMsg{tabID: tabID, err: err}
		}

		content, links := feeds.RenderFeed(feed)
		return feedLoadedMsg{tabID: tabID, content: content, title: feed.Title, links: links}
	}
}

// fetchSearch creates a tea.Cmd that searches DuckDuckGo asynchronously.
func (m Model) fetchSearch(query string) tea.Cmd {
	tab := m.tabBar.ActiveTab()
	if tab == nil {
		return nil
	}
	tabID := tab.ID

	return func() tea.Msg {
		results, err := feeds.SearchDDG(query)
		if err != nil {
			return feedLoadedMsg{tabID: tabID, err: err}
		}

		content, links := feeds.RenderSearchResults(results, query)
		title := fmt.Sprintf("Search: %s", query)
		return feedLoadedMsg{tabID: tabID, content: content, title: title, links: links}
	}
}

// showHelp displays the keybinding reference in the viewport.
func (m *Model) showHelp() {
	ts := m.activeTabState()
	if ts == nil {
		return
	}

	t := theme.Current

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Primary).
		MarginBottom(1)

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Accent).
		MarginTop(1)

	keyStyle := lipgloss.NewStyle().
		Foreground(t.Secondary).
		Width(16)

	descStyle := lipgloss.NewStyle().
		Foreground(t.Text)

	var sb strings.Builder

	sb.WriteString(titleStyle.Render("tsurf Keybindings"))
	sb.WriteString("\n\n")

	sections := []struct {
		name string
		keys []struct{ k, d string }
	}{
		{"Navigation", []struct{ k, d string }{
			{"j / Down", "Scroll down"},
			{"k / Up", "Scroll up"},
			{"Ctrl+d", "Half page down"},
			{"Ctrl+u", "Half page up"},
			{"gg", "Go to top"},
			{"G", "Go to bottom"},
		}},
		{"Browsing", []struct{ k, d string }{
			{"o", "Open URL / search"},
			{"f", "Follow link by number"},
			{"H", "Go back in history"},
			{"L", "Go forward in history"},
			{"r", "Reload page"},
			{"B", "Bookmark current page"},
			{"R", "Add to read later"},
			{"Ctrl+h", "Toggle history panel"},
		}},
		{"Tabs", []struct{ k, d string }{
			{"Ctrl+t", "New tab"},
			{"Ctrl+w", "Close tab"},
			{"gt / Tab", "Next tab"},
			{"gT / S-Tab", "Previous tab"},
		}},
		{"Modes", []struct{ k, d string }{
			{":", "Command mode"},
			{"/", "Search on page"},
			{"Space", "Leader key (shortcut palette)"},
			{"?", "Show this help"},
		}},
		{"Commands", []struct{ k, d string }{
			{":open <url>", "Open URL"},
			{":theme <n>", "Change theme"},
			{":tabnew", "New tab"},
			{":tabclose", "Close tab"},
			{":vsplit", "Vertical split"},
			{":hsplit", "Horizontal split"},
			{":unsplit", "Remove split"},
			{":history", "Toggle history panel"},
			{":clearhistory", "Clear all history"},
			{":quit", "Quit tsurf"},
		}},
		{"Feeds & Search", []struct{ k, d string }{
			{":hn [type]", "Hacker News (top/new/best/ask/show)"},
			{":reddit <sub>", "Browse subreddit"},
			{":rss <url>", "Load RSS/Atom feed"},
			{":search <q>", "DuckDuckGo search"},
			{":bookmarks", "List bookmarks"},
			{":readlater", "List read later queue"},
			{":bookmark", "Bookmark current page"},
		}},
		{"Leader Key (Space+...)", []struct{ k, d string }{
			{"Space o", "Open URL"},
			{"Space b", "Back"},
			{"Space f", "Forward"},
			{"Space l", "Follow link"},
			{"Space r", "Reload"},
			{"Space t", "New tab"},
			{"Space w", "Close tab"},
			{"Space n/p", "Next/Prev tab"},
			{"Space h", "Hacker News"},
			{"Space e", "Reddit"},
			{"Space s", "Search"},
			{"Space a", "RSS feed"},
			{"Space B", "Bookmarks"},
			{"Space R", "Read later"},
			{"Space H", "History panel"},
			{"Space T", "Cycle theme"},
			{"Space v", "Split vertical"},
			{"Space ?", "Help"},
		}},
	}

	for _, section := range sections {
		sb.WriteString(sectionStyle.Render(section.name))
		sb.WriteString("\n\n")
		for _, binding := range section.keys {
			sb.WriteString(keyStyle.Render(binding.k))
			sb.WriteString(descStyle.Render(binding.d))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	ts.viewport.SetContent(sb.String())
	m.tabBar.SetActiveTitle("Help - Keybindings")
	m.statusBar.SetTitle("Help - Keybindings")
	m.statusBar.SetLinkCount(0)
}
