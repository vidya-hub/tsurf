package theme

import "github.com/charmbracelet/lipgloss"

// Theme defines the color palette for the TUI.
type Theme struct {
	Name string

	// Core colors
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Accent    lipgloss.Color

	// Text colors
	Text       lipgloss.Color
	TextDim    lipgloss.Color
	TextBright lipgloss.Color

	// UI element colors
	Background  lipgloss.Color
	Surface     lipgloss.Color
	Border      lipgloss.Color
	BorderFocus lipgloss.Color

	// Semantic colors
	Link      lipgloss.Color
	LinkIndex lipgloss.Color
	Heading   lipgloss.Color
	Code      lipgloss.Color
	CodeBg    lipgloss.Color
	Quote     lipgloss.Color
	Error     lipgloss.Color
	Success   lipgloss.Color
	Warning   lipgloss.Color
	Info      lipgloss.Color

	// Tab bar
	TabActive   lipgloss.Color
	TabInactive lipgloss.Color
}

var themes = map[string]Theme{
	"default":    Default,
	"gruvbox":    Gruvbox,
	"catppuccin": Catppuccin,
	"nord":       Nord,
	"dracula":    Dracula,
	"solarized":  Solarized,
	"tokyonight": TokyoNight,
}

var Default = Theme{
	Name:        "default",
	Primary:     lipgloss.Color("#7C3AED"),
	Secondary:   lipgloss.Color("#06B6D4"),
	Accent:      lipgloss.Color("#F59E0B"),
	Text:        lipgloss.Color("#E2E8F0"),
	TextDim:     lipgloss.Color("#64748B"),
	TextBright:  lipgloss.Color("#F8FAFC"),
	Background:  lipgloss.Color("#0F172A"),
	Surface:     lipgloss.Color("#1E293B"),
	Border:      lipgloss.Color("#334155"),
	BorderFocus: lipgloss.Color("#7C3AED"),
	Link:        lipgloss.Color("#38BDF8"),
	LinkIndex:   lipgloss.Color("#F59E0B"),
	Heading:     lipgloss.Color("#A78BFA"),
	Code:        lipgloss.Color("#34D399"),
	CodeBg:      lipgloss.Color("#1E293B"),
	Quote:       lipgloss.Color("#94A3B8"),
	Error:       lipgloss.Color("#EF4444"),
	Success:     lipgloss.Color("#22C55E"),
	Warning:     lipgloss.Color("#F59E0B"),
	Info:        lipgloss.Color("#3B82F6"),
	TabActive:   lipgloss.Color("#7C3AED"),
	TabInactive: lipgloss.Color("#475569"),
}

var Gruvbox = Theme{
	Name:        "gruvbox",
	Primary:     lipgloss.Color("#D65D0E"),
	Secondary:   lipgloss.Color("#458588"),
	Accent:      lipgloss.Color("#D79921"),
	Text:        lipgloss.Color("#EBDBB2"),
	TextDim:     lipgloss.Color("#928374"),
	TextBright:  lipgloss.Color("#FBF1C7"),
	Background:  lipgloss.Color("#282828"),
	Surface:     lipgloss.Color("#3C3836"),
	Border:      lipgloss.Color("#504945"),
	BorderFocus: lipgloss.Color("#D65D0E"),
	Link:        lipgloss.Color("#83A598"),
	LinkIndex:   lipgloss.Color("#FABD2F"),
	Heading:     lipgloss.Color("#FB4934"),
	Code:        lipgloss.Color("#B8BB26"),
	CodeBg:      lipgloss.Color("#3C3836"),
	Quote:       lipgloss.Color("#928374"),
	Error:       lipgloss.Color("#FB4934"),
	Success:     lipgloss.Color("#B8BB26"),
	Warning:     lipgloss.Color("#FABD2F"),
	Info:        lipgloss.Color("#83A598"),
	TabActive:   lipgloss.Color("#D65D0E"),
	TabInactive: lipgloss.Color("#665C54"),
}

var Catppuccin = Theme{
	Name:        "catppuccin",
	Primary:     lipgloss.Color("#CBA6F7"),
	Secondary:   lipgloss.Color("#89DCEB"),
	Accent:      lipgloss.Color("#F9E2AF"),
	Text:        lipgloss.Color("#CDD6F4"),
	TextDim:     lipgloss.Color("#6C7086"),
	TextBright:  lipgloss.Color("#F5E0DC"),
	Background:  lipgloss.Color("#1E1E2E"),
	Surface:     lipgloss.Color("#313244"),
	Border:      lipgloss.Color("#45475A"),
	BorderFocus: lipgloss.Color("#CBA6F7"),
	Link:        lipgloss.Color("#89B4FA"),
	LinkIndex:   lipgloss.Color("#F9E2AF"),
	Heading:     lipgloss.Color("#CBA6F7"),
	Code:        lipgloss.Color("#A6E3A1"),
	CodeBg:      lipgloss.Color("#313244"),
	Quote:       lipgloss.Color("#9399B2"),
	Error:       lipgloss.Color("#F38BA8"),
	Success:     lipgloss.Color("#A6E3A1"),
	Warning:     lipgloss.Color("#F9E2AF"),
	Info:        lipgloss.Color("#89B4FA"),
	TabActive:   lipgloss.Color("#CBA6F7"),
	TabInactive: lipgloss.Color("#585B70"),
}

var Nord = Theme{
	Name:        "nord",
	Primary:     lipgloss.Color("#88C0D0"),
	Secondary:   lipgloss.Color("#81A1C1"),
	Accent:      lipgloss.Color("#EBCB8B"),
	Text:        lipgloss.Color("#ECEFF4"),
	TextDim:     lipgloss.Color("#4C566A"),
	TextBright:  lipgloss.Color("#ECEFF4"),
	Background:  lipgloss.Color("#2E3440"),
	Surface:     lipgloss.Color("#3B4252"),
	Border:      lipgloss.Color("#434C5E"),
	BorderFocus: lipgloss.Color("#88C0D0"),
	Link:        lipgloss.Color("#88C0D0"),
	LinkIndex:   lipgloss.Color("#EBCB8B"),
	Heading:     lipgloss.Color("#81A1C1"),
	Code:        lipgloss.Color("#A3BE8C"),
	CodeBg:      lipgloss.Color("#3B4252"),
	Quote:       lipgloss.Color("#4C566A"),
	Error:       lipgloss.Color("#BF616A"),
	Success:     lipgloss.Color("#A3BE8C"),
	Warning:     lipgloss.Color("#EBCB8B"),
	Info:        lipgloss.Color("#5E81AC"),
	TabActive:   lipgloss.Color("#88C0D0"),
	TabInactive: lipgloss.Color("#4C566A"),
}

var Dracula = Theme{
	Name:        "dracula",
	Primary:     lipgloss.Color("#BD93F9"),
	Secondary:   lipgloss.Color("#8BE9FD"),
	Accent:      lipgloss.Color("#F1FA8C"),
	Text:        lipgloss.Color("#F8F8F2"),
	TextDim:     lipgloss.Color("#6272A4"),
	TextBright:  lipgloss.Color("#F8F8F2"),
	Background:  lipgloss.Color("#282A36"),
	Surface:     lipgloss.Color("#44475A"),
	Border:      lipgloss.Color("#6272A4"),
	BorderFocus: lipgloss.Color("#BD93F9"),
	Link:        lipgloss.Color("#8BE9FD"),
	LinkIndex:   lipgloss.Color("#F1FA8C"),
	Heading:     lipgloss.Color("#FF79C6"),
	Code:        lipgloss.Color("#50FA7B"),
	CodeBg:      lipgloss.Color("#44475A"),
	Quote:       lipgloss.Color("#6272A4"),
	Error:       lipgloss.Color("#FF5555"),
	Success:     lipgloss.Color("#50FA7B"),
	Warning:     lipgloss.Color("#F1FA8C"),
	Info:        lipgloss.Color("#8BE9FD"),
	TabActive:   lipgloss.Color("#BD93F9"),
	TabInactive: lipgloss.Color("#6272A4"),
}

var Solarized = Theme{
	Name:        "solarized",
	Primary:     lipgloss.Color("#268BD2"),
	Secondary:   lipgloss.Color("#2AA198"),
	Accent:      lipgloss.Color("#B58900"),
	Text:        lipgloss.Color("#839496"),
	TextDim:     lipgloss.Color("#586E75"),
	TextBright:  lipgloss.Color("#FDF6E3"),
	Background:  lipgloss.Color("#002B36"),
	Surface:     lipgloss.Color("#073642"),
	Border:      lipgloss.Color("#586E75"),
	BorderFocus: lipgloss.Color("#268BD2"),
	Link:        lipgloss.Color("#268BD2"),
	LinkIndex:   lipgloss.Color("#B58900"),
	Heading:     lipgloss.Color("#CB4B16"),
	Code:        lipgloss.Color("#859900"),
	CodeBg:      lipgloss.Color("#073642"),
	Quote:       lipgloss.Color("#586E75"),
	Error:       lipgloss.Color("#DC322F"),
	Success:     lipgloss.Color("#859900"),
	Warning:     lipgloss.Color("#B58900"),
	Info:        lipgloss.Color("#268BD2"),
	TabActive:   lipgloss.Color("#268BD2"),
	TabInactive: lipgloss.Color("#586E75"),
}

var TokyoNight = Theme{
	Name:        "tokyonight",
	Primary:     lipgloss.Color("#7AA2F7"),
	Secondary:   lipgloss.Color("#7DCFFF"),
	Accent:      lipgloss.Color("#E0AF68"),
	Text:        lipgloss.Color("#C0CAF5"),
	TextDim:     lipgloss.Color("#565F89"),
	TextBright:  lipgloss.Color("#C0CAF5"),
	Background:  lipgloss.Color("#1A1B26"),
	Surface:     lipgloss.Color("#24283B"),
	Border:      lipgloss.Color("#3B4261"),
	BorderFocus: lipgloss.Color("#7AA2F7"),
	Link:        lipgloss.Color("#7DCFFF"),
	LinkIndex:   lipgloss.Color("#E0AF68"),
	Heading:     lipgloss.Color("#BB9AF7"),
	Code:        lipgloss.Color("#9ECE6A"),
	CodeBg:      lipgloss.Color("#24283B"),
	Quote:       lipgloss.Color("#565F89"),
	Error:       lipgloss.Color("#F7768E"),
	Success:     lipgloss.Color("#9ECE6A"),
	Warning:     lipgloss.Color("#E0AF68"),
	Info:        lipgloss.Color("#7AA2F7"),
	TabActive:   lipgloss.Color("#7AA2F7"),
	TabInactive: lipgloss.Color("#3B4261"),
}

// Current is the active theme.
var Current = Default

// Set changes the active theme by name.
func Set(name string) bool {
	if t, ok := themes[name]; ok {
		Current = t
		return true
	}
	return false
}

// List returns all available theme names.
func List() []string {
	names := make([]string, 0, len(themes))
	for name := range themes {
		names = append(names, name)
	}
	return names
}
