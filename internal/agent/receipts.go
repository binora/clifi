package agent

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/core/types"

	_ "modernc.org/sqlite"
)

// ReceiptStore persists transaction receipts for later retrieval.
// It is intentionally minimal: append-only table keyed by tx hash + chain.
type ReceiptStore struct {
	db *sql.DB
}

type StoredReceipt struct {
	Chain     string
	TxHash    string
	Status    uint64
	GasUsed   uint64
	RawJSON   string
	CreatedAt time.Time
}

// OpenReceiptStore opens (or creates) the receipt DB under dataDir/receipts.db.
func OpenReceiptStore(dataDir string) (*ReceiptStore, error) {
	dbPath := filepath.Join(dataDir, "receipts.db")
	return OpenReceiptStoreDSN(dbPath)
}

// OpenReceiptStoreDSN opens (or creates) a receipt DB using the given sqlite DSN/path.
// Tests may pass ":memory:" to avoid touching disk.
func OpenReceiptStoreDSN(dsn string) (*ReceiptStore, error) {
	db, err := sql.Open("sqlite", dsn)
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

func (s *ReceiptStore) Upsert(chain string, receipt *types.Receipt) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("receipt store not initialized")
	}
	if chain == "" {
		return fmt.Errorf("chain is required")
	}
	if receipt == nil {
		return fmt.Errorf("receipt is required")
	}

	raw, err := json.Marshal(receipt)
	if err != nil {
		return fmt.Errorf("marshal receipt: %w", err)
	}

	_, err = s.db.Exec(`
INSERT INTO receipts (chain, tx_hash, status, gas_used, raw_json)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(chain, tx_hash) DO UPDATE SET
	status=excluded.status,
	gas_used=excluded.gas_used,
	raw_json=excluded.raw_json
`, chain, receipt.TxHash.Hex(), receipt.Status, receipt.GasUsed, string(raw))
	if err != nil {
		return fmt.Errorf("persist receipt: %w", err)
	}
	return nil
}

func (s *ReceiptStore) Get(chain, txHash string) (*StoredReceipt, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("receipt store not initialized")
	}
	if chain == "" || txHash == "" {
		return nil, fmt.Errorf("chain and tx hash are required")
	}

	var out StoredReceipt
	var created string
	row := s.db.QueryRow(
		`SELECT chain, tx_hash, COALESCE(status, 0), COALESCE(gas_used, 0), COALESCE(raw_json, ''), created_at FROM receipts WHERE chain = ? AND tx_hash = ?`,
		chain, txHash,
	)
	if err := row.Scan(&out.Chain, &out.TxHash, &out.Status, &out.GasUsed, &out.RawJSON, &created); err != nil {
		return nil, err
	}
	if ts, err := time.Parse("2006-01-02 15:04:05", created); err == nil {
		out.CreatedAt = ts
	}
	return &out, nil
}
