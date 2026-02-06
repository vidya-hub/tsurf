package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/vidyasagar/tsurf/internal/theme"
)

// Tab represents a single browser tab.
type Tab struct {
	ID    int
	Title string
	URL   string
}

// TabBar manages and renders browser tabs.
type TabBar struct {
	tabs       []Tab
	active     int
	nextID     int
	width      int
	maxVisible int
}

// NewTabBar creates a tab bar with one initial tab.
func NewTabBar() TabBar {
	tb := TabBar{
		nextID:     1,
		maxVisible: 8,
	}
	tb.tabs = append(tb.tabs, Tab{
		ID:    tb.nextID,
		Title: "New Tab",
	})
	tb.active = 0
	return tb
}

// SetWidth sets the tab bar width.
func (tb *TabBar) SetWidth(w int) {
	tb.width = w
	// Adjust visible tabs based on width.
	tb.maxVisible = w / 20
	if tb.maxVisible < 2 {
		tb.maxVisible = 2
	}
	if tb.maxVisible > 10 {
		tb.maxVisible = 10
	}
}

// NewTab adds a new tab and switches to it. Returns the tab index.
func (tb *TabBar) NewTab() int {
	tb.nextID++
	tab := Tab{
		ID:    tb.nextID,
		Title: "New Tab",
	}
	// Insert after current tab.
	insertAt := tb.active + 1
	if insertAt > len(tb.tabs) {
		insertAt = len(tb.tabs)
	}
	tb.tabs = append(tb.tabs[:insertAt], append([]Tab{tab}, tb.tabs[insertAt:]...)...)
	tb.active = insertAt
	return tb.active
}

// CloseTab closes the tab at the given index.
func (tb *TabBar) CloseTab(idx int) bool {
	if len(tb.tabs) <= 1 {
		return false // don't close the last tab
	}
	if idx < 0 || idx >= len(tb.tabs) {
		return false
	}
	tb.tabs = append(tb.tabs[:idx], tb.tabs[idx+1:]...)
	if tb.active >= len(tb.tabs) {
		tb.active = len(tb.tabs) - 1
	} else if tb.active > idx {
		tb.active--
	}
	return true
}

// CloseCurrentTab closes the active tab.
func (tb *TabBar) CloseCurrentTab() bool {
	return tb.CloseTab(tb.active)
}

// NextTab switches to the next tab.
func (tb *TabBar) NextTab() {
	if len(tb.tabs) > 1 {
		tb.active = (tb.active + 1) % len(tb.tabs)
	}
}

// PrevTab switches to the previous tab.
func (tb *TabBar) PrevTab() {
	if len(tb.tabs) > 1 {
		tb.active--
		if tb.active < 0 {
			tb.active = len(tb.tabs) - 1
		}
	}
}

// Active returns the active tab index.
func (tb *TabBar) Active() int {
	return tb.active
}

// ActiveTab returns the active Tab.
func (tb *TabBar) ActiveTab() *Tab {
	if tb.active >= 0 && tb.active < len(tb.tabs) {
		return &tb.tabs[tb.active]
	}
	return nil
}

// SetActiveTitle sets the title of the active tab.
func (tb *TabBar) SetActiveTitle(title string) {
	if tb.active >= 0 && tb.active < len(tb.tabs) {
		if len(title) > 30 {
			title = title[:27] + "..."
		}
		tb.tabs[tb.active].Title = title
	}
}

// SetActiveURL sets the URL of the active tab.
func (tb *TabBar) SetActiveURL(url string) {
	if tb.active >= 0 && tb.active < len(tb.tabs) {
		tb.tabs[tb.active].URL = url
	}
}

// Count returns the number of tabs.
func (tb *TabBar) Count() int {
	return len(tb.tabs)
}

// View renders the tab bar.
func (tb *TabBar) View() string {
	t := theme.Current

	activeStyle := lipgloss.NewStyle().
		Foreground(t.TextBright).
		Background(t.TabActive).
		Bold(true).
		Padding(0, 1)

	inactiveStyle := lipgloss.NewStyle().
		Foreground(t.TextDim).
		Background(t.TabInactive).
		Padding(0, 1)

	separatorStyle := lipgloss.NewStyle().
		Foreground(t.Border)

	// Determine visible tab range.
	start := 0
	end := len(tb.tabs)
	if end > tb.maxVisible {
		start = tb.active - tb.maxVisible/2
		if start < 0 {
			start = 0
		}
		end = start + tb.maxVisible
		if end > len(tb.tabs) {
			end = len(tb.tabs)
			start = end - tb.maxVisible
			if start < 0 {
				start = 0
			}
		}
	}

	var result string

	// Left overflow indicator.
	if start > 0 {
		overflowStyle := lipgloss.NewStyle().
			Foreground(t.TextDim)
		result += overflowStyle.Render(fmt.Sprintf(" +%d ", start))
	}

	for i := start; i < end; i++ {
		title := tb.tabs[i].Title
		if title == "" {
			title = "📄 New Tab"
		}

		// Truncate long titles.
		maxTitleLen := (tb.width / tb.maxVisible) - 4
		if maxTitleLen < 8 {
			maxTitleLen = 8
		}
		if len(title) > maxTitleLen {
			title = title[:maxTitleLen-3] + "..."
		}

		var label string
		if i == tb.active {
			label = fmt.Sprintf(" 🌐 %s ", title)
		} else {
			label = fmt.Sprintf(" %s ", title)
		}

		if i == tb.active {
			result += activeStyle.Render(label)
		} else {
			result += inactiveStyle.Render(label)
		}

		if i < end-1 {
			result += separatorStyle.Render("|")
		}
	}

	// Right overflow indicator.
	if end < len(tb.tabs) {
		overflowStyle := lipgloss.NewStyle().
			Foreground(t.TextDim)
		result += overflowStyle.Render(fmt.Sprintf(" +%d ", len(tb.tabs)-end))
	}

	// Fill remaining width.
	barStyle := lipgloss.NewStyle().
		Background(t.Surface).
		Width(tb.width)

	return barStyle.Render(result)
}
