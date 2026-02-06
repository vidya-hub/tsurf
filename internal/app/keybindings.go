package app

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keybindings for tsurf.
type KeyMap struct {
	// Navigation
	ScrollDown   key.Binding
	ScrollUp     key.Binding
	HalfPageDown key.Binding
	HalfPageUp   key.Binding
	GotoTop      key.Binding
	GotoBottom   key.Binding

	// Browser
	OpenURL    key.Binding
	Back       key.Binding
	Forward    key.Binding
	Reload     key.Binding
	FollowLink key.Binding

	// Tabs
	NewTab   key.Binding
	CloseTab key.Binding
	NextTab  key.Binding
	PrevTab  key.Binding

	// Modes
	CommandMode key.Binding
	SearchMode  key.Binding

	// Actions
	Quit      key.Binding
	Help      key.Binding
	Bookmark  key.Binding
	ReadLater key.Binding

	// Splits
	SplitVertical   key.Binding
	SplitHorizontal key.Binding
	SplitClose      key.Binding
	SplitToggle     key.Binding

	// History
	HistoryToggle key.Binding
}

// DefaultKeyMap returns the default vim-style keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		ScrollDown: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/down", "scroll down"),
		),
		ScrollUp: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/up", "scroll up"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("Ctrl+d", "half page down"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("Ctrl+u", "half page up"),
		),
		GotoTop: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("gg", "go to top"),
		),
		GotoBottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "go to bottom"),
		),
		OpenURL: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "open URL"),
		),
		Back: key.NewBinding(
			key.WithKeys("H"),
			key.WithHelp("H", "go back"),
		),
		Forward: key.NewBinding(
			key.WithKeys("L"),
			key.WithHelp("L", "go forward"),
		),
		Reload: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "reload page"),
		),
		FollowLink: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "follow link"),
		),
		NewTab: key.NewBinding(
			key.WithKeys("ctrl+t"),
			key.WithHelp("Ctrl+t", "new tab"),
		),
		CloseTab: key.NewBinding(
			key.WithKeys("ctrl+w"),
			key.WithHelp("Ctrl+w", "close tab"),
		),
		NextTab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("gt/Tab", "next tab"),
		),
		PrevTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("gT/S-Tab", "prev tab"),
		),
		CommandMode: key.NewBinding(
			key.WithKeys(":"),
			key.WithHelp(":", "command mode"),
		),
		SearchMode: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Bookmark: key.NewBinding(
			key.WithKeys("B"),
			key.WithHelp("B", "bookmark page"),
		),
		ReadLater: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "read later"),
		),
		SplitVertical: key.NewBinding(
			key.WithKeys("ctrl+\\"),
			key.WithHelp("Ctrl+\\", "split vertical"),
		),
		SplitHorizontal: key.NewBinding(
			key.WithKeys("ctrl+_"),
			key.WithHelp("Ctrl+_", "split horizontal"),
		),
		SplitClose: key.NewBinding(
			key.WithKeys("ctrl+x"),
			key.WithHelp("Ctrl+x", "close split"),
		),
		SplitToggle: key.NewBinding(
			key.WithKeys("ctrl+o"),
			key.WithHelp("Ctrl+o", "toggle split"),
		),
		HistoryToggle: key.NewBinding(
			key.WithKeys("ctrl+h"),
			key.WithHelp("Ctrl+h", "toggle history"),
		),
	}
}
