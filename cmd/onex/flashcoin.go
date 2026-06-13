package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type flashCoinMirrorConfig struct {
	Name               string   `json:"name"`
	Symbol             string   `json:"symbol"`
	Decimals           int      `json:"decimals"`
	Supply             string   `json:"supply"`
	OriginChain        string   `json:"originChain"`
	MirrorChains       []string `json:"mirrorChains"`
	WrapAmountPerChain string   `json:"wrapAmountPerChain"`
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
		cfg.WrapAmountPerChain = "10000000000"
	}

	base := bridgeURL(*bridge)
	fmt.Printf("Deploying %s (%s) on %s...\n", cfg.Name, cfg.Symbol, cfg.OriginChain)
	deploy, code := bridgePost(base, "/bridge/platform/deploy", map[string]interface{}{
		"chainId": cfg.OriginChain, "name": cfg.Name, "symbol": cfg.Symbol,
		"decimals": cfg.Decimals, "supply": cfg.Supply,
	})
	printJSON(deploy, code)
	if code >= 400 {
		os.Exit(1)
	}
	tokenID, _ := deploy["id"].(string)
	if tokenID == "" {
		tokenID = cfg.Symbol
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
