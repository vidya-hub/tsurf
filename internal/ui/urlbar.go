package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vidyasagar/tsurf/internal/theme"
)

// URLBar is the URL input bar at the top of the browser.
type URLBar struct {
	input    textinput.Model
	active   bool
	width    int
}

// NewURLBar creates a new URL bar.
func NewURLBar() URLBar {
	ti := textinput.New()
	ti.Placeholder = "Enter URL or search..."
	ti.CharLimit = 2048
	ti.Width = 60

	return URLBar{
		input: ti,
	}
}

// SetWidth updates the URL bar width.
func (u *URLBar) SetWidth(w int) {
	u.width = w
	u.input.Width = w - 8 // account for prompt and padding
}

// Focus activates the URL bar for input.
func (u *URLBar) Focus() tea.Cmd {
	u.active = true
	return u.input.Focus()
}

// Blur deactivates the URL bar.
func (u *URLBar) Blur() {
	u.active = false
	u.input.Blur()
}

// IsActive reports whether the URL bar is focused.
func (u *URLBar) IsActive() bool {
	return u.active
}

// Value returns the current input text.
func (u *URLBar) Value() string {
	return u.input.Value()
}

// SetValue sets the URL bar text.
func (u *URLBar) SetValue(s string) {
	u.input.SetValue(s)
}

// Reset clears the URL bar.
func (u *URLBar) Reset() {
	u.input.Reset()
}

// Update handles messages for the URL bar.
func (u *URLBar) Update(msg tea.Msg) (*URLBar, tea.Cmd) {
	if !u.active {
		return u, nil
	}
	var cmd tea.Cmd
	u.input, cmd = u.input.Update(msg)
	return u, cmd
}

// View renders the URL bar.
func (u *URLBar) View() string {
	t := theme.Current

	var barStyle lipgloss.Style
	if u.active {
		barStyle = lipgloss.NewStyle().
			Foreground(t.Text).
			Background(t.Surface).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(t.BorderFocus).
			Padding(0, 1).
			Width(u.width - 2)
	} else {
		barStyle = lipgloss.NewStyle().
			Foreground(t.TextDim).
			Background(t.Surface).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(t.Border).
			Padding(0, 1).
			Width(u.width - 2)
	}

	promptStyle := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true)

	content := promptStyle.Render(" ") + " " + u.input.View()

	return barStyle.Render(content)
}
