package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type flashCoinMirrorConfig struct {
	Name               string   `json:"name"`
	Symbol             string   `json:"symbol"`
	Decimals           int      `json:"decimals"`
	Supply             string   `json:"supply"`
	OriginChain        string   `json:"originChain"`
	MirrorChains       []string `json:"mirrorChains"`
	WrapAmountPerChain string   `json:"wrapAmountPerChain"`
	MirrorMode         string   `json:"mirrorMode"`
	CanonicalOwner     string   `json:"canonicalOwner"`
}

func runFlashCoinMirror(args []string) {
	fs := flag.NewFlagSet("flash-coin-mirror", flag.ExitOnError)
	configPath := fs.String("config", "", "path to flash-coin-mirror.json")
	bridge := fs.String("bridge", "", "bridge URL")
	_ = fs.Parse(args)

	cfgFile := *configPath
	if cfgFile == "" {
		cfgFile = findFlashCoinConfig()
	}
	raw, err := os.ReadFile(cfgFile)
	if err != nil {
		log.Fatalf("read config: %v", err)
	}
	var cfg flashCoinMirrorConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		log.Fatalf("parse config: %v", err)
	}
	if cfg.Name == "" || cfg.Symbol == "" || cfg.Supply == "" {
		log.Fatal("config must include name, symbol, supply")
	}
	if cfg.OriginChain == "" {
		cfg.OriginChain = "onex-mainnet-1"
	}
	if cfg.Decimals == 0 {
		cfg.Decimals = 8
	}
	if cfg.WrapAmountPerChain == "" {
		cfg.WrapAmountPerChain = "1000000000"
	}

	base := bridgeURL(*bridge)
	symbol := strings.ToUpper(cfg.Symbol)

	var tokenID string
	var deploy map[string]interface{}
	if existingID, existing, ok := findExistingPlatformToken(base, cfg.OriginChain, symbol); ok {
		tokenID = existingID
		deploy = existing
		fmt.Printf("Reusing %s (%s) on %s — id %s\n", cfg.Name, symbol, cfg.OriginChain, tokenID)
	} else {
		fmt.Printf("Deploying %s (%s) on %s...\n", cfg.Name, symbol, cfg.OriginChain)
		var code int
		deploy, code = bridgePost(base, "/bridge/platform/deploy", map[string]interface{}{
			"chainId": cfg.OriginChain, "name": cfg.Name, "symbol": symbol,
			"decimals": cfg.Decimals, "supply": cfg.Supply,
		})
		printJSON(deploy, code)
		if code >= 400 {
			os.Exit(1)
		}
		tokenID, _ = deploy["id"].(string)
		if tokenID == "" {
			tokenID = symbol
		}
	}

	results := []map[string]interface{}{{"step": "deploy", "chain": cfg.OriginChain, "token": tokenID, "result": deploy}}
	for _, chain := range cfg.MirrorChains {
		fmt.Printf("Mirroring to %s (wrap %s)...\n", chain, cfg.WrapAmountPerChain)
		wrap, wcode := bridgePost(base, "/bridge/platform/wrap", map[string]string{
			"originChainId": cfg.OriginChain, "originTokenId": tokenID,
			"targetChainId": chain, "amount": cfg.WrapAmountPerChain,
		})
		printJSON(wrap, wcode)
		if wcode >= 400 {
			log.Fatalf("wrap to %s failed", chain)
		}
		results = append(results, map[string]interface{}{"step": "mirror", "chain": chain, "result": wrap})
	}

	summary := map[string]interface{}{
		"name": cfg.Name, "symbol": cfg.Symbol,
		"origin": cfg.OriginChain, "tokenId": tokenID,
		"mirrors": cfg.MirrorChains, "steps": results,
	}
	outPath := filepath.Join(filepath.Dir(cfgFile), "flash-coin-mirror-result.json")
	data, _ := json.MarshalIndent(summary, "", "  ")
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		log.Printf("warning: could not write %s: %v", outPath, err)
	} else {
		fmt.Printf("Saved mirror manifest: %s\n", outPath)
	}
	printJSON(summary, 200)
}

func findExistingPlatformToken(base, chainID, symbol string) (string, map[string]interface{}, bool) {
	out, code := bridgeGet(base, "/bridge/platform/tokens")
	if code >= 400 {
		return "", nil, false
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return "", nil, false
	}
	var tokens []map[string]interface{}
	if err := json.Unmarshal(raw, &tokens); err != nil {
		return "", nil, false
	}
	wantChain := strings.TrimSpace(chainID)
	wantSymbol := strings.ToUpper(strings.TrimSpace(symbol))
	for _, t := range tokens {
		ch, _ := t["chainId"].(string)
		sym, _ := t["symbol"].(string)
		if ch != wantChain || strings.ToUpper(sym) != wantSymbol {
			continue
		}
		id, _ := t["id"].(string)
		if id == "" {
			continue
		}
		return id, t, true
	}
	return "", nil, false
}

func findFlashCoinConfig() string {
	candidates := []string{
		"configs/flash-coin-mirror.json",
		filepath.Join("..", "configs", "flash-coin-mirror.json"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	log.Fatal("flash-coin-mirror.json not found; use -config PATH")
	return ""
}
