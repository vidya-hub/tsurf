package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/vidyasagar/tsurf/internal/theme"
)

// StatusBar shows the current page info at the bottom of the screen.
type StatusBar struct {
	url        string
	title      string
	loading    bool
	scrollInfo string
	mode       string
	linkCount  int
	width      int
	message    string // temporary status message
}

// NewStatusBar creates a new status bar.
func NewStatusBar() StatusBar {
	return StatusBar{
		mode: "NORMAL",
	}
}

// SetWidth sets the status bar width.
func (s *StatusBar) SetWidth(w int) {
	s.width = w
}

// SetURL updates the displayed URL.
func (s *StatusBar) SetURL(url string) {
	s.url = url
}

// SetTitle updates the page title.
func (s *StatusBar) SetTitle(title string) {
	s.title = title
}

// SetLoading sets the loading indicator state.
func (s *StatusBar) SetLoading(loading bool) {
	s.loading = loading
}

// SetScrollInfo sets the scroll position string (e.g. "42%", "TOP", "BOT").
func (s *StatusBar) SetScrollInfo(info string) {
	s.scrollInfo = info
}

// SetMode sets the current mode indicator (NORMAL, INSERT, COMMAND, etc).
func (s *StatusBar) SetMode(mode string) {
	s.mode = mode
}

// SetLinkCount sets the total link count displayed.
func (s *StatusBar) SetLinkCount(n int) {
	s.linkCount = n
}

// SetMessage sets a temporary status message.
func (s *StatusBar) SetMessage(msg string) {
	s.message = msg
}

// View renders the status bar.
func (s *StatusBar) View() string {
	t := theme.Current

	modeStyle := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1)

	switch s.mode {
	case "NORMAL":
		modeStyle = modeStyle.
			Foreground(t.Background).
			Background(t.Primary)
	case "INSERT":
		modeStyle = modeStyle.
			Foreground(t.Background).
			Background(t.Success)
	case "COMMAND":
		modeStyle = modeStyle.
			Foreground(t.Background).
			Background(t.Accent)
	case "FOLLOW":
		modeStyle = modeStyle.
			Foreground(t.Background).
			Background(t.Link)
	case "SEARCH":
		modeStyle = modeStyle.
			Foreground(t.Background).
			Background(t.Warning)
	case "HISTORY":
		modeStyle = modeStyle.
			Foreground(t.Background).
			Background(t.Secondary)
	case "LEADER":
		modeStyle = modeStyle.
			Foreground(t.Background).
			Background(t.Primary)
	default:
		modeStyle = modeStyle.
			Foreground(t.Background).
			Background(t.Secondary)
	}

	mode := modeStyle.Render(s.mode)

	// Add mode icon.
	var modeIcon string
	switch s.mode {
	case "NORMAL":
		modeIcon = "👁 "
	case "INSERT":
		modeIcon = "✏ "
	case "COMMAND":
		modeIcon = "⌘ "
	case "FOLLOW":
		modeIcon = "🔗 "
	case "SEARCH":
		modeIcon = "🔍 "
	case "HISTORY":
		modeIcon = "📜 "
	case "LEADER":
		modeIcon = "⚡ "
	}
	mode = modeStyle.Render(modeIcon + s.mode)

	barStyle := lipgloss.NewStyle().
		Foreground(t.Text).
		Background(t.Surface)

	// Left side: mode + URL/title
	var left string
	if s.loading {
		loadStyle := lipgloss.NewStyle().
			Foreground(t.Warning).
			Background(t.Surface).
			Bold(true).
			Padding(0, 1)
		left = loadStyle.Render("⏳ Loading...")
	} else if s.message != "" {
		msgStyle := lipgloss.NewStyle().
			Foreground(t.Info).
			Background(t.Surface).
			Padding(0, 1)
		left = msgStyle.Render(s.message)
	} else if s.title != "" {
		titleStyle := lipgloss.NewStyle().
			Foreground(t.Text).
			Background(t.Surface).
			Padding(0, 1)
		left = titleStyle.Render(s.title)
	}

	// Right side: link count + scroll position
	var right string
	rightStyle := lipgloss.NewStyle().
		Foreground(t.TextDim).
		Background(t.Surface).
		Padding(0, 1)

	if s.linkCount > 0 {
		right += rightStyle.Render(fmt.Sprintf("🔗 %d links", s.linkCount))
	}

	scrollStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Secondary).
		Background(t.Surface).
		Padding(0, 1)
	right += scrollStyle.Render("📜 " + s.scrollInfo)

	// Calculate spacing.
	modeWidth := lipgloss.Width(mode)
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	spacerWidth := s.width - modeWidth - leftWidth - rightWidth
	if spacerWidth < 0 {
		spacerWidth = 0
	}

	spacerStyle := lipgloss.NewStyle().
		Background(t.Surface)
	spacer := spacerStyle.Render(fmt.Sprintf("%*s", spacerWidth, ""))

	return barStyle.Render(mode + left + spacer + right)
}
