package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type ListingReadiness struct {
	Score          int      `json:"score"`
	Ready          bool     `json:"ready"`
	Missing        []string `json:"missing"`
	HasLiquidity   bool     `json:"hasLiquidity"`
	LiquidityUSD   float64  `json:"liquidityUsd"`
	HasHolders     bool     `json:"hasHolders"`
	Holders        int      `json:"holders"`
	HasExplorer    bool     `json:"hasExplorer"`
	ContractLive   bool     `json:"contractLive"`
}

type ListingProvider struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	SubmitURL   string `json:"submitUrl"`
	DocsURL     string `json:"docsUrl,omitempty"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
	Notes       string `json:"notes,omitempty"`
}

type DexPoolLink struct {
	DexID       string  `json:"dexId"`
	DexName     string  `json:"dexName"`
	Version     int     `json:"version"`
	PairAddress string  `json:"pairAddress,omitempty"`
	LiquidityUSD float64 `json:"liquidityUsd,omitempty"`
	PriceUSD    float64 `json:"priceUsd,omitempty"`
	SwapURL     string  `json:"swapUrl"`
	InfoURL     string  `json:"infoUrl"`
}

type ListingBridgeResult struct {
	TokenAddress     string            `json:"tokenAddress"`
	ChainSlug        string            `json:"chainSlug"`
	ChainName        string            `json:"chainName"`
	Explorer         string            `json:"explorer"`
	ExplorerTokenURL string            `json:"explorerTokenUrl"`
	Name             string            `json:"name"`
	Symbol           string            `json:"symbol"`
	Readiness        ListingReadiness  `json:"readiness"`
	Providers        []ListingProvider `json:"providers"`
	DexPools         []DexPoolLink     `json:"dexPools"`
	GeneratedAt      string            `json:"generatedAt"`
}

func (s *Server) buildListingBridge(ctx context.Context, chainSlug, tokenAddr, name, symbol string) (ListingBridgeResult, error) {
	chainSlug = normalizeRegistryChain(chainSlug)
	if chainSlug == "" {
		chainSlug = "bsc"
	}
	chain, err := chainBySlug(chainSlug)
	if err != nil {
		if s.dexRegistry != nil {
			if cfg, e := s.dexRegistry.chainConfig(chainSlug); e == nil {
				chain = Chain{Slug: chainSlug, Name: cfg.Name, Explorer: cfg.Explorer, DexChainID: cfg.DexChainID}
			}
		}
		if chain.Slug == "" {
			return ListingBridgeResult{}, err
		}
	}

	tokenAddr = strings.TrimSpace(tokenAddr)
	if tokenAddr == "" {
		return ListingBridgeResult{}, fmt.Errorf("token address required")
	}

	res := ListingBridgeResult{
		TokenAddress:     tokenAddr,
		ChainSlug:        chainSlug,
		ChainName:        chain.Name,
		Explorer:         chain.Explorer,
		ExplorerTokenURL: explorerTokenURL(chain.Explorer, tokenAddr),
		Name:             name,
		Symbol:           symbol,
		GeneratedAt:      time.Now().UTC().Format(time.RFC3339),
	}

	dexChain := chain.DexChainID
	if dexChain == "" {
		dexChain = chainSlug
	}
	if s.dexRegistry != nil {
		if cfg, e := s.dexRegistry.chainConfig(chainSlug); e == nil {
			dexChain = cfg.DexChainID
			if res.Explorer == "" {
				res.Explorer = cfg.Explorer
				res.ExplorerTokenURL = explorerTokenURL(cfg.Explorer, tokenAddr)
			}
		}
	}

	var price *PriceQuote
	if s.price != nil {
		price, _ = s.price.Quote(dexChain, tokenAddr)
	}
	var holders int
	if s.bscscan != nil && chain.ChainID > 0 {
		if info, err := s.bscscan.TokenInfoForChain(chain.ChainID, tokenAddr); err == nil && info != nil {
			if n, e := strconv.Atoi(strings.TrimSpace(info.Holders)); e == nil {
				holders = n
			}
			if res.Name == "" {
				res.Name = info.TokenName
			}
			if res.Symbol == "" {
				res.Symbol = info.Symbol
			}
		}
	}
	if chain.ChainID > 0 && chain.RPCURL != "" {
		ok, _ := s.isContractOn(ctx, chain.RPCURL, tokenAddr)
		res.Readiness.ContractLive = ok
	}

	res.Readiness.HasLiquidity = price != nil && price.HasLiquidity
	if price != nil {
		res.Readiness.LiquidityUSD = price.LiquidityUSD
	}
	res.Readiness.HasHolders = holders > 0
	res.Readiness.Holders = holders
	res.Readiness.HasExplorer = res.Explorer != ""

	res.DexPools = s.dexPoolsForToken(chainSlug, tokenAddr, price)
	res.Readiness = scoreReadiness(res.Readiness)
	res.Providers = s.listingProviders(res)
	return res, nil
}

func scoreReadiness(r ListingReadiness) ListingReadiness {
	score := 0
	var missing []string
	if r.ContractLive {
		score += 25
	} else {
		missing = append(missing, "contract bytecode on-chain")
	}
	if r.HasLiquidity && r.LiquidityUSD >= 1000 {
		score += 35
	} else if r.HasLiquidity {
		score += 15
		missing = append(missing, "liquidity ≥ $1,000 recommended")
	} else {
		missing = append(missing, "DEX liquidity pool")
	}
	if r.HasHolders && r.Holders >= 10 {
		score += 20
	} else if r.HasHolders {
		score += 10
		missing = append(missing, "more holders recommended (10+)")
	} else {
		missing = append(missing, "token holders")
	}
	if r.HasExplorer {
		score += 20
	}
	r.Score = score
	r.Ready = score >= 70 && r.HasLiquidity && r.ContractLive
	r.Missing = missing
	return r
}

func (s *Server) dexPoolsForToken(chainSlug, tokenAddr string, price *PriceQuote) []DexPoolLink {
	if s.dexRegistry == nil {
		return nil
	}
	cfg, err := s.dexRegistry.chainConfig(chainSlug)
	if err != nil {
		return nil
	}
	out := make([]DexPoolLink, 0, len(cfg.Dexes))
	for _, d := range cfg.Dexes {
		link := DexPoolLink{
			DexID:   d.ID,
			DexName: d.Name,
			Version: d.Version,
			SwapURL: d.SwapURL,
			InfoURL: d.InfoURL,
		}
		if price != nil && strings.EqualFold(price.DexID, d.DexScreenerID) {
			link.PairAddress = price.PairAddress
			link.LiquidityUSD = price.LiquidityUSD
			link.PriceUSD = price.PriceUSD
			if price.PairAddress != "" {
				link.InfoURL = dexPairURL(d, price.PairAddress)
			}
		}
		out = append(out, link)
	}
	return out
}

func (s *Server) listingProviders(res ListingBridgeResult) []ListingProvider {
	payloadBase := map[string]interface{}{
		"contract_address": res.TokenAddress,
		"chain":            res.ChainName,
		"chain_slug":       res.ChainSlug,
		"name":             res.Name,
		"symbol":           res.Symbol,
		"explorer":         res.ExplorerTokenURL,
		"liquidity_usd":    res.Readiness.LiquidityUSD,
		"holders":          res.Readiness.Holders,
	}

	cgStatus := "manual"
	if res.Readiness.Ready {
		cgStatus = "ready_to_submit"
	}
	cmcStatus := "manual"
	if res.Readiness.Ready && res.Readiness.LiquidityUSD >= 5000 {
		cmcStatus = "ready_to_submit"
	}

	providers := []ListingProvider{
		{
			ID:        "coingecko",
			Name:      "CoinGecko",
			Status:    cgStatus,
			SubmitURL: "https://www.coingecko.com/en/request",
			DocsURL:   "https://support.coingecko.com/hc/en-us/articles/360017972052",
			Payload:   payloadBase,
			Notes:     "Submit after liquidity is live on a major DEX. CoinGecko indexes DexScreener pairs automatically within hours.",
		},
		{
			ID:        "coinmarketcap",
			Name:      "CoinMarketCap",
			Status:    cmcStatus,
			SubmitURL: "https://coinmarketcap.com/request/",
			DocsURL:   "https://support.coinmarketcap.com/hc/en-us/articles/360043018092",
			Payload:   payloadBase,
			Notes:     "CMC requires verified contract, liquidity, and project info. Use CMC API for automated updates after listing.",
		},
		{
			ID:     "explorer",
			Name:   explorerName(res.ChainSlug),
			Status: "info",
			SubmitURL: res.ExplorerTokenURL,
			Payload: map[string]interface{}{
				"token_url": res.ExplorerTokenURL,
				"update":    "Explorer USD price updates from indexed DEX pairs (5–15 min after pool creation).",
			},
			Notes: "Etherscan / BSCScan / Polygonscan pull prices from indexed AMM pools.",
		},
		{
			ID:        "dexscreener",
			Name:      "DexScreener",
			Status:    "auto",
			SubmitURL: fmt.Sprintf("https://dexscreener.com/%s/%s", res.ChainSlug, res.TokenAddress),
			Notes:     "Pools appear automatically once liquidity is added on-chain.",
		},
	}
	return providers
}

func explorerName(chainSlug string) string {
	switch normalizeRegistryChain(chainSlug) {
	case "bsc":
		return "BSCScan"
	case "ethereum":
		return "Etherscan"
	case "polygon":
		return "Polygonscan"
	case "arbitrum":
		return "Arbiscan"
	case "optimism":
		return "Optimistic Etherscan"
	case "avalanche":
		return "Snowtrace"
	case "base":
		return "Basescan"
	case "tron":
		return "Tronscan"
	default:
		return "Block Explorer"
	}
}

func (s *Server) handleDexRegistry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	if s.dexRegistry == nil {
		writeJSON(w, map[string]string{"error": "dex registry not loaded"})
		return
	}
	chain := r.URL.Query().Get("chain")
	dexID := r.URL.Query().Get("dex")
	if chain != "" && dexID != "" {
		cfg, err := s.dexConfigFor(chain, dexID)
		if err != nil {
			writeJSON(w, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, cfg)
		return
	}
	if chain != "" {
		cfg, err := s.dexRegistry.chainConfig(chain)
		if err != nil {
			writeJSON(w, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, map[string]interface{}{
			"chain":  normalizeRegistryChain(chain),
			"name":   cfg.Name,
			"quotes": s.dexRegistry.allQuotes(chain),
			"dexes":  cfg.Dexes,
		})
		return
	}
	writeJSON(w, s.dexRegistry.publicView())
}

func (s *Server) handleListingBridge(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		chain := r.URL.Query().Get("chain")
		token := r.URL.Query().Get("token")
		name := r.URL.Query().Get("name")
		symbol := r.URL.Query().Get("symbol")
		if token == "" {
			http.Error(w, "token required", http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		res, err := s.buildListingBridge(ctx, chain, token, name, symbol)
		if err != nil {
			writeJSON(w, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, res)
	case http.MethodPost:
		if !s.limiter.Allow(clientIP(r)) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		var req struct {
			ChainSlug string `json:"chainSlug"`
			Token     string `json:"tokenAddress"`
			Name      string `json:"name"`
			Symbol    string `json:"symbol"`
			Save      bool   `json:"save"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
		defer cancel()
		res, err := s.buildListingBridge(ctx, req.ChainSlug, req.Token, req.Name, req.Symbol)
		if err != nil {
			writeJSON(w, map[string]string{"error": err.Error()})
			return
		}
		if req.Save {
			_ = s.saveListingBridge(res)
		}
		writeJSON(w, res)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) saveListingBridge(res ListingBridgeResult) error {
	dir := s.cfg.ConfigDir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "listings-bridge-result.json")
	existing := map[string]ListingBridgeResult{}
	if raw, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(raw, &existing)
	}
	key := res.ChainSlug + ":" + strings.ToLower(res.TokenAddress)
	existing[key] = res
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (s *Server) handleFlashListingBridge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	book := s.loadFlashMirrorBook()
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()

	type chainListing struct {
		ChainSlug string              `json:"chainSlug"`
		ChainName string              `json:"chainName"`
		Address   string              `json:"address"`
		Listing   ListingBridgeResult `json:"listing"`
	}
	out := make([]chainListing, 0, len(book.Deployments))
	addr := book.CanonicalAddress
	for _, dep := range book.Deployments {
		if addr == "" {
			addr = dep.ContractAddress
		}
		if addr == "" {
			continue
		}
		slug := dep.ChainID
		li, err := s.buildListingBridge(ctx, slug, addr, book.Name, book.Symbol)
		if err != nil {
			continue
		}
		out = append(out, chainListing{
			ChainSlug: slug,
			ChainName: dep.ChainName,
			Address:   addr,
			Listing:   li,
		})
	}
	writeJSON(w, map[string]interface{}{
		"name":             book.Name,
		"symbol":           book.Symbol,
		"canonicalAddress": book.CanonicalAddress,
		"chains":           out,
		"generatedAt":      time.Now().UTC().Format(time.RFC3339),
	})
}
