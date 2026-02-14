// ABOUTME: Database connection management and initialization
// ABOUTME: Provides SQLite connection with FTS5 support for ccvault

package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

// DB wraps the SQLite database connection
type DB struct {
	*sql.DB
	path string
}

// Open opens or creates the ccvault database
func Open(dataDir string) (*DB, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "ccvault.db")

	// Open database with WAL mode for better concurrency
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000", dbPath)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxOpenConns(1) // SQLite works best with single writer
	sqlDB.SetMaxIdleConns(1)

	db := &DB{
		DB:   sqlDB,
		path: dbPath,
	}

	// Initialize schema
	if err := db.init(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return db, nil
}

// init creates the database schema and runs migrations
func (db *DB) init() error {
	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("exec schema: %w", err)
	}

	// Run migrations for columns added after initial schema
	migrations := []string{
		"ALTER TABLE sessions ADD COLUMN has_error BOOLEAN DEFAULT 0",
		"ALTER TABLE sessions ADD COLUMN has_subagent BOOLEAN DEFAULT 0",
	}
	for _, m := range migrations {
		// Ignore "duplicate column" errors — means migration already ran
		_, _ = db.Exec(m)
	}

	// Partial indexes for fast has:error / has:subagent queries
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_sessions_has_error ON sessions(has_error) WHERE has_error = 1",
		"CREATE INDEX IF NOT EXISTS idx_sessions_has_subagent ON sessions(has_subagent) WHERE has_subagent = 1",
	}
	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return fmt.Errorf("create index: %w", err)
		}
	}

	return nil
}

// Path returns the database file path
func (db *DB) Path() string {
	return db.path
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}

// BeginTx starts a new transaction
func (db *DB) BeginTx() (*sql.Tx, error) {
	return db.Begin()
}

// WithTx executes a function within a transaction
func (db *DB) WithTx(fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx()
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}
