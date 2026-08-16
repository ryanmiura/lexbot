package sqlite

import (
	"database/sql"
	_ "embed"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/001_schema.sql
var schemaSQL string

// Adapter implements the service.UserRepository and service.WordRepository
// interfaces using SQLite. It shares the same database file used by the
// whatsmeow session store.
type Adapter struct {
	db *sql.DB
}

// NewAdapter opens the SQLite database and applies the initial schema.
func NewAdapter(dbPath string) (*Adapter, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on&_journal_mode=WAL", dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &Adapter{db: db}, nil
}
