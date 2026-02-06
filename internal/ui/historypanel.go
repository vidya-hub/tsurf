package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/vidyasagar/tsurf/internal/storage"
	"github.com/vidyasagar/tsurf/internal/theme"
)

// HistoryPanel displays a scrollable browsing history list with vim navigation.
type HistoryPanel struct {
	entries  []storage.HistoryEntry
	cursor   int
	offset   int // scroll offset for visible window
	width    int
	height   int
	visible  bool
	lastGKey bool // for gg detection within the panel
}

// NewHistoryPanel creates a new history panel.
func NewHistoryPanel() HistoryPanel {
	return HistoryPanel{}
}

// SetEntries updates the history entries displayed.
func (hp *HistoryPanel) SetEntries(entries []storage.HistoryEntry) {
	hp.entries = entries
	hp.cursor = 0
	hp.offset = 0
}

// SetSize updates the panel dimensions.
func (hp *HistoryPanel) SetSize(w, h int) {
	hp.width = w
	hp.height = h
}

// Show makes the panel visible.
func (hp *HistoryPanel) Show() {
	hp.visible = true
	hp.cursor = 0
	hp.offset = 0
	hp.lastGKey = false
}

// Hide closes the panel.
func (hp *HistoryPanel) Hide() {
	hp.visible = false
	hp.lastGKey = false
}

// IsVisible reports whether the panel is shown.
func (hp *HistoryPanel) IsVisible() bool {
	return hp.visible
}

// Toggle switches visibility.
func (hp *HistoryPanel) Toggle() {
	if hp.visible {
		hp.Hide()
	} else {
		hp.Show()
	}
}

// CursorUp moves the cursor up one entry.
func (hp *HistoryPanel) CursorUp() {
	hp.lastGKey = false
	if hp.cursor > 0 {
		hp.cursor--
		hp.ensureVisible()
	}
}

// CursorDown moves the cursor down one entry.
func (hp *HistoryPanel) CursorDown() {
	hp.lastGKey = false
	if hp.cursor < len(hp.entries)-1 {
		hp.cursor++
		hp.ensureVisible()
	}
}

// GotoTop moves to the first entry.
func (hp *HistoryPanel) GotoTop() {
	hp.lastGKey = false
	hp.cursor = 0
	hp.offset = 0
}

// GotoBottom moves to the last entry.
func (hp *HistoryPanel) GotoBottom() {
	hp.lastGKey = false
	if len(hp.entries) > 0 {
		hp.cursor = len(hp.entries) - 1
		hp.ensureVisible()
	}
}

// HalfPageDown scrolls down half a page.
func (hp *HistoryPanel) HalfPageDown() {
	hp.lastGKey = false
	visible := hp.visibleCount()
	hp.cursor += visible / 2
	if hp.cursor >= len(hp.entries) {
		hp.cursor = len(hp.entries) - 1
	}
	if hp.cursor < 0 {
		hp.cursor = 0
	}
	hp.ensureVisible()
}

// HalfPageUp scrolls up half a page.
func (hp *HistoryPanel) HalfPageUp() {
	hp.lastGKey = false
	visible := hp.visibleCount()
	hp.cursor -= visible / 2
	if hp.cursor < 0 {
		hp.cursor = 0
	}
	hp.ensureVisible()
}

// HandleGKey handles the "g" key for gg detection.
// Returns true if "gg" was completed (go to top).
func (hp *HistoryPanel) HandleGKey() bool {
	if hp.lastGKey {
		hp.GotoTop()
		return true
	}
	hp.lastGKey = true
	return false
}

// ResetGKey resets the g key state (called on any non-g key press).
func (hp *HistoryPanel) ResetGKey() {
	hp.lastGKey = false
}

// SelectedEntry returns the entry at the cursor, or nil if empty.
func (hp *HistoryPanel) SelectedEntry() *storage.HistoryEntry {
	if len(hp.entries) == 0 || hp.cursor < 0 || hp.cursor >= len(hp.entries) {
		return nil
	}
	e := hp.entries[hp.cursor]
	return &e
}

// SelectedIndex returns the cursor index.
func (hp *HistoryPanel) SelectedIndex() int {
	return hp.cursor
}

// RemoveSelected removes the entry at the cursor and adjusts position.
func (hp *HistoryPanel) RemoveSelected() {
	if len(hp.entries) == 0 || hp.cursor < 0 || hp.cursor >= len(hp.entries) {
		return
	}
	hp.entries = append(hp.entries[:hp.cursor], hp.entries[hp.cursor+1:]...)
	if hp.cursor >= len(hp.entries) && hp.cursor > 0 {
		hp.cursor--
	}
	hp.ensureVisible()
}

// visibleCount returns how many entries fit in the visible area.
// Each entry takes 2 lines (title + url), plus we need header space.
func (hp *HistoryPanel) visibleCount() int {
	// 3 lines for header (title + separator + blank), 2 lines per entry
	available := hp.height - 3
	if available <= 0 {
		return 1
	}
	count := available / 2
	if count < 1 {
		count = 1
	}
	return count
}

// ensureVisible adjusts offset so the cursor is within the visible window.
func (hp *HistoryPanel) ensureVisible() {
	visible := hp.visibleCount()
	if hp.cursor < hp.offset {
		hp.offset = hp.cursor
	}
	if hp.cursor >= hp.offset+visible {
		hp.offset = hp.cursor - visible + 1
	}
	if hp.offset < 0 {
		hp.offset = 0
	}
}

// View renders the history panel.
func (hp *HistoryPanel) View() string {
	if !hp.visible {
		return ""
	}

	t := theme.Current

	// Panel container style.
	panelStyle := lipgloss.NewStyle().
		Width(hp.width).
		Height(hp.height).
		Background(t.Background)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Primary).
		Background(t.Surface).
		Width(hp.width).
		Padding(0, 1)

	separatorStyle := lipgloss.NewStyle().
		Foreground(t.Border)

	selectedStyle := lipgloss.NewStyle().
		Foreground(t.TextBright).
		Background(t.TabActive).
		Bold(true).
		Width(hp.width).
		Padding(0, 1)

	selectedURLStyle := lipgloss.NewStyle().
		Foreground(t.Link).
		Background(t.TabActive).
		Width(hp.width).
		Padding(0, 1)

	normalStyle := lipgloss.NewStyle().
		Foreground(t.Text).
		Width(hp.width).
		Padding(0, 1)

	urlStyle := lipgloss.NewStyle().
		Foreground(t.TextDim).
		Width(hp.width).
		Padding(0, 1)

	dimStyle := lipgloss.NewStyle().
		Foreground(t.TextDim).
		Padding(0, 1)

	var sb strings.Builder

	// Header.
	sb.WriteString(titleStyle.Render("📜 History"))
	sb.WriteString("\n")

	sepWidth := hp.width - 2
	if sepWidth < 1 {
		sepWidth = 1
	}
	sb.WriteString(separatorStyle.Render(strings.Repeat("─", sepWidth)))
	sb.WriteString("\n")

	if len(hp.entries) == 0 {
		sb.WriteString(dimStyle.Render("No history yet."))
		sb.WriteString("\n")
		return panelStyle.Render(sb.String())
	}

	// Render visible entries.
	visible := hp.visibleCount()
	end := hp.offset + visible
	if end > len(hp.entries) {
		end = len(hp.entries)
	}

	maxTitleLen := hp.width - 4
	if maxTitleLen < 10 {
		maxTitleLen = 10
	}
	maxURLLen := hp.width - 4
	if maxURLLen < 10 {
		maxURLLen = 10
	}

	for i := hp.offset; i < end; i++ {
		entry := hp.entries[i]

		title := entry.Title
		if title == "" {
			title = entry.URL
		}
		if len(title) > maxTitleLen {
			title = title[:maxTitleLen-3] + "..."
		}

		url := entry.URL
		if len(url) > maxURLLen {
			url = url[:maxURLLen-3] + "..."
		}

		timeStr := timeAgo(entry.VisitedAt)

		if i == hp.cursor {
			sb.WriteString(selectedStyle.Render(fmt.Sprintf("▸ %s", title)))
			sb.WriteString("\n")
			sb.WriteString(selectedURLStyle.Render(fmt.Sprintf("  %s  %s", url, timeStr)))
			sb.WriteString("\n")
		} else {
			sb.WriteString(normalStyle.Render(fmt.Sprintf("  %s", title)))
			sb.WriteString("\n")
			sb.WriteString(urlStyle.Render(fmt.Sprintf("  %s  %s", url, timeStr)))
			sb.WriteString("\n")
		}
	}

	// Footer hint.
	linesUsed := 2 + (end-hp.offset)*2 // header + entries
	remaining := hp.height - linesUsed
	if remaining > 1 {
		// Pad blank lines.
		for i := 0; i < remaining-1; i++ {
			sb.WriteString("\n")
		}
		hintStyle := lipgloss.NewStyle().
			Foreground(t.TextDim).
			Italic(true).
			Padding(0, 1)
		sb.WriteString(hintStyle.Render("j/k:move  Enter:open  d:del  Esc:close"))
	}

	return panelStyle.Render(sb.String())
}

// timeAgo returns a human-readable relative time string.
func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
