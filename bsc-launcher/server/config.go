package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Env             string
	Listen          string
	RPCURL          string
	ChainID         int64
	Explorer        string
	DeployerKey     string
	BSCScanAPIKey   string
	APIKey          string
	CORSOrigins     []string
	DataDir         string
	WebDir          string
	RateLimitPerMin int
	MaxBodyBytes    int64
}

func LoadConfig() Config {
	root := projectRoot()
	loadDotEnv(filepath.Join(root, ".env"))

	dataDir := envOr("BSC_LAUNCHER_DATA_DIR", filepath.Join(root, "data"))
	scanKey := strings.TrimSpace(os.Getenv("BSCSCAN_API_KEY"))
	if scanKey == "" {
		scanKey = strings.TrimSpace(os.Getenv("ETHERSCAN_API_KEY"))
	}

	env := strings.ToLower(envOr("BSC_LAUNCHER_ENV", "development"))
	cors := parseOrigins(envOr("BSC_LAUNCHER_CORS_ORIGINS", ""))
	if env == "production" && len(cors) == 0 {
		cors = []string{} // deny cross-origin writes unless explicitly set
	}
	if env != "production" && len(cors) == 0 {
		cors = []string{"*"}
	}

	rateLimit := 10
	if v := envOr("BSC_LAUNCHER_RATE_LIMIT", ""); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			rateLimit = n
		}
	}

	return Config{
		Env:             env,
		Listen:          envOr("BSC_LAUNCHER_LISTEN", ":9340"),
		RPCURL:          envOr("BSC_RPC_URL", "https://bsc-dataseed.binance.org"),
		ChainID:         56,
		Explorer:        envOr("BSC_EXPLORER_URL", "https://bscscan.com"),
		DeployerKey:     strings.TrimPrefix(strings.TrimSpace(os.Getenv("BSC_DEPLOYER_PRIVATE_KEY")), "0x"),
		BSCScanAPIKey:   scanKey,
		APIKey:          strings.TrimSpace(os.Getenv("BSC_LAUNCHER_API_KEY")),
		CORSOrigins:     cors,
		DataDir:         dataDir,
		WebDir:          filepath.Join(root, "web"),
		RateLimitPerMin:  rateLimit,
		MaxBodyBytes:    1 << 20, // 1 MiB
	}
}

func parseOrigins(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func loadDotEnv(path string) {
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
		val = strings.TrimSpace(val)
		val = strings.Trim(val, `"'`)
		if key == "" {
			continue
		}
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func (c Config) IsProduction() bool {
	return c.Env == "production"
}

func projectRoot() string {
	if v := strings.TrimSpace(os.Getenv("BSC_LAUNCHER_ROOT")); v != "" {
		return v
	}

	if exe, err := os.Executable(); err == nil {
		exeDir, _ := filepath.Abs(filepath.Dir(exe))
		candidates := []string{
			filepath.Join(exeDir, "..", "bsc-launcher"),
			filepath.Join(exeDir, "bsc-launcher"),
			filepath.Join(exeDir, ".."),
		}
		for _, c := range candidates {
			c, _ = filepath.Abs(c)
			if fileExists(filepath.Join(c, "web", "index.html")) {
				return c
			}
		}
	}

	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	if filepath.Base(wd) == "server" {
		return filepath.Dir(wd)
	}
	if fileExists(filepath.Join(wd, "bsc-launcher", "web", "index.html")) {
		return filepath.Join(wd, "bsc-launcher")
	}
	return wd
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
