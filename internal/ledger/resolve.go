package ledger

import (
	"os"
	"path/filepath"
)

// ResolvePaths fills default production paths (e.g. bundled bank ledger file).
func ResolvePaths(cfg Config, projectRoot string) Config {
	if cfg.BankFile == "" && projectRoot != "" {
		candidate := filepath.Join(projectRoot, "configs", "bank-ledger.example.json")
		if _, err := os.Stat(candidate); err == nil {
			cfg.BankFile = candidate
		}
	}
	return cfg
}
