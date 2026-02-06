package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/vidyasagar/tsurf/internal/theme"
)

// SplitDirection defines horizontal or vertical splits.
type SplitDirection int

const (
	SplitNone       SplitDirection = iota
	SplitVertical                          // side by side
	SplitHorizontal                        // top and bottom
)

// SplitPane manages a split view with two content areas.
type SplitPane struct {
	Direction SplitDirection
	Ratio     float64 // 0.0-1.0, proportion of first pane
	Active    int     // 0 = first pane, 1 = second pane
	width     int
	height    int
}

// NewSplitPane creates a split pane (starts with no split).
func NewSplitPane() SplitPane {
	return SplitPane{
		Direction: SplitNone,
		Ratio:     0.5,
		Active:    0,
	}
}

// SetSize updates the split pane dimensions.
func (sp *SplitPane) SetSize(w, h int) {
	sp.width = w
	sp.height = h
}

// IsSplit reports whether the pane is split.
func (sp *SplitPane) IsSplit() bool {
	return sp.Direction != SplitNone
}

// Split activates a split with the given direction.
func (sp *SplitPane) Split(dir SplitDirection) {
	sp.Direction = dir
	sp.Ratio = 0.5
}

// Unsplit removes the split.
func (sp *SplitPane) Unsplit() {
	sp.Direction = SplitNone
	sp.Active = 0
}

// Toggle switches between panes.
func (sp *SplitPane) Toggle() {
	if sp.IsSplit() {
		sp.Active = 1 - sp.Active
	}
}

// FirstPaneDimensions returns the width and height for the first pane.
func (sp *SplitPane) FirstPaneDimensions() (int, int) {
	if !sp.IsSplit() {
		return sp.width, sp.height
	}

	switch sp.Direction {
	case SplitVertical:
		w := int(float64(sp.width) * sp.Ratio) - 1 // -1 for border
		return w, sp.height
	case SplitHorizontal:
		h := int(float64(sp.height) * sp.Ratio) - 1
		return sp.width, h
	default:
		return sp.width, sp.height
	}
}

// SecondPaneDimensions returns the width and height for the second pane.
func (sp *SplitPane) SecondPaneDimensions() (int, int) {
	if !sp.IsSplit() {
		return 0, 0
	}

	switch sp.Direction {
	case SplitVertical:
		w := sp.width - int(float64(sp.width)*sp.Ratio) - 1
		return w, sp.height
	case SplitHorizontal:
		h := sp.height - int(float64(sp.height)*sp.Ratio) - 1
		return sp.width, h
	default:
		return 0, 0
	}
}

// RenderSplit renders two content strings in a split layout.
func (sp *SplitPane) RenderSplit(first, second string) string {
	if !sp.IsSplit() {
		return first
	}

	t := theme.Current

	borderStyle := lipgloss.NewStyle().
		Foreground(t.Border)

	switch sp.Direction {
	case SplitVertical:
		w1, _ := sp.FirstPaneDimensions()
		w2, _ := sp.SecondPaneDimensions()

		leftStyle := lipgloss.NewStyle().
			Width(w1).
			Height(sp.height)

		rightStyle := lipgloss.NewStyle().
			Width(w2).
			Height(sp.height)

		divider := borderStyle.Render("│")

		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			leftStyle.Render(first),
			divider,
			rightStyle.Render(second),
		)

	case SplitHorizontal:
		_, h1 := sp.FirstPaneDimensions()
		_, h2 := sp.SecondPaneDimensions()

		topStyle := lipgloss.NewStyle().
			Width(sp.width).
			Height(h1)

		bottomStyle := lipgloss.NewStyle().
			Width(sp.width).
			Height(h2)

		dividerStr := ""
		for i := 0; i < sp.width; i++ {
			dividerStr += "─"
		}
		divider := borderStyle.Render(dividerStr)

		return lipgloss.JoinVertical(
			lipgloss.Left,
			topStyle.Render(first),
			divider,
			bottomStyle.Render(second),
		)
	}

	return first
}
