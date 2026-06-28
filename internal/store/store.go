// Package store implements the SQLite-backed dedup store.
//
// The store keeps one row per active availability "episode". When an item
// appears for the first time we notify and insert a row. While it remains in
// stock we simply refresh last_checked_at. When it goes out of stock the row
// is removed, so a future restock triggers a fresh notification.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (no CGO)

	"portasplit-monitor/internal/model"
)

// Store wraps a SQLite database used to remember notified availabilities.
type Store struct {
	db *sql.DB
}

// Open opens (or creates) the database at path and ensures the schema exists.
func Open(_ context.Context, path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// SQLite is happiest with a single writer in this low-concurrency app.
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) init() error {
	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA busy_timeout=5000;",
		"PRAGMA synchronous=NORMAL;",
	}
	for _, p := range pragmas {
		if _, err := s.db.Exec(p); err != nil {
			return fmt.Errorf("pragma %q: %w", p, err)
		}
	}
	const ddl = `
CREATE TABLE IF NOT EXISTS notifications (
    key              TEXT    PRIMARY KEY,
    source           TEXT    NOT NULL,
    store_name       TEXT    NOT NULL,
    product          TEXT    NOT NULL,
    stock            INTEGER NOT NULL,
    price            REAL,
    url              TEXT,
    location         TEXT,
    notified_at      INTEGER NOT NULL,
    last_checked_at  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_notifications_source ON notifications(source);`
	if _, err := s.db.Exec(ddl); err != nil {
		return fmt.Errorf("init schema: %w", err)
	}
	return nil
}

// Close releases the database connection.
func (s *Store) Close() error { return s.db.Close() }

// Record inserts a newly-notified availability.
func (s *Store) Record(ctx context.Context, a model.Availability) error {
	now := time.Now().Unix()
	var price any
	if a.Price != nil {
		price = *a.Price
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO notifications
    (key, source, store_name, product, stock, price, url, location, notified_at, last_checked_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.Key, a.Source, a.StoreName, a.ProductName, a.Stock, price, a.URL, a.Location, now, now)
	if err != nil {
		return fmt.Errorf("record: %w", err)
	}
	return nil
}

// Exists reports whether a given key has already been notified.
func (s *Store) Exists(ctx context.Context, key string) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx,
		`SELECT 1 FROM notifications WHERE key = ? LIMIT 1`, key).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("exists: %w", err)
	}
	return true, nil
}

// Touch refreshes last_checked_at (and the latest stock/price) for a key.
func (s *Store) Touch(ctx context.Context, a model.Availability) error {
	now := time.Now().Unix()
	var price any
	if a.Price != nil {
		price = *a.Price
	}
	_, err := s.db.ExecContext(ctx, `
UPDATE notifications
   SET stock = ?, price = ?, last_checked_at = ?
 WHERE key = ?`, a.Stock, price, now, a.Key)
	if err != nil {
		return fmt.Errorf("touch: %w", err)
	}
	return nil
}

// AllKeys returns every key currently tracked in the store.
func (s *Store) AllKeys(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key FROM notifications`)
	if err != nil {
		return nil, fmt.Errorf("all keys: %w", err)
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// Delete removes a key from the store.
func (s *Store) Delete(ctx context.Context, key string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM notifications WHERE key = ?`, key)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}
