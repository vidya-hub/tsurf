package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// HistoryEntry represents a single visited page.
type HistoryEntry struct {
	URL       string    `json:"url"`
	Title     string    `json:"title"`
	VisitedAt time.Time `json:"visited_at"`
}

// HistoryStore manages persistent browsing history.
type HistoryStore struct {
	entries []HistoryEntry
	path    string
	maxSize int // max number of entries to keep
}

// NewHistoryStore creates a history store at the given data directory.
func NewHistoryStore(dataDir string) (*HistoryStore, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating data dir: %w", err)
	}

	path := filepath.Join(dataDir, "history.json")
	hs := &HistoryStore{
		path:    path,
		maxSize: 1000,
	}

	if err := hs.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading history: %w", err)
	}

	return hs, nil
}

// Add records a page visit. If the URL was already the most recent entry,
// it updates the timestamp instead of creating a duplicate.
func (hs *HistoryStore) Add(url, title string) {
	if url == "" {
		return
	}

	now := time.Now()

	// If the last entry is the same URL, just update the timestamp.
	if len(hs.entries) > 0 && hs.entries[0].URL == url {
		hs.entries[0].VisitedAt = now
		if title != "" {
			hs.entries[0].Title = title
		}
		hs.save()
		return
	}

	entry := HistoryEntry{
		URL:       url,
		Title:     title,
		VisitedAt: now,
	}

	// Prepend (newest first).
	hs.entries = append([]HistoryEntry{entry}, hs.entries...)

	// Trim if over max.
	if len(hs.entries) > hs.maxSize {
		hs.entries = hs.entries[:hs.maxSize]
	}

	hs.save()
}

// List returns all history entries, newest first.
func (hs *HistoryStore) List() []HistoryEntry {
	result := make([]HistoryEntry, len(hs.entries))
	copy(result, hs.entries)
	return result
}

// Search finds entries matching a query in title or URL.
func (hs *HistoryStore) Search(query string) []HistoryEntry {
	var results []HistoryEntry
	for _, e := range hs.entries {
		if contains(e.Title, query) || contains(e.URL, query) {
			results = append(results, e)
		}
	}
	return results
}

// Remove deletes a history entry by index (0-based).
func (hs *HistoryStore) Remove(idx int) bool {
	if idx < 0 || idx >= len(hs.entries) {
		return false
	}
	hs.entries = append(hs.entries[:idx], hs.entries[idx+1:]...)
	hs.save()
	return true
}

// Clear removes all history entries.
func (hs *HistoryStore) Clear() {
	hs.entries = nil
	hs.save()
}

// Count returns the number of history entries.
func (hs *HistoryStore) Count() int {
	return len(hs.entries)
}

// SortByTime sorts entries newest first.
func (hs *HistoryStore) SortByTime() {
	sort.Slice(hs.entries, func(i, j int) bool {
		return hs.entries[i].VisitedAt.After(hs.entries[j].VisitedAt)
	})
}

func (hs *HistoryStore) load() error {
	data, err := os.ReadFile(hs.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &hs.entries)
}

func (hs *HistoryStore) save() error {
	data, err := json.MarshalIndent(hs.entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(hs.path, data, 0o644)
}
