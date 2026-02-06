package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/vidyasagar/tsurf/internal/browser"
)

// Bookmark represents a saved page.
type Bookmark struct {
	URL       string    `json:"url"`
	Title     string    `json:"title"`
	Tags      []string  `json:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// BookmarkStore manages bookmarks persisted to a JSON file.
type BookmarkStore struct {
	bookmarks []Bookmark
	path      string
}

// NewBookmarkStore creates a bookmark store at the given data directory.
func NewBookmarkStore(dataDir string) (*BookmarkStore, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating data dir: %w", err)
	}

	path := filepath.Join(dataDir, "bookmarks.json")
	bs := &BookmarkStore{path: path}

	if err := bs.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading bookmarks: %w", err)
	}

	return bs, nil
}

// Add adds a bookmark. Returns false if already bookmarked.
func (bs *BookmarkStore) Add(url, title string, tags ...string) bool {
	for _, b := range bs.bookmarks {
		if b.URL == url {
			return false // already exists
		}
	}

	bs.bookmarks = append(bs.bookmarks, Bookmark{
		URL:       url,
		Title:     title,
		Tags:      tags,
		CreatedAt: time.Now(),
	})

	bs.save()
	return true
}

// Remove removes a bookmark by URL. Returns false if not found.
func (bs *BookmarkStore) Remove(url string) bool {
	for i, b := range bs.bookmarks {
		if b.URL == url {
			bs.bookmarks = append(bs.bookmarks[:i], bs.bookmarks[i+1:]...)
			bs.save()
			return true
		}
	}
	return false
}

// Has reports whether a URL is bookmarked.
func (bs *BookmarkStore) Has(url string) bool {
	for _, b := range bs.bookmarks {
		if b.URL == url {
			return true
		}
	}
	return false
}

// List returns all bookmarks, newest first.
func (bs *BookmarkStore) List() []Bookmark {
	result := make([]Bookmark, len(bs.bookmarks))
	copy(result, bs.bookmarks)
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result
}

// Search finds bookmarks matching a query (title or URL contains query).
func (bs *BookmarkStore) Search(query string) []Bookmark {
	var results []Bookmark
	for _, b := range bs.bookmarks {
		if contains(b.Title, query) || contains(b.URL, query) {
			results = append(results, b)
		}
	}
	return results
}

// Count returns the number of bookmarks.
func (bs *BookmarkStore) Count() int {
	return len(bs.bookmarks)
}

func (bs *BookmarkStore) load() error {
	data, err := os.ReadFile(bs.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &bs.bookmarks)
}

func (bs *BookmarkStore) save() error {
	data, err := json.MarshalIndent(bs.bookmarks, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(bs.path, data, 0o644)
}

// RenderBookmarks formats bookmarks for the viewport.
func RenderBookmarks(bookmarks []Bookmark) (string, []browser.Link) {
	var result string
	var links []browser.Link

	result += "  🔖 Bookmarks\n"
	result += "  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n"

	if len(bookmarks) == 0 {
		result += "  No bookmarks yet. Press 'B' to bookmark a page.\n"
		return result, links
	}

	for i, b := range bookmarks {
		idx := i + 1
		result += fmt.Sprintf("  [%d] %s\n", idx, b.Title)
		result += fmt.Sprintf("       %s\n", b.URL)
		if len(b.Tags) > 0 {
			result += fmt.Sprintf("       tags: %s\n", joinStrings(b.Tags, ", "))
		}
		result += fmt.Sprintf("       saved %s\n\n", timeAgoStore(b.CreatedAt))

		links = append(links, browser.Link{
			Index: idx,
			Text:  b.Title,
			URL:   b.URL,
		})
	}

	return result, links
}

func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
