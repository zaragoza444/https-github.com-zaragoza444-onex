package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Server) mergeFlashCoinToken(list []TokenRecord) []TokenRecord {
	book := s.loadFlashMirrorBook()
	addr := strings.TrimSpace(book.CanonicalAddress)
	if addr == "" && len(book.Deployments) > 0 {
		addr = book.Deployments[0].ContractAddress
	}
	if addr == "" {
		return list
	}
	for _, t := range list {
		if strings.EqualFold(t.ContractAddress, addr) {
			return list
		}
	}
	decimals := book.Decimals
	if decimals == 0 {
		decimals = 8
	}
	supply := book.WrapAmountPerChain
	if supply == "" {
		supply = "1000000000"
	}
	return append(list, TokenRecord{
		ContractAddress: addr,
		Name:            book.Name,
		Symbol:          book.Symbol,
		Decimals:        decimals,
		Supply:          supply,
		DeployMethod:    "mirror",
		ChainSlug:       "bsc",
		ChainName:       "BNB Chain",
		ChainID:         56,
		Explorer:        "https://bscscan.com",
		CreatedAt:       time.Now().Unix(),
	})
}

func (s *Server) registerFlashCoinToken() (string, error) {
	book := s.loadFlashMirrorBook()
	addr := strings.TrimSpace(book.CanonicalAddress)
	if addr == "" && len(book.Deployments) > 0 {
		addr = book.Deployments[0].ContractAddress
	}
	if addr == "" {
		return "", nil
	}
	decimals := book.Decimals
	if decimals == 0 {
		decimals = 8
	}
	supply := book.WrapAmountPerChain
	if supply == "" {
		supply = "1000000000"
	}
	rec := TokenRecord{
		ContractAddress: addr,
		Name:            book.Name,
		Symbol:          book.Symbol,
		Decimals:        decimals,
		Supply:          supply,
		DeployMethod:    "mirror",
		ChainSlug:       "bsc",
		ChainName:       "BNB Chain",
		ChainID:         56,
		Explorer:        "https://bscscan.com",
		Creator:         s.cfg.DeployerAddress,
	}
	if err := s.store.Upsert(rec); err != nil {
		return "", err
	}
	return addr, nil
}

func (s *Server) loadLiquidityCopyPreset() map[string]interface{} {
	path := filepath.Join(s.cfg.ConfigDir, "bscscan-1b-usdt-test.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cfg map[string]interface{}
	if json.Unmarshal(raw, &cfg) != nil {
		return nil
	}
	book := s.loadFlashMirrorBook()
	if addr := book.CanonicalAddress; addr != "" {
		cfg["flashCoinAddress"] = addr
	}
	if s.cfg.DeployerAddress != "" {
		cfg["deployerAddress"] = s.cfg.DeployerAddress
	}
	cfg["dex"] = "pancake-v2"
	cfg["router"] = "0x10ED43C718714eb63d5aA57B78B54704E256024E"
	cfg["usdtAddress"] = "0x55d398326f99059fF775485246999027B3197955"
	return cfg
}

func (s *Server) handleFixPending(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	addr, err := s.registerFlashCoinToken()
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}

	book := s.loadFlashMirrorBook()
	if r.Method == http.MethodPost || r.URL.Query().Get("reload") == "1" {
		chains := s.loadChainMeta()
		s.reloadFlashMirrorPayload(ctx, &book, chains, true)
		book.PayloadReloaded = true
		book.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		_ = s.saveFlashMirrorLive(book)
	}

	pending := 0
	live := 0
	for _, d := range book.Deployments {
		if d.VerifiedOnChain || d.Status == "live" {
			live++
		} else {
			pending++
		}
	}

	copyPayload := s.loadLiquidityCopyPreset()
	writeJSON(w, map[string]interface{}{
		"ok":               true,
		"flashCoinAddress": addr,
		"canonicalAddress": book.CanonicalAddress,
		"pendingChains":    pending,
		"liveChains":       live,
		"deployerAddress":  s.cfg.DeployerAddress,
		"poolLiveMode":     poolLiveMode(s.cfg.DeployerKey),
		"liquidityPreset":  copyPayload,
		"liquidityUrl":     "/?view=liquidity&preset=bscscan1b",
		"nextSteps": []string{
			"Fund deployer wallet on BSC (BNB + USDT)",
			"Open Liquidity → BSCScan $1B USDT test",
			"Connect MetaMask → Add liquidity (MetaMask V2)",
			"Mirrors → Reload full details after pool indexes",
		},
	})
}

func (s *Server) handleLiquidityCopy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	preset := s.loadLiquidityCopyPreset()
	if preset == nil {
		writeJSON(w, map[string]string{"error": "liquidity preset config not found"})
		return
	}
	chain := r.URL.Query().Get("chain")
	if chain == "" {
		chain = "bsc"
	}
	token := r.URL.Query().Get("token")
	if token == "" {
		if v, ok := preset["flashCoinAddress"].(string); ok {
			token = v
		}
	}
	tokenAmt := r.URL.Query().Get("tokenAmount")
	if tokenAmt == "" {
		tokenAmt = "1000000000"
	}
	quoteAmt := r.URL.Query().Get("quoteAmount")
	if quoteAmt == "" {
		quoteAmt = tokenAmt
	}
	text := strings.Join([]string{
		"OneX Liquidity — BSC PancakeSwap V2 USDT",
		"Token: " + token,
		"Token amount: " + tokenAmt,
		"USDT amount: " + quoteAmt,
		"Target price: $1 per token",
		"Chain: " + chain,
		"DEX: pancake-v2",
		"Wallet: " + s.cfg.DeployerAddress,
	}, "\n")
	writeJSON(w, map[string]interface{}{
		"preset":      preset,
		"chain":       chain,
		"token":       token,
		"tokenAmount": tokenAmt,
		"quoteAmount": quoteAmt,
		"quote":       "usdt",
		"dex":         "pancake-v2",
		"wallet":      s.cfg.DeployerAddress,
		"copyText":    text,
		"copyJson":    preset,
	})
}
