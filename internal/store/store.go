// Package store implements the SQLite-backed dedup store: one row per active
// availability. A row is inserted on first notify, refreshed while in stock, and
// removed when out of stock so a future restock notifies again.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (no CGO)

	"portasplit-monitor/internal/model"
)

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

func (s *Store) Close() error { return s.db.Close() }

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

// Lookup returns the last stored price for a key and whether the key exists.
// The price is nil when the key is unknown or was recorded without a price.
func (s *Store) Lookup(ctx context.Context, key string) (*float64, bool, error) {
	var price sql.NullFloat64
	err := s.db.QueryRowContext(ctx, `SELECT price FROM notifications WHERE key = ?`, key).Scan(&price)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("lookup: %w", err)
	}
	if price.Valid {
		p := price.Float64
		return &p, true, nil
	}
	return nil, true, nil
}

// Touch refreshes last_checked_at and the latest stock/price for a key.
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

func (s *Store) Delete(ctx context.Context, key string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM notifications WHERE key = ?`, key)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}
