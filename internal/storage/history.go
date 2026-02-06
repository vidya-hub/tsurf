package storage

import (
	"database/sql"
	"time"
)

// HistoryEntry represents a single visited page.
type HistoryEntry struct {
	ID        int64
	URL       string
	Title     string
	VisitedAt time.Time
}

// HistoryStore manages persistent browsing history in SQLite.
type HistoryStore struct {
	db      *sql.DB
	maxSize int
}

// NewHistoryStore creates a history store using the given database.
func NewHistoryStore(db *DB) *HistoryStore {
	return &HistoryStore{
		db:      db.Conn(),
		maxSize: 1000,
	}
}

// Add records a page visit. If the URL matches the most recent entry,
// it updates the timestamp instead of creating a duplicate.
func (hs *HistoryStore) Add(url, title string) {
	if url == "" {
		return
	}

	// Check if the most recent entry is the same URL.
	var lastURL string
	err := hs.db.QueryRow(
		`SELECT url FROM history ORDER BY visited_at DESC LIMIT 1`,
	).Scan(&lastURL)

	if err == nil && lastURL == url {
		// Update existing entry.
		hs.db.Exec(
			`UPDATE history SET visited_at = datetime('now'), title = CASE WHEN ? != '' THEN ? ELSE title END
			 WHERE id = (SELECT id FROM history ORDER BY visited_at DESC LIMIT 1)`,
			title, title,
		)
		return
	}

	// Insert new entry.
	hs.db.Exec(
		`INSERT INTO history (url, title) VALUES (?, ?)`,
		url, title,
	)

	// Trim if over max.
	hs.db.Exec(
		`DELETE FROM history WHERE id NOT IN (
			SELECT id FROM history ORDER BY visited_at DESC LIMIT ?
		)`,
		hs.maxSize,
	)
}

// List returns all history entries, newest first.
func (hs *HistoryStore) List() []HistoryEntry {
	rows, err := hs.db.Query(
		`SELECT id, url, title, visited_at FROM history ORDER BY visited_at DESC`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanHistoryEntries(rows)
}

// Search finds entries matching a query in title or URL.
func (hs *HistoryStore) Search(query string) []HistoryEntry {
	like := "%" + query + "%"
	rows, err := hs.db.Query(
		`SELECT id, url, title, visited_at FROM history
		 WHERE title LIKE ? OR url LIKE ?
		 ORDER BY visited_at DESC`,
		like, like,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanHistoryEntries(rows)
}

// Remove deletes a history entry by index (0-based, from newest-first ordering).
func (hs *HistoryStore) Remove(idx int) bool {
	// Get the ID of the entry at the given index.
	var id int64
	err := hs.db.QueryRow(
		`SELECT id FROM history ORDER BY visited_at DESC LIMIT 1 OFFSET ?`,
		idx,
	).Scan(&id)
	if err != nil {
		return false
	}

	res, err := hs.db.Exec(`DELETE FROM history WHERE id = ?`, id)
	if err != nil {
		return false
	}
	n, _ := res.RowsAffected()
	return n > 0
}

// Clear removes all history entries.
func (hs *HistoryStore) Clear() {
	hs.db.Exec(`DELETE FROM history`)
}

// Count returns the number of history entries.
func (hs *HistoryStore) Count() int {
	var count int
	hs.db.QueryRow(`SELECT COUNT(*) FROM history`).Scan(&count)
	return count
}

func scanHistoryEntries(rows *sql.Rows) []HistoryEntry {
	var entries []HistoryEntry
	for rows.Next() {
		var e HistoryEntry
		var visitedAt string
		if err := rows.Scan(&e.ID, &e.URL, &e.Title, &visitedAt); err != nil {
			continue
		}
		e.VisitedAt, _ = time.Parse("2006-01-02 15:04:05", visitedAt)
		entries = append(entries, e)
	}
	return entries
}
