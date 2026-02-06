package ui

import (
	"fmt"
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
				{Key: "v", Desc: "Split vert"},
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

	// ── Styles ──────────────────────────────────────────────────

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Primary)

	groupNameStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Accent).
		Underline(true)

	keyBadgeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Background).
		Background(t.Secondary).
		Padding(0, 1)

	descStyle := lipgloss.NewStyle().
		Foreground(t.Text)

	dimStyle := lipgloss.NewStyle().
		Foreground(t.TextDim).
		Italic(true)

	separatorStyle := lipgloss.NewStyle().
		Foreground(t.Border)

	// ── Layout constants ────────────────────────────────────────

	const (
		colWidth = 18 // width of each group column
		keyWidth = 3  // width of the key badge
		gapWidth = 3  // gap between columns (includes separator)
	)

	// Find max rows across all groups for uniform column heights.
	maxRows := 0
	for _, g := range lp.groups {
		if len(g.Bindings) > maxRows {
			maxRows = len(g.Bindings)
		}
	}

	// ── Render each group column ────────────────────────────────

	colStyle := lipgloss.NewStyle().Width(colWidth)

	var columns []string
	for i, group := range lp.groups {
		var lines []string

		// Group header line.
		header := groupNameStyle.Render(group.Icon + " " + group.Name)
		lines = append(lines, header)
		lines = append(lines, "") // blank line after header

		// Binding rows.
		for _, b := range group.Bindings {
			badge := keyBadgeStyle.Render(fmt.Sprintf("%-1s", b.Key))
			desc := descStyle.Render(" " + b.Desc)
			lines = append(lines, badge+desc)
		}

		// Pad to uniform height.
		for j := len(group.Bindings); j < maxRows; j++ {
			lines = append(lines, "")
		}

		col := colStyle.Render(strings.Join(lines, "\n"))
		columns = append(columns, col)

		// Vertical separator between groups.
		if i < len(lp.groups)-1 {
			sepHeight := lipgloss.Height(col)
			var sepLines []string
			for s := 0; s < sepHeight; s++ {
				sepLines = append(sepLines, separatorStyle.Render(" │ "))
			}
			columns = append(columns, strings.Join(sepLines, "\n"))
		}
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top, columns...)

	// ── Assemble final content ──────────────────────────────────

	bodyWidth := lipgloss.Width(body)

	headerText := "⚡ Leader Key"
	headerLine := titleStyle.Render(headerText)

	// Horizontal rule under header, matching body width.
	rule := separatorStyle.Render(strings.Repeat("─", bodyWidth))

	footerText := dimStyle.Render("press a key or Esc to dismiss")
	// Center the footer.
	footerPad := ""
	fw := lipgloss.Width(footerText)
	if fw < bodyWidth {
		footerPad = strings.Repeat(" ", (bodyWidth-fw)/2)
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		headerLine,
		rule,
		"",
		body,
		"",
		rule,
		footerPad+footerText,
	)

	// ── Outer box ───────────────────────────────────────────────

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Padding(1, 2)

	return boxStyle.Render(content)
}
