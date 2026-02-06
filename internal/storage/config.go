package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Config holds tsurf user configuration.
type Config struct {
	Theme       string   `json:"theme"`
	Homepage    string   `json:"homepage"`
	SearchEngine string  `json:"search_engine"` // "duckduckgo" (only option for now)
	RSSFeeds    []string `json:"rss_feeds"`
	Subreddits  []string `json:"subreddits"`
	path        string
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		Theme:        "default",
		Homepage:     "",
		SearchEngine: "duckduckgo",
		RSSFeeds: []string{
			"https://hnrss.org/frontpage",
			"https://blog.golang.org/feed.atom",
		},
		Subreddits: []string{
			"programming",
			"golang",
			"linux",
		},
	}
}

// LoadConfig loads configuration from the standard config directory.
func LoadConfig() (*Config, error) {
	dir, err := configDir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(dir, "config.json")
	cfg := DefaultConfig()
	cfg.path = path

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Save default config.
			cfg.Save()
			return &cfg, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	cfg.path = path
	return &cfg, nil
}

// Save writes the configuration to disk.
func (c *Config) Save() error {
	if c.path == "" {
		dir, err := configDir()
		if err != nil {
			return err
		}
		c.path = filepath.Join(dir, "config.json")
	}

	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(c.path, data, 0o644)
}

// DataDir returns the data directory for persistent storage.
func DataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home dir: %w", err)
	}

	var dir string
	switch runtime.GOOS {
	case "darwin":
		dir = filepath.Join(home, "Library", "Application Support", "tsurf")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData != "" {
			dir = filepath.Join(appData, "tsurf")
		} else {
			dir = filepath.Join(home, ".tsurf")
		}
	default: // Linux, BSD, etc.
		xdgData := os.Getenv("XDG_DATA_HOME")
		if xdgData != "" {
			dir = filepath.Join(xdgData, "tsurf")
		} else {
			dir = filepath.Join(home, ".local", "share", "tsurf")
		}
	}

	return dir, nil
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home dir: %w", err)
	}

	var dir string
	switch runtime.GOOS {
	case "darwin":
		dir = filepath.Join(home, "Library", "Application Support", "tsurf")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData != "" {
			dir = filepath.Join(appData, "tsurf")
		} else {
			dir = filepath.Join(home, ".tsurf")
		}
	default:
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig != "" {
			dir = filepath.Join(xdgConfig, "tsurf")
		} else {
			dir = filepath.Join(home, ".config", "tsurf")
		}
	}

	return dir, nil
}
