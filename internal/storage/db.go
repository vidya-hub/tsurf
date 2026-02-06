package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB wraps the SQLite database connection.
type DB struct {
	conn *sql.DB
	path string
}

// OpenDB opens (or creates) the tsurf SQLite database in the given data directory.
func OpenDB(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "tsurf.db")

	conn, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Enable WAL mode and foreign keys.
	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("setting WAL mode: %w", err)
	}
	if _, err := conn.Exec("PRAGMA foreign_keys=ON"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	db := &DB{conn: conn, path: dbPath}

	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

// Conn returns the underlying sql.DB for direct queries.
func (db *DB) Conn() *sql.DB {
	return db.conn
}

// migrate creates the schema if it doesn't exist.
func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS bookmarks (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		url        TEXT    NOT NULL UNIQUE,
		title      TEXT    NOT NULL DEFAULT '',
		tags       TEXT    NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS read_later (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		url        TEXT    NOT NULL UNIQUE,
		title      TEXT    NOT NULL DEFAULT '',
		is_read    INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS history (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		url        TEXT    NOT NULL,
		title      TEXT    NOT NULL DEFAULT '',
		visited_at DATETIME NOT NULL DEFAULT (datetime('now'))
	);

	CREATE INDEX IF NOT EXISTS idx_history_visited_at ON history(visited_at DESC);
	CREATE INDEX IF NOT EXISTS idx_history_url ON history(url);
	CREATE INDEX IF NOT EXISTS idx_bookmarks_url ON bookmarks(url);
	CREATE INDEX IF NOT EXISTS idx_read_later_url ON read_later(url);
	`

	_, err := db.conn.Exec(schema)
	return err
}
