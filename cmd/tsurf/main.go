package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vidyasagar/tsurf/internal/app"
	"github.com/vidyasagar/tsurf/internal/theme"
)

var (
	version = "0.1.0"
)

func main() {
	var (
		themeName   string
		showVersion bool
	)

	flag.StringVar(&themeName, "theme", "default", "color theme (default, gruvbox, catppuccin, nord, dracula, solarized, tokyonight)")
	flag.BoolVar(&showVersion, "version", false, "show version")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "tsurf - a terminal web browser for developers\n\n")
		fmt.Fprintf(os.Stderr, "Usage: tsurf [flags] [url]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  tsurf                          # start with welcome screen\n")
		fmt.Fprintf(os.Stderr, "  tsurf https://example.com      # open a URL\n")
		fmt.Fprintf(os.Stderr, "  tsurf golang.org               # auto-adds https://\n")
		fmt.Fprintf(os.Stderr, "  tsurf \"how to use goroutines\"   # search DuckDuckGo\n")
		fmt.Fprintf(os.Stderr, "  tsurf --theme catppuccin        # use catppuccin theme\n")
	}
	flag.Parse()

	if showVersion {
		fmt.Printf("tsurf %s\n", version)
		os.Exit(0)
	}

	// Apply theme.
	if !theme.Set(themeName) {
		fmt.Fprintf(os.Stderr, "Unknown theme: %s\nAvailable: default, gruvbox, catppuccin, nord, dracula, solarized, tokyonight\n", themeName)
		os.Exit(1)
	}

	// Get optional URL argument.
	var startURL string
	if flag.NArg() > 0 {
		startURL = flag.Arg(0)
	}

	m := app.New(startURL)
	p := tea.NewProgram(m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
