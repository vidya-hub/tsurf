package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vidyasagar/tsurf/internal/theme"
)

// CommandType identifies the kind of command bar interaction.
type CommandType int

const (
	CommandNone   CommandType = iota
	CommandEx                 // : commands
	CommandSearch             // / search
	CommandFollow             // f link follow
)

// CommandResult is emitted when a command is submitted.
type CommandResult struct {
	Type  CommandType
	Value string
}

// CommandBar handles vim-style : commands, / search, and f link following.
type CommandBar struct {
	input      textinput.Model
	active     bool
	cmdType    CommandType
	width      int
	history    []string
	historyPos int
}

// NewCommandBar creates a new command bar.
func NewCommandBar() CommandBar {
	ti := textinput.New()
	ti.CharLimit = 256

	return CommandBar{
		input:      ti,
		historyPos: -1,
	}
}

// SetWidth sets the command bar width.
func (c *CommandBar) SetWidth(w int) {
	c.width = w
	c.input.Width = w - 4
}

// Open activates the command bar in the given mode.
func (c *CommandBar) Open(ct CommandType) tea.Cmd {
	c.active = true
	c.cmdType = ct
	c.input.Reset()
	c.historyPos = -1

	switch ct {
	case CommandEx:
		c.input.Placeholder = "command..."
		c.input.Prompt = ":"
	case CommandSearch:
		c.input.Placeholder = "search..."
		c.input.Prompt = "/"
	case CommandFollow:
		c.input.Placeholder = "link #..."
		c.input.Prompt = "f"
	}

	return c.input.Focus()
}

// Close deactivates the command bar.
func (c *CommandBar) Close() {
	c.active = false
	c.cmdType = CommandNone
	c.input.Blur()
	c.input.Reset()
}

// IsActive reports whether the command bar is open.
func (c *CommandBar) IsActive() bool {
	return c.active
}

// SetValue sets the text input value (useful for pre-filling commands).
func (c *CommandBar) SetValue(val string) {
	c.input.SetValue(val)
	c.input.SetCursor(len(val))
}

// Type returns the current command type.
func (c *CommandBar) Type() CommandType {
	return c.cmdType
}

// Submit returns the command result and adds to history.
func (c *CommandBar) Submit() CommandResult {
	val := strings.TrimSpace(c.input.Value())
	result := CommandResult{
		Type:  c.cmdType,
		Value: val,
	}

	if val != "" && c.cmdType == CommandEx {
		c.history = append(c.history, val)
	}

	c.Close()
	return result
}

// Update processes messages for the command bar.
func (c *CommandBar) Update(msg tea.Msg) (*CommandBar, tea.Cmd) {
	if !c.active {
		return c, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			c.Close()
			return c, nil
		case tea.KeyEnter:
			// Handled by the parent (app.go) to process the result.
			return c, nil
		case tea.KeyUp:
			// History navigation for ex commands.
			if c.cmdType == CommandEx && len(c.history) > 0 {
				if c.historyPos < len(c.history)-1 {
					c.historyPos++
				}
				c.input.SetValue(c.history[len(c.history)-1-c.historyPos])
			}
			return c, nil
		case tea.KeyDown:
			if c.cmdType == CommandEx && c.historyPos > 0 {
				c.historyPos--
				c.input.SetValue(c.history[len(c.history)-1-c.historyPos])
			} else if c.historyPos == 0 {
				c.historyPos = -1
				c.input.Reset()
			}
			return c, nil
		}
	}

	var cmd tea.Cmd
	c.input, cmd = c.input.Update(msg)
	return c, cmd
}

// View renders the command bar.
func (c *CommandBar) View() string {
	if !c.active {
		return ""
	}

	t := theme.Current

	barStyle := lipgloss.NewStyle().
		Foreground(t.Text).
		Background(t.Surface).
		Width(c.width)

	return barStyle.Render(c.input.View())
}
