package ledger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePathsBankFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "configs")
	if err := os.MkdirAll(cfgPath, 0o755); err != nil {
		t.Fatal(err)
	}
	bank := filepath.Join(cfgPath, "bank-ledger.example.json")
	if err := os.WriteFile(bank, []byte(`{"accounts":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := ResolvePaths(Config{}, dir)
	if cfg.BankFile != bank {
		t.Fatalf("expected %s got %s", bank, cfg.BankFile)
	}
}
