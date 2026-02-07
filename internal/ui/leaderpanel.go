package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
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

	dimStyle := lipgloss.NewStyle().
		Foreground(t.TextDim).
		Italic(true)

	separatorStyle := lipgloss.NewStyle().
		Foreground(t.Border)

	// Per-group table cell styles.
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Accent).
		Padding(0, 1).
		Align(lipgloss.Center)

	keyCellStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Background).
		Background(t.Secondary).
		Padding(0, 1).
		Align(lipgloss.Center)

	descCellStyle := lipgloss.NewStyle().
		Foreground(t.Text).
		PaddingRight(1)

	// ── Find max rows for uniform height ────────────────────────

	maxRows := 0
	for _, g := range lp.groups {
		if len(g.Bindings) > maxRows {
			maxRows = len(g.Bindings)
		}
	}

	// ── Build one table per group ───────────────────────────────

	groupTables := make([]string, len(lp.groups))
	for gi, group := range lp.groups {
		tbl := table.New().
			Headers(group.Icon+" "+group.Name, "").
			Border(lipgloss.HiddenBorder()).
			BorderColumn(false).
			BorderHeader(true).
			BorderRow(false).
			BorderStyle(separatorStyle).
			StyleFunc(func(row, col int) lipgloss.Style {
				if row == table.HeaderRow {
					return headerStyle
				}
				if col == 0 {
					return keyCellStyle
				}
				return descCellStyle
			})

		// Add binding rows.
		for _, b := range group.Bindings {
			tbl.Row(b.Key, b.Desc)
		}
		// Pad with empty rows so all groups have the same height.
		for j := len(group.Bindings); j < maxRows; j++ {
			tbl.Row("", "")
		}

		groupTables[gi] = tbl.Render()
	}

	// ── Join group tables horizontally with vertical separators ─

	var parts []string
	for i, gt := range groupTables {
		parts = append(parts, gt)
		if i < len(groupTables)-1 {
			h := lipgloss.Height(gt)
			sep := make([]string, h)
			for s := range sep {
				sep[s] = separatorStyle.Render(" │ ")
			}
			parts = append(parts, strings.Join(sep, "\n"))
		}
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top, parts...)

	// ── Assemble final content ──────────────────────────────────

	bodyWidth := lipgloss.Width(body)

	headerLine := titleStyle.Render("⚡ Leader Key")
	rule := separatorStyle.Render(strings.Repeat("─", bodyWidth))

	footerText := dimStyle.Render("press a key or Esc to dismiss")
	footerLine := lipgloss.PlaceHorizontal(bodyWidth, lipgloss.Center, footerText)

	content := lipgloss.JoinVertical(lipgloss.Left,
		headerLine,
		rule,
		"",
		body,
		"",
		rule,
		footerLine,
	)

	// ── Outer box ───────────────────────────────────────────────

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Padding(1, 2)

	return boxStyle.Render(content)
}
