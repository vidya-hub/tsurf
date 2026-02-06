package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vidyasagar/tsurf/internal/browser"
)

// ReadLaterItem represents a page saved for later reading.
type ReadLaterItem struct {
	URL       string    `json:"url"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	Read      bool      `json:"read"`
}

// ReadLaterStore manages the read-later queue.
type ReadLaterStore struct {
	items []ReadLaterItem
	path  string
}

// NewReadLaterStore creates a read-later store at the given data directory.
func NewReadLaterStore(dataDir string) (*ReadLaterStore, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating data dir: %w", err)
	}

	path := filepath.Join(dataDir, "readlater.json")
	rl := &ReadLaterStore{path: path}

	if err := rl.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading read later: %w", err)
	}

	return rl, nil
}

// Add adds an item to the read-later queue. Returns false if already queued.
func (rl *ReadLaterStore) Add(url, title string) bool {
	for _, item := range rl.items {
		if item.URL == url {
			return false
		}
	}

	rl.items = append(rl.items, ReadLaterItem{
		URL:       url,
		Title:     title,
		CreatedAt: time.Now(),
	})

	rl.save()
	return true
}

// Remove removes an item by URL.
func (rl *ReadLaterStore) Remove(url string) bool {
	for i, item := range rl.items {
		if item.URL == url {
			rl.items = append(rl.items[:i], rl.items[i+1:]...)
			rl.save()
			return true
		}
	}
	return false
}

// MarkRead marks an item as read.
func (rl *ReadLaterStore) MarkRead(url string) {
	for i, item := range rl.items {
		if item.URL == url {
			rl.items[i].Read = true
			rl.save()
			return
		}
	}
}

// ListUnread returns unread items, oldest first.
func (rl *ReadLaterStore) ListUnread() []ReadLaterItem {
	var results []ReadLaterItem
	for _, item := range rl.items {
		if !item.Read {
			results = append(results, item)
		}
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.Before(results[j].CreatedAt)
	})
	return results
}

// ListAll returns all items, newest first.
func (rl *ReadLaterStore) ListAll() []ReadLaterItem {
	result := make([]ReadLaterItem, len(rl.items))
	copy(result, rl.items)
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result
}

// Count returns total items.
func (rl *ReadLaterStore) Count() int {
	return len(rl.items)
}

// UnreadCount returns the number of unread items.
func (rl *ReadLaterStore) UnreadCount() int {
	n := 0
	for _, item := range rl.items {
		if !item.Read {
			n++
		}
	}
	return n
}

func (rl *ReadLaterStore) load() error {
	data, err := os.ReadFile(rl.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &rl.items)
}

func (rl *ReadLaterStore) save() error {
	data, err := json.MarshalIndent(rl.items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(rl.path, data, 0o644)
}

// RenderReadLater formats read-later items for the viewport.
func RenderReadLater(items []ReadLaterItem) (string, []browser.Link) {
	var result string
	var links []browser.Link

	result += "  📚 Read Later\n"
	result += "  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n"

	if len(items) == 0 {
		result += "  No items in read later queue. Press 'R' to add a page.\n"
		return result, links
	}

	for i, item := range items {
		idx := i + 1
		status := "  "
		if item.Read {
			status = "  "
		}
		result += fmt.Sprintf("  [%d]%s %s\n", idx, status, item.Title)
		result += fmt.Sprintf("       %s\n", item.URL)
		result += fmt.Sprintf("       added %s\n\n", timeAgoStore(item.CreatedAt))

		links = append(links, browser.Link{
			Index: idx,
			Text:  item.Title,
			URL:   item.URL,
		})
	}

	return result, links
}

// Shared helpers for the storage package.

func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func timeAgoStore(t time.Time) string {
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
