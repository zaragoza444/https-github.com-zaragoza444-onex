package ledger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/onex-blockchain/onex/internal/legacy"
)

const (
	bridge7LocalLedger2026 = "local-ledger-2026"
	bridge7LedgerPro       = "ledger-pro"
	bridge7CryptoLedger    = "crypto-ledger"
)

// Bridge7Config wires external ledger files into the OneX real ledger.
type Bridge7Config struct {
	Enabled         bool
	LocalLedger2026 string
	LedgerPro       string
	CryptoLedger    string
}

// bridge7PathsFile is loaded from configs/bridge7.paths.json (or ONEX_BRIDGE7_PATHS_FILE).
// Environment variables override these values when set.
type bridge7PathsFile struct {
	Enabled         *bool  `json:"enabled"`
	LocalLedger2026 string `json:"localLedger2026"`
	LedgerPro       string `json:"ledgerPro"`
	CryptoLedger    string `json:"cryptoLedger"`
	ProjectRoot     string `json:"projectRoot"`
}

func bridge7PathsFileLocation(root string) string {
	if p := strings.TrimSpace(legacy.EnvOrLegacy("ONEX_BRIDGE7_PATHS_FILE", "SHIVA_BRIDGE7_PATHS_FILE")); p != "" {
		return p
	}
	if root != "" {
		return filepath.Join(root, "configs", "bridge7.paths.json")
	}
	return filepath.Join("configs", "bridge7.paths.json")
}

func bridge7EffectiveRoot(envRoot, pathsFile string, paths bridge7PathsFile) string {
	if envRoot != "" {
		return envRoot
	}
	pr := strings.TrimSpace(paths.ProjectRoot)
	if pr != "" && pr != "." {
		if filepath.IsAbs(pr) {
			return pr
		}
		if pathsFile != "" {
			return resolveBridge7Path(pr, filepath.Dir(pathsFile))
		}
		return pr
	}
	if pathsFile != "" {
		return filepath.Clean(filepath.Join(filepath.Dir(pathsFile), ".."))
	}
	return ""
}

func loadBridge7PathsFile(root string) (bridge7PathsFile, string, bool) {
	path := bridge7PathsFileLocation(root)
	data, err := os.ReadFile(path)
	if err != nil {
		return bridge7PathsFile{}, path, false
	}
	var f bridge7PathsFile
	if err := json.Unmarshal(data, &f); err != nil {
		return bridge7PathsFile{}, path, false
	}
	return f, path, true
}

func resolveBridge7Path(p, root string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	if root == "" {
		root = "."
	}
	return filepath.Clean(filepath.Join(root, filepath.FromSlash(p)))
}

func LoadBridge7Config() Bridge7Config {
	envRoot := strings.TrimSpace(legacy.EnvOrLegacy("ONEX_PROJECT_ROOT", "SHIVA_PROJECT_ROOT"))
	paths, pathsFile, hasPaths := loadBridge7PathsFile(envRoot)
	root := bridge7EffectiveRoot(envRoot, pathsFile, paths)

	v := strings.ToLower(strings.TrimSpace(legacy.EnvOrLegacy("ONEX_BRIDGE7_ENABLED", "SHIVA_BRIDGE7_ENABLED")))
	enabled := v == "1" || v == "true" || v == "on"
	if v == "" && hasPaths && paths.Enabled != nil {
		enabled = *paths.Enabled
	}
	if v == "" && (!hasPaths || paths.Enabled == nil) && LoadConfig().Production() {
		enabled = true
	}

	local := legacy.EnvOrLegacy("ONEX_LOCAL_LEDGER_2026_FILE", "SHIVA_LOCAL_LEDGER_2026_FILE")
	pro := legacy.EnvOrLegacy("ONEX_LEDGER_PRO_FILE", "SHIVA_LEDGER_PRO_FILE")
	crypto := legacy.EnvOrLegacy("ONEX_CRYPTO_LEDGER_FILE", "SHIVA_CRYPTO_LEDGER_FILE")
	if local == "" {
		local = resolveBridge7Path(paths.LocalLedger2026, root)
	}
	if pro == "" {
		pro = resolveBridge7Path(paths.LedgerPro, root)
	}
	if crypto == "" {
		crypto = resolveBridge7Path(paths.CryptoLedger, root)
	}
	if local == "" && root != "" {
		local = filepath.Join(root, "data", "bridge7", "local-ledger-2026.json")
	}
	if pro == "" && root != "" {
		pro = filepath.Join(root, "data", "bridge7", "ledger-pro.json")
	}
	if crypto == "" && root != "" {
		crypto = filepath.Join(root, "data", "bridge7", "crypto-ledger.json")
	}
	if local == "" {
		local = filepath.Join("data", "bridge7", "local-ledger-2026.json")
	}
	if pro == "" {
		pro = filepath.Join("data", "bridge7", "ledger-pro.json")
	}
	if crypto == "" {
		crypto = filepath.Join("data", "bridge7", "crypto-ledger.json")
	}
	return Bridge7Config{
		Enabled: enabled, LocalLedger2026: local, LedgerPro: pro, CryptoLedger: crypto,
	}
}

// Bridge7LedgerSummary describes one connected ledger source.
type Bridge7LedgerSummary struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
	Path     string `json:"path,omitempty"`
	Entries  int    `json:"entries"`
	Loaded   bool   `json:"loaded"`
	Error    string `json:"error,omitempty"`
}

// Bridge7Status reports Bridge7 connectivity.
func Bridge7Status() map[string]interface{} {
	cfg := LoadBridge7Config()
	st := map[string]interface{}{
		"service": "onex-bridge7", "enabled": cfg.Enabled, "provider": "Bridge7",
		"ledgers": []string{bridge7LocalLedger2026, bridge7LedgerPro, bridge7CryptoLedger},
		"api": map[string]string{
			"status": "/bridge/bridge7/status",
			"import": "/bridge/bridge7/import",
			"sync":   "/bridge/bridge7/sync",
		},
	}
	if !cfg.Enabled {
		return st
	}
	summaries := SummarizeBridge7Files(cfg)
	total := 0
	for _, s := range summaries {
		total += s.Entries
	}
	st["sources"] = summaries
	st["entries"] = total
	st["paths"] = map[string]string{
		"localLedger2026": cfg.LocalLedger2026,
		"ledgerPro":       cfg.LedgerPro,
		"cryptoLedger":    cfg.CryptoLedger,
	}
	return st
}

// SummarizeBridge7Files previews each configured ledger file.
func SummarizeBridge7Files(cfg Bridge7Config) []Bridge7LedgerSummary {
	out := make([]Bridge7LedgerSummary, 0, 3)
	add := func(id, path string, parse func([]byte) ([]Entry, error)) {
		sum := Bridge7LedgerSummary{ID: id, Provider: id, Path: path}
		if path == "" {
			sum.Error = "path not configured"
			out = append(out, sum)
			return
		}
		data, err := os.ReadFile(path)
		if err != nil {
			sum.Error = err.Error()
			out = append(out, sum)
			return
		}
		entries, err := parse(data)
		if err != nil {
			sum.Error = err.Error()
			out = append(out, sum)
			return
		}
		sum.Loaded = true
		sum.Entries = len(entries)
		out = append(out, sum)
	}
	add(bridge7LocalLedger2026, cfg.LocalLedger2026, ParseLocalLedger2026)
	add(bridge7LedgerPro, cfg.LedgerPro, ParseLedgerPro)
	add(bridge7CryptoLedger, cfg.CryptoLedger, ParseCryptoLedger)
	return out
}

// LoadBridge7Entries imports all configured Bridge7 ledger files.
func LoadBridge7Entries() ([]Entry, error) {
	cfg := LoadBridge7Config()
	if !cfg.Enabled {
		return nil, fmt.Errorf("bridge7 disabled")
	}
	var all []Entry
	appendFile := func(path string, parse func([]byte) ([]Entry, error), ref string) error {
		if strings.TrimSpace(path) == "" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		rows, err := parse(data)
		if err != nil {
			return err
		}
		for i := range rows {
			rows[i].Reference = ref
			if rows[i].Source == "" {
				rows[i].Source = SourceImport
			}
		}
		all = append(all, rows...)
		return nil
	}
	if err := appendFile(cfg.LocalLedger2026, ParseLocalLedger2026, "bridge7:"+bridge7LocalLedger2026); err != nil {
		return nil, err
	}
	if err := appendFile(cfg.LedgerPro, ParseLedgerPro, "bridge7:"+bridge7LedgerPro); err != nil {
		return nil, err
	}
	if err := appendFile(cfg.CryptoLedger, ParseCryptoLedger, "bridge7:"+bridge7CryptoLedger); err != nil {
		return nil, err
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("no bridge7 ledger files loaded")
	}
	return all, nil
}

// ParseLocalLedger2026 parses the local-ledger-2026 export format.
func ParseLocalLedger2026(data []byte) ([]Entry, error) {
	var f struct {
		Provider string `json:"provider"`
		Year     int    `json:"year"`
		Accounts []struct {
			ID       string            `json:"id"`
			Name     string            `json:"name"`
			FundClass string           `json:"fundClass,omitempty"`
			Balances map[string]string `json:"balances"`
		} `json:"accounts"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	var out []Entry
	for _, acct := range f.Accounts {
		for asset, amt := range acct.Balances {
			asset = strings.ToUpper(strings.TrimSpace(asset))
			amt = strings.TrimSpace(amt)
			if asset == "" || amt == "" {
				continue
			}
			out = append(out, Entry{
				ID: importStableID(asset, acct.ID, len(out)), Source: SourceImport,
				Mode: modeForAsset(asset), Asset: asset, Human: amt,
				Account: acct.Name, FundClass: acct.FundClass, Timestamp: now,
				Reference: "bridge7:" + bridge7LocalLedger2026,
			})
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("local-ledger-2026: no balances")
	}
	return out, nil
}

// ParseLedgerPro parses the ledger-pro professional export format.
func ParseLedgerPro(data []byte) ([]Entry, error) {
	var f struct {
		Provider string `json:"provider"`
		Books    []struct {
			BookID  string `json:"bookId"`
			Entries []struct {
				Asset     string `json:"asset"`
				Amount    string `json:"amount"`
				Account   string `json:"account"`
				FundClass string `json:"fundClass,omitempty"`
			} `json:"entries"`
		} `json:"books"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	var out []Entry
	for _, book := range f.Books {
		for i, row := range book.Entries {
			asset := strings.ToUpper(strings.TrimSpace(row.Asset))
			amt := strings.TrimSpace(row.Amount)
			if asset == "" || amt == "" {
				continue
			}
			acct := strings.TrimSpace(row.Account)
			if acct == "" {
				acct = book.BookID
			}
			out = append(out, Entry{
				ID: importStableID(asset, acct, i), Source: SourceImport,
				Mode: modeForAsset(asset), Asset: asset, Human: amt,
				Account: acct, FundClass: row.FundClass, Timestamp: now,
				Reference: "bridge7:" + bridge7LedgerPro,
			})
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("ledger-pro: no entries")
	}
	return out, nil
}

// ParseCryptoLedger parses the crypto-ledger wallet export format.
func ParseCryptoLedger(data []byte) ([]Entry, error) {
	var f struct {
		Provider string `json:"provider"`
		Wallets  []struct {
			Chain    string            `json:"chain"`
			Address  string            `json:"address"`
			Label    string            `json:"label,omitempty"`
			Balances map[string]string `json:"balances"`
		} `json:"wallets"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	var out []Entry
	for _, w := range f.Wallets {
		chain := strings.TrimSpace(w.Chain)
		addr := strings.TrimSpace(w.Address)
		label := strings.TrimSpace(w.Label)
		if label == "" {
			label = addr
		}
		for asset, amt := range w.Balances {
			asset = strings.ToUpper(strings.TrimSpace(asset))
			amt = strings.TrimSpace(amt)
			if asset == "" || amt == "" {
				continue
			}
			e := Entry{
				ID: importStableID(asset, addr, len(out)), Source: SourceImport,
				Mode: ModeReal, Asset: asset, Human: amt, ChainID: chain,
				Account: label, Timestamp: now, Reference: "bridge7:" + bridge7CryptoLedger,
			}
			if chain != "" {
				e.TokenKey = chain + ":" + asset
			}
			out = append(out, e)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("crypto-ledger: no wallet balances")
	}
	return out, nil
}
