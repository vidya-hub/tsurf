package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vidyasagar/tsurf/internal/theme"
)

// PageViewport wraps bubbles/viewport with search highlighting and scroll info.
type PageViewport struct {
	viewport   viewport.Model
	ready      bool
	searchTerm string
	totalLines int
	contentSet bool
}

// NewPageViewport creates a new viewport (dimensions set on first WindowSizeMsg).
func NewPageViewport() PageViewport {
	return PageViewport{}
}

// SetSize updates the viewport dimensions.
func (pv *PageViewport) SetSize(width, height int) {
	if !pv.ready {
		pv.viewport = viewport.New(width, height)
		pv.viewport.MouseWheelEnabled = true
		pv.viewport.MouseWheelDelta = 3
		pv.ready = true
	} else {
		pv.viewport.Width = width
		pv.viewport.Height = height
	}
}

// SetContent replaces the viewport content.
func (pv *PageViewport) SetContent(content string) {
	if !pv.ready {
		return
	}
	pv.viewport.SetContent(content)
	pv.totalLines = strings.Count(content, "\n") + 1
	pv.contentSet = true
	pv.viewport.GotoTop()
}

// Update forwards messages to the viewport.
func (pv *PageViewport) Update(msg tea.Msg) (*PageViewport, tea.Cmd) {
	if !pv.ready {
		return pv, nil
	}
	var cmd tea.Cmd
	pv.viewport, cmd = pv.viewport.Update(msg)
	return pv, cmd
}

// View renders the viewport.
func (pv *PageViewport) View() string {
	if !pv.ready {
		return "\n  Initializing..."
	}
	if !pv.contentSet {
		return pv.renderWelcome()
	}
	return pv.viewport.View()
}

// ScrollPercent returns the scroll percentage.
func (pv *PageViewport) ScrollPercent() float64 {
	if !pv.ready {
		return 0
	}
	return pv.viewport.ScrollPercent()
}

// ScrollInfo returns a string like "42%" or "TOP" or "BOT".
func (pv *PageViewport) ScrollInfo() string {
	pct := pv.ScrollPercent()
	switch {
	case pct <= 0:
		return "TOP"
	case pct >= 1:
		return "BOT"
	default:
		return fmt.Sprintf("%d%%", int(pct*100))
	}
}

// HalfPageDown scrolls down half a page.
func (pv *PageViewport) HalfPageDown() {
	if pv.ready {
		pv.viewport.HalfViewDown()
	}
}

// HalfPageUp scrolls up half a page.
func (pv *PageViewport) HalfPageUp() {
	if pv.ready {
		pv.viewport.HalfViewUp()
	}
}

// LineDown scrolls down one line.
func (pv *PageViewport) LineDown(n int) {
	if pv.ready {
		pv.viewport.LineDown(n)
	}
}

// LineUp scrolls up one line.
func (pv *PageViewport) LineUp(n int) {
	if pv.ready {
		pv.viewport.LineUp(n)
	}
}

// GotoTop scrolls to the top.
func (pv *PageViewport) GotoTop() {
	if pv.ready {
		pv.viewport.GotoTop()
	}
}

// GotoBottom scrolls to the bottom.
func (pv *PageViewport) GotoBottom() {
	if pv.ready {
		pv.viewport.GotoBottom()
	}
}

// Ready reports whether the viewport has been initialized.
func (pv *PageViewport) Ready() bool {
	return pv.ready
}

// Width returns the viewport width.
func (pv *PageViewport) Width() int {
	if !pv.ready {
		return 0
	}
	return pv.viewport.Width
}

// Height returns the viewport height.
func (pv *PageViewport) Height() int {
	if !pv.ready {
		return 0
	}
	return pv.viewport.Height
}

func (pv *PageViewport) renderWelcome() string {
	t := theme.Current

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Primary)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(t.TextDim)

	accentStyle := lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true)

	keyStyle := lipgloss.NewStyle().
		Foreground(t.Secondary)

	descStyle := lipgloss.NewStyle().
		Foreground(t.Text)

	logo := `
  🌊 _                    __ 
    | |_ ___ _   _ _ __ / _|
    | __/ __| | | | '__| |_ 
    | |_\__ \ |_| | |  |  _|
     \__|___/\__,_|_|  |_|  
`

	var sb strings.Builder
	sb.WriteString(titleStyle.Render(logo))
	sb.WriteString("\n")
	sb.WriteString(subtitleStyle.Render("  A terminal web browser for developers"))
	sb.WriteString("\n\n")
	sb.WriteString(accentStyle.Render("  ⌨ Quick Start"))
	sb.WriteString("\n\n")

	shortcuts := []struct {
		key  string
		desc string
	}{
		{"  o", "Open URL / search"},
		{"  f", "Follow link by number"},
		{"  H / L", "Go back / forward"},
		{"  j / k", "Scroll down / up"},
		{"  gg / G", "Top / bottom of page"},
		{"  Ctrl+d/u", "Half page down / up"},
		{"  gt / gT", "Next / previous tab"},
		{"  Ctrl+t", "New tab"},
		{"  Ctrl+w", "Close tab"},
		{"  /", "Search on page"},
		{"  :", "Command mode"},
		{"  ?", "Show all keybindings"},
		{"  q", "Quit"},
	}

	for _, s := range shortcuts {
		sb.WriteString(keyStyle.Render(fmt.Sprintf("  %-14s", s.key)))
		sb.WriteString(descStyle.Render(s.desc))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(subtitleStyle.Render("  Type 'o' to open a URL or search the web"))
	sb.WriteString("\n")

	return sb.String()
}
