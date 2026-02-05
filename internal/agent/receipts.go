package agent

import (
	"database/sql"
	"fmt"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// ReceiptStore persists transaction receipts for later retrieval.
// It is intentionally minimal: append-only table keyed by tx hash + chain.
type ReceiptStore struct {
	db *sql.DB
}

// OpenReceiptStore opens (or creates) the receipt DB under dataDir/receipts.db.
func OpenReceiptStore(dataDir string) (*ReceiptStore, error) {
	dbPath := filepath.Join(dataDir, "receipts.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open receipts db: %w", err)
	}

	if err := ensureSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &ReceiptStore{db: db}, nil
}

func ensureSchema(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS receipts (
	chain TEXT NOT NULL,
	tx_hash TEXT NOT NULL,
	status INTEGER,
	gas_used INTEGER,
	raw_json TEXT,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (chain, tx_hash)
);
`)
	if err != nil {
		return fmt.Errorf("create receipts table: %w", err)
	}
	return nil
}

// Close closes the underlying DB.
func (s *ReceiptStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}
