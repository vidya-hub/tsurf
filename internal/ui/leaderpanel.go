package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vidyasagar/tsurf/internal/theme"
)

// LeaderBinding represents a single leader key shortcut.
type LeaderBinding struct {
	Key  string // the key to press after leader (e.g. "o", "t", "B")
	Desc string // short description
}

// LeaderGroup is a named group of leader shortcuts.
type LeaderGroup struct {
	Name     string
	Icon     string
	Bindings []LeaderBinding
}

// LeaderPanel renders the popup shortcut palette shown after pressing the leader key.
type LeaderPanel struct {
	visible bool
	width   int
	height  int
	groups  []LeaderGroup
}

// NewLeaderPanel creates a leader panel with the default shortcut groups.
func NewLeaderPanel() LeaderPanel {
	return LeaderPanel{
		groups: defaultLeaderGroups(),
	}
}

// defaultLeaderGroups returns the built-in shortcut groups.
func defaultLeaderGroups() []LeaderGroup {
	return []LeaderGroup{
		{
			Name: "Navigate",
			Icon: "🧭",
			Bindings: []LeaderBinding{
				{Key: "o", Desc: "Open URL"},
				{Key: "b", Desc: "Back"},
				{Key: "f", Desc: "Forward"},
				{Key: "l", Desc: "Follow link"},
				{Key: "r", Desc: "Reload"},
			},
		},
		{
			Name: "Tabs",
			Icon: "📑",
			Bindings: []LeaderBinding{
				{Key: "t", Desc: "New tab"},
				{Key: "w", Desc: "Close tab"},
				{Key: "n", Desc: "Next tab"},
				{Key: "p", Desc: "Prev tab"},
			},
		},
		{
			Name: "Feeds",
			Icon: "📡",
			Bindings: []LeaderBinding{
				{Key: "h", Desc: "Hacker News"},
				{Key: "e", Desc: "Reddit"},
				{Key: "s", Desc: "Search"},
				{Key: "a", Desc: "RSS feed"},
			},
		},
		{
			Name: "Tools",
			Icon: "🔧",
			Bindings: []LeaderBinding{
				{Key: "B", Desc: "Bookmarks"},
				{Key: "R", Desc: "Read later"},
				{Key: "/", Desc: "Search page"},
				{Key: ":", Desc: "Command"},
			},
		},
		{
			Name: "Views",
			Icon: "👁",
			Bindings: []LeaderBinding{
				{Key: "H", Desc: "History"},
				{Key: "v", Desc: "Split vertical"},
				{Key: "x", Desc: "Close split"},
				{Key: "T", Desc: "Theme cycle"},
				{Key: "?", Desc: "Help"},
			},
		},
	}
}

// Show makes the panel visible.
func (lp *LeaderPanel) Show() {
	lp.visible = true
}

// Hide closes the panel.
func (lp *LeaderPanel) Hide() {
	lp.visible = false
}

// IsVisible reports whether the panel is shown.
func (lp *LeaderPanel) IsVisible() bool {
	return lp.visible
}

// SetSize sets the available area for rendering.
func (lp *LeaderPanel) SetSize(w, h int) {
	lp.width = w
	lp.height = h
}

// View renders the leader palette as a centered popup overlay.
func (lp *LeaderPanel) View() string {
	if !lp.visible {
		return ""
	}

	t := theme.Current

	// Style definitions.
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Primary).
		Padding(0, 1)

	groupTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Accent)

	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Secondary).
		Background(t.Surface).
		Padding(0, 1).
		Width(3).
		Align(lipgloss.Center)

	descStyle := lipgloss.NewStyle().
		Foreground(t.Text)

	hintStyle := lipgloss.NewStyle().
		Foreground(t.TextDim).
		Italic(true).
		Padding(0, 1)

	// Find max rows across all groups for uniform column height.
	maxRows := 0
	for _, g := range lp.groups {
		if len(g.Bindings) > maxRows {
			maxRows = len(g.Bindings)
		}
	}

	// Fixed column width so all columns align.
	colWidth := 20

	// Render each group as a fixed-width, fixed-height column.
	var columns []string
	for _, group := range lp.groups {
		colStyle := lipgloss.NewStyle().Width(colWidth)

		var sb strings.Builder
		sb.WriteString(groupTitleStyle.Render(group.Icon + " " + group.Name))
		sb.WriteString("\n")

		for _, b := range group.Bindings {
			row := keyStyle.Render(b.Key) + " " + descStyle.Render(b.Desc)
			sb.WriteString(row)
			sb.WriteString("\n")
		}

		// Pad empty rows so all columns are the same height.
		for i := len(group.Bindings); i < maxRows; i++ {
			sb.WriteString("\n")
		}

		columns = append(columns, colStyle.Render(sb.String()))
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top, columns...)

	// Header + body + footer hint.
	header := titleStyle.Render("⚡ Leader Key  ─  press a key or Esc to cancel")
	footer := hintStyle.Render("Space = leader key")

	content := lipgloss.JoinVertical(lipgloss.Left,
		"",
		header,
		"",
		body,
		"",
		footer,
	)

	// Wrap in a bordered box.
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Background(t.Background).
		Padding(1, 2)

	return boxStyle.Render(content)
}
