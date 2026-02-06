# tsurf

```
  _                    __ 
 | |_ ___ _   _ _ __ / _|
 | __/ __| | | | '__| |_ 
 | |_\__ \ |_| | |  |  _|
  \__|___/\__,_|_|  |_|  
```

**A terminal web browser for developers with vim keybindings**

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

tsurf is a TUI web browser that strips away the noise and renders web content as clean, readable text right in your terminal. Built with the [Charmbracelet](https://charm.sh) ecosystem, it features vim-style navigation, tabs, split panes, feed integration (Hacker News, Reddit, RSS), bookmarks, and 7 color themes.

---

## Features

- **Clean HTML rendering** — HTML to Markdown pipeline via glamour with numbered link references `[1]`, `[2]`, etc.
- **Vim keybindings** — `j`/`k` scroll, `gg`/`G` jump, `f` follow links, `o` open URL, `H`/`L` back/forward
- **7 input modes** — Normal, Insert, Command, Follow, Search, History, Leader
- **Tabs** — `Ctrl+t` new, `Ctrl+w` close, `gt`/`gT` switch
- **Split panes** — `:vsplit`, `:hsplit`, `:unsplit`
- **Leader key (`Space`)** — Centered popup palette with grouped shortcuts, auto-dismisses after 2s
- **Feed integration** — Hacker News (`:hn`), Reddit (`:reddit`), RSS/Atom (`:rss`), DuckDuckGo (`:search`)
- **Reddit support** — Reddit URLs intercepted and rendered via `.json` API with posts and comments
- **Bookmarks & Read Later** — `B` to bookmark, `R` to read later, JSON persistence
- **Browsing history** — `Ctrl+h` toggles scrollable history panel, persistent across sessions (max 1000 entries)
- **7 color themes** — default, gruvbox, catppuccin, nord, dracula, solarized, tokyonight
- **Async loading** — Non-blocking page fetch with loading indicator
- **XDG storage** — Config and data stored in OS-appropriate directories

---

## Installation

### Build from source

```bash
git clone https://github.com/vidya-hub/tsurf.git
cd tsurf
go build -o tsurf ./cmd/tsurf
./tsurf
```

### Using the install script (macOS)

```bash
curl -sSL https://raw.githubusercontent.com/vidya-hub/tsurf/main/install.sh | bash
```

### Go install

```bash
go install github.com/vidyasagar/tsurf/cmd/tsurf@latest
```

---

## Quick Start

```bash
# Open a URL directly
tsurf https://example.com

# Start with a specific theme
tsurf --theme gruvbox

# Launch and open URL via command mode
tsurf
# Then press : and type "open https://news.ycombinator.com"

# Browse Hacker News
tsurf
# Then press : and type "hn"
```

---

## Keybindings

### Navigation

| Key | Action |
|-----|--------|
| `j` / `Down` | Scroll down |
| `k` / `Up` | Scroll up |
| `Ctrl+d` | Half-page down |
| `Ctrl+u` | Half-page up |
| `gg` | Go to top |
| `G` | Go to bottom |

### Browsing

| Key | Action |
|-----|--------|
| `o` | Open URL (enter insert mode) |
| `f` | Follow link by number |
| `H` | Go back |
| `L` | Go forward |
| `r` | Reload page |
| `B` | Bookmark current page |
| `R` | Add to read later |
| `Ctrl+h` | Toggle history panel |

### Tabs

| Key | Action |
|-----|--------|
| `Ctrl+t` | New tab |
| `Ctrl+w` | Close tab |
| `gt` / `Tab` | Next tab |
| `gT` / `Shift+Tab` | Previous tab |

### Modes

| Key | Action |
|-----|--------|
| `:` | Command mode |
| `/` | Search mode |
| `Space` | Leader key palette |
| `?` | Help |
| `Esc` | Return to normal mode |
| `q` | Quit |

---

## Leader Key

Press `Space` in normal mode to open a centered shortcut palette. It auto-dismisses after 2 seconds or on any keypress.

| Navigate | Tabs | Feeds | Tools | Views |
|----------|------|-------|-------|-------|
| `o` Open URL | `t` New tab | `h` Hacker News | `B` Bookmarks | `H` History |
| `b` Back | `w` Close tab | `e` Reddit | `R` Read later | `v` Split vertical |
| `f` Forward | `n` Next tab | `s` Search | `/` Search page | `x` Close split |
| `l` Follow link | `p` Prev tab | `a` RSS feed | `:` Command | `T` Theme cycle |
| `r` Reload | | | | `?` Help |

---

## Commands

Type `:` to enter command mode, then enter a command:

| Command | Description |
|---------|-------------|
| `:open <url>` | Open a URL |
| `:tabnew` | Open a new tab |
| `:tabclose` | Close current tab |
| `:vsplit` | Vertical split |
| `:hsplit` | Horizontal split |
| `:unsplit` | Remove split |
| `:hn [type]` | Hacker News (top/new/best/ask/show) |
| `:reddit <sub>` | Browse a subreddit |
| `:rss <url>` | Load an RSS/Atom feed |
| `:search <query>` | Search with DuckDuckGo |
| `:bookmarks` | List bookmarks |
| `:readlater` | List read later items |
| `:history` | Toggle history panel |
| `:clearhistory` | Clear all history |
| `:theme <name>` | Switch theme |
| `:quit` | Quit tsurf |

---

## Themes

Switch themes with `:theme <name>` or press `Space` then `T` to cycle through them.

| Theme | Description |
|-------|-------------|
| `default` | Clean dark theme |
| `gruvbox` | Retro groove colors |
| `catppuccin` | Soothing pastel theme |
| `nord` | Arctic, north-bluish palette |
| `dracula` | Dark theme with vibrant colors |
| `solarized` | Precision colors for machines and people |
| `tokyonight` | Clean dark theme inspired by Tokyo nights |

---

## Architecture

tsurf is built with the [Bubbletea](https://github.com/charmbracelet/bubbletea) TUI framework and follows the Elm architecture (Model-Update-View).

```
cmd/tsurf/main.go          Entry point, CLI flags
internal/
  app/                      Main application model, keybindings, mode handling
  browser/                  HTTP fetching, HTML extraction (go-readability),
                            Markdown rendering (glamour), per-tab history
  ui/                       UI components: viewport, URL bar, status bar,
                            tab bar, command bar, split pane, history panel,
                            leader palette
  feeds/                    Hacker News, Reddit, RSS/Atom, DuckDuckGo
  storage/                  Bookmarks, read later, config, persistent history
  theme/                    7 color themes with lipgloss styles
```

---

## CLI Flags

```
tsurf [flags] [url]

Flags:
  --theme <name>    Start with a specific theme
  --version         Print version and exit

Arguments:
  url               URL to open on startup
```

---

## Data Storage

tsurf stores data in XDG-compliant directories:

- **macOS**: `~/Library/Application Support/tsurf/`
- **Linux**: `~/.local/share/tsurf/`

Files:
- `bookmarks.json` — Saved bookmarks
- `readlater.json` — Read later list
- `history.json` — Browsing history (max 1000 entries)

---

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## License

MIT License. See [LICENSE](LICENSE) for details.
