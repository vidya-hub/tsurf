package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/vidyasagar/tsurf/internal/browser"
)

// ReadLaterItem represents a page saved for later reading.
type ReadLaterItem struct {
	ID        int64
	URL       string
	Title     string
	CreatedAt time.Time
	Read      bool
}

// ReadLaterStore manages the read-later queue in SQLite.
type ReadLaterStore struct {
	db *sql.DB
}

// NewReadLaterStore creates a read-later store using the given database.
func NewReadLaterStore(db *DB) *ReadLaterStore {
	return &ReadLaterStore{db: db.Conn()}
}

// Add adds an item to the read-later queue. Returns false if already queued.
func (rl *ReadLaterStore) Add(url, title string) bool {
	_, err := rl.db.Exec(
		`INSERT OR IGNORE INTO read_later (url, title) VALUES (?, ?)`,
		url, title,
	)
	return err == nil
}

// Remove removes an item by URL.
func (rl *ReadLaterStore) Remove(url string) bool {
	res, err := rl.db.Exec(`DELETE FROM read_later WHERE url = ?`, url)
	if err != nil {
		return false
	}
	n, _ := res.RowsAffected()
	return n > 0
}

// MarkRead marks an item as read.
func (rl *ReadLaterStore) MarkRead(url string) {
	rl.db.Exec(`UPDATE read_later SET is_read = 1 WHERE url = ?`, url)
}

// ListUnread returns unread items, oldest first.
func (rl *ReadLaterStore) ListUnread() []ReadLaterItem {
	rows, err := rl.db.Query(
		`SELECT id, url, title, is_read, created_at FROM read_later
		 WHERE is_read = 0 ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanReadLaterItems(rows)
}

// ListAll returns all items, newest first.
func (rl *ReadLaterStore) ListAll() []ReadLaterItem {
	rows, err := rl.db.Query(
		`SELECT id, url, title, is_read, created_at FROM read_later ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanReadLaterItems(rows)
}

// Count returns total items.
func (rl *ReadLaterStore) Count() int {
	var count int
	rl.db.QueryRow(`SELECT COUNT(*) FROM read_later`).Scan(&count)
	return count
}

// UnreadCount returns the number of unread items.
func (rl *ReadLaterStore) UnreadCount() int {
	var count int
	rl.db.QueryRow(`SELECT COUNT(*) FROM read_later WHERE is_read = 0`).Scan(&count)
	return count
}

func scanReadLaterItems(rows *sql.Rows) []ReadLaterItem {
	var items []ReadLaterItem
	for rows.Next() {
		var item ReadLaterItem
		var isRead int
		var createdAt string
		if err := rows.Scan(&item.ID, &item.URL, &item.Title, &isRead, &createdAt); err != nil {
			continue
		}
		item.Read = isRead == 1
		item.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		items = append(items, item)
	}
	return items
}

// RenderReadLater formats read-later items for the viewport.
func RenderReadLater(items []ReadLaterItem) (string, []browser.Link) {
	var sb strings.Builder
	var links []browser.Link

	sb.WriteString("  📚 Read Later\n")
	sb.WriteString("  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	if len(items) == 0 {
		sb.WriteString("  No items in read later queue. Press 'R' to add a page.\n")
		return sb.String(), links
	}

	for i, item := range items {
		idx := i + 1
		status := "  "
		if item.Read {
			status = "  "
		}
		sb.WriteString(fmt.Sprintf("  [%d]%s %s\n", idx, status, item.Title))
		sb.WriteString(fmt.Sprintf("       %s\n", item.URL))
		sb.WriteString(fmt.Sprintf("       added %s\n\n", timeAgoStore(item.CreatedAt)))

		links = append(links, browser.Link{
			Index: idx,
			Text:  item.Title,
			URL:   item.URL,
		})
	}

	return sb.String(), links
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
