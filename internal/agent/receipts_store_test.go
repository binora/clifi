package agent

import (
	"os"
	"testing"
)

func TestReceiptStore_CreateAndClose(t *testing.T) {
	dataDir := t.TempDir()
	store, err := OpenReceiptStore(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if store == nil || store.db == nil {
		t.Fatalf("expected store and db")
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if _, err := os.Stat(dataDir + "/receipts.db"); err != nil {
		t.Fatalf("expected db file: %v", err)
	}
}
