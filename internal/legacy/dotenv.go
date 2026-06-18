package legacy

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// LoadDotEnvFile sets env vars from a KEY=VALUE file when not already set.
func LoadDotEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		_ = os.Setenv(key, val)
	}
}

// LoadProjectEnv loads root .env and bsc-launcher/.env (fallback keys).
func LoadProjectEnv(root string) {
	if root == "" {
		return
	}
	LoadDotEnvFile(filepath.Join(root, ".env"))
	LoadDotEnvFile(filepath.Join(root, "bsc-launcher", ".env"))
}
