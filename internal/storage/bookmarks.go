package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/vidyasagar/tsurf/internal/browser"
)

// Bookmark represents a saved page.
type Bookmark struct {
	ID        int64
	URL       string
	Title     string
	Tags      []string
	CreatedAt time.Time
}

// BookmarkStore manages bookmarks persisted in SQLite.
type BookmarkStore struct {
	db *sql.DB
}

// NewBookmarkStore creates a bookmark store using the given database.
func NewBookmarkStore(db *DB) *BookmarkStore {
	return &BookmarkStore{db: db.Conn()}
}

// Add adds a bookmark. Returns false if already bookmarked.
func (bs *BookmarkStore) Add(url, title string, tags ...string) bool {
	tagStr := strings.Join(tags, ",")
	_, err := bs.db.Exec(
		`INSERT OR IGNORE INTO bookmarks (url, title, tags) VALUES (?, ?, ?)`,
		url, title, tagStr,
	)
	if err != nil {
		return false
	}
	// INSERT OR IGNORE returns RowsAffected=0 if it was a duplicate.
	return true
}

// Remove removes a bookmark by URL. Returns false if not found.
func (bs *BookmarkStore) Remove(url string) bool {
	res, err := bs.db.Exec(`DELETE FROM bookmarks WHERE url = ?`, url)
	if err != nil {
		return false
	}
	n, _ := res.RowsAffected()
	return n > 0
}

// Has reports whether a URL is bookmarked.
func (bs *BookmarkStore) Has(url string) bool {
	var count int
	err := bs.db.QueryRow(`SELECT COUNT(*) FROM bookmarks WHERE url = ?`, url).Scan(&count)
	return err == nil && count > 0
}

// List returns all bookmarks, newest first.
func (bs *BookmarkStore) List() []Bookmark {
	rows, err := bs.db.Query(
		`SELECT id, url, title, tags, created_at FROM bookmarks ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanBookmarks(rows)
}

// Search finds bookmarks matching a query (title or URL contains query).
func (bs *BookmarkStore) Search(query string) []Bookmark {
	like := "%" + query + "%"
	rows, err := bs.db.Query(
		`SELECT id, url, title, tags, created_at FROM bookmarks
		 WHERE title LIKE ? OR url LIKE ?
		 ORDER BY created_at DESC`,
		like, like,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanBookmarks(rows)
}

// Count returns the number of bookmarks.
func (bs *BookmarkStore) Count() int {
	var count int
	bs.db.QueryRow(`SELECT COUNT(*) FROM bookmarks`).Scan(&count)
	return count
}

func scanBookmarks(rows *sql.Rows) []Bookmark {
	var bookmarks []Bookmark
	for rows.Next() {
		var b Bookmark
		var tagStr string
		var createdAt string
		if err := rows.Scan(&b.ID, &b.URL, &b.Title, &tagStr, &createdAt); err != nil {
			continue
		}
		if tagStr != "" {
			b.Tags = strings.Split(tagStr, ",")
		}
		b.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		bookmarks = append(bookmarks, b)
	}
	return bookmarks
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
			result += fmt.Sprintf("       tags: %s\n", strings.Join(b.Tags, ", "))
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
