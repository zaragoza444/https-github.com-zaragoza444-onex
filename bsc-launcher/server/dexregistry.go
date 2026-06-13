package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type RegistryQuote struct {
	ID       string `json:"id"`
	Symbol   string `json:"symbol"`
	Address  string `json:"address"`
	Decimals int    `json:"decimals"`
}

type RegistryDEX struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Version           int    `json:"version"`
	Type              string `json:"type"`
	LiquidityMode     string `json:"liquidityMode"`
	Router            string `json:"router,omitempty"`
	Factory           string `json:"factory,omitempty"`
	PositionManager   string `json:"positionManager,omitempty"`
	RouterABI         string `json:"routerAbi,omitempty"`
	FactoryABI        string `json:"factoryAbi,omitempty"`
	LiquidityURL      string `json:"liquidityUrl"`
	InfoURL           string `json:"infoUrl"`
	SwapURL           string `json:"swapUrl"`
	DexScreenerID     string `json:"dexscreenerId"`
}

type ChainDEXConfig struct {
	Name          string          `json:"name"`
	NetworkID     int64           `json:"networkId"`
	Explorer      string          `json:"explorer"`
	DexChainID    string          `json:"dexChainId"`
	WrappedNative RegistryQuote   `json:"wrappedNative"`
	Stablecoins   []RegistryQuote `json:"stablecoins"`
	Dexes         []RegistryDEX   `json:"dexes"`
}

type DexRegistry struct {
	Version int                       `json:"version"`
	Chains  map[string]ChainDEXConfig `json:"chains"`
}

func loadDexRegistry(configDir string) (*DexRegistry, error) {
	path := filepath.Join(configDir, "dex-registry.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var reg DexRegistry
	if err := json.Unmarshal(raw, &reg); err != nil {
		return nil, err
	}
	return &reg, nil
}

func normalizeRegistryChain(slug string) string {
	slug = strings.ToLower(strings.TrimSpace(slug))
	switch slug {
	case "", "bnb", "bsc":
		return "bsc"
	case "eth", "mainnet":
		return "ethereum"
	case "matic":
		return "polygon"
	case "arb":
		return "arbitrum"
	case "op":
		return "optimism"
	case "avax":
		return "avalanche"
	default:
		return slug
	}
}

func (r *DexRegistry) chainConfig(chainSlug string) (ChainDEXConfig, error) {
	if r == nil {
		return ChainDEXConfig{}, fmt.Errorf("dex registry not loaded")
	}
	key := normalizeRegistryChain(chainSlug)
	cfg, ok := r.Chains[key]
	if !ok {
		return ChainDEXConfig{}, fmt.Errorf("no dex config for chain %s", chainSlug)
	}
	return cfg, nil
}

func (r *DexRegistry) dex(chainSlug, dexID string) (ChainDEXConfig, RegistryDEX, error) {
	cfg, err := r.chainConfig(chainSlug)
	if err != nil {
		return cfg, RegistryDEX{}, err
	}
	if dexID == "" {
		if len(cfg.Dexes) == 0 {
			return cfg, RegistryDEX{}, fmt.Errorf("no dexes on %s", chainSlug)
		}
		return cfg, cfg.Dexes[0], nil
	}
	for _, d := range cfg.Dexes {
		if d.ID == dexID {
			return cfg, d, nil
		}
	}
	return cfg, RegistryDEX{}, fmt.Errorf("dex %s not found on %s", dexID, chainSlug)
}

func (r *DexRegistry) quoteToken(chainSlug, quoteID string) (RegistryQuote, error) {
	cfg, err := r.chainConfig(chainSlug)
	if err != nil {
		return RegistryQuote{}, err
	}
	quoteID = strings.ToLower(quoteID)
	if quoteID == "bnb" || quoteID == "eth" || quoteID == "matic" || quoteID == "avax" {
		return cfg.WrappedNative, nil
	}
	for _, q := range cfg.Stablecoins {
		if q.ID == quoteID {
			return q, nil
		}
	}
	if cfg.WrappedNative.ID == quoteID {
		return cfg.WrappedNative, nil
	}
	return RegistryQuote{}, fmt.Errorf("unsupported quote %s on %s", quoteID, chainSlug)
}

func (r *DexRegistry) allQuotes(chainSlug string) []QuoteToken {
	cfg, err := r.chainConfig(chainSlug)
	if err != nil {
		return nil
	}
	out := make([]QuoteToken, 0, 1+len(cfg.Stablecoins))
	wn := cfg.WrappedNative
	out = append(out, QuoteToken{ID: wn.ID, Symbol: wn.Symbol, Address: wn.Address, Decimals: wn.Decimals})
	for _, s := range cfg.Stablecoins {
		out = append(out, QuoteToken{ID: s.ID, Symbol: s.Symbol, Address: s.Address, Decimals: s.Decimals})
	}
	return out
}

func (s *Server) dexConfigFor(chainSlug, dexID string) (DexConfig, error) {
	if s.dexRegistry == nil {
		return DexConfig{}, fmt.Errorf("dex registry unavailable")
	}
	chainCfg, dex, err := s.dexRegistry.dex(chainSlug, dexID)
	if err != nil {
		return DexConfig{}, err
	}
	quotes := s.dexRegistry.allQuotes(chainSlug)

	cfg := DexConfig{
		ID:            dex.ID,
		ChainSlug:     normalizeRegistryChain(chainSlug),
		ChainName:     chainCfg.Name,
		NetworkID:     chainCfg.NetworkID,
		Explorer:      chainCfg.Explorer,
		DexChainID:    chainCfg.DexChainID,
		Router:        dex.Router,
		Factory:       dex.Factory,
		Name:          dex.Name,
		Version:       dex.Version,
		Type:          dex.Type,
		LiquidityMode: dex.LiquidityMode,
		LiquidityURL:  dex.LiquidityURL,
		InfoURL:       dex.InfoURL,
		SwapURL:       dex.SwapURL,
		DexScreenerID: dex.DexScreenerID,
		Quotes:        quotes,
	}
	if dex.LiquidityMode == "router-v2" {
		if dex.RouterABI == "" {
			dex.RouterABI = "PancakeRouter.json"
		}
		if dex.FactoryABI == "" {
			dex.FactoryABI = "PancakeFactory.json"
		}
		routerABI, err := loadABIFile(dex.RouterABI)
		if err != nil {
			return DexConfig{}, err
		}
		factoryABI, err := loadABIFile(dex.FactoryABI)
		if err != nil {
			return DexConfig{}, err
		}
		cfg.RouterABI = routerABI
		cfg.FactoryABI = factoryABI
	}
	if len(quotes) > 0 {
		cfg.WrappedNative = quotes[0].Address
	}
	for _, q := range quotes {
		if q.ID == "usdt" {
			cfg.USDT = q.Address
			break
		}
	}
	return cfg, nil
}

func dexPairURL(dex RegistryDEX, pair string) string {
	if pair == "" {
		return dex.LiquidityURL
	}
	return strings.TrimRight(dex.InfoURL, "/") + "/" + pair
}

func (r *DexRegistry) publicView() map[string]interface{} {
	if r == nil {
		return map[string]interface{}{"chains": map[string]interface{}{}}
	}
	chains := make(map[string]interface{}, len(r.Chains))
	for id, cfg := range r.Chains {
		dexes := make([]map[string]interface{}, 0, len(cfg.Dexes))
		for _, d := range cfg.Dexes {
			dexes = append(dexes, map[string]interface{}{
				"id":            d.ID,
				"name":          d.Name,
				"version":       d.Version,
				"type":          d.Type,
				"liquidityMode": d.LiquidityMode,
				"liquidityUrl":  d.LiquidityURL,
				"infoUrl":       d.InfoURL,
				"swapUrl":       d.SwapURL,
				"dexscreenerId": d.DexScreenerID,
			})
		}
		quotes := r.allQuotes(id)
		chains[id] = map[string]interface{}{
			"name":       cfg.Name,
			"networkId":  cfg.NetworkID,
			"explorer":   cfg.Explorer,
			"dexChainId": cfg.DexChainID,
			"quotes":     quotes,
			"dexes":      dexes,
		}
	}
	return map[string]interface{}{
		"version": r.Version,
		"chains":  chains,
	}
}
