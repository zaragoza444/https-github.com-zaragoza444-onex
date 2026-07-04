package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type flashMirrorDeployment struct {
	ChainID          string  `json:"chainId"`
	ChainName        string  `json:"chainName"`
	TokenName        string  `json:"tokenName,omitempty"`
	Symbol           string  `json:"symbol"`
	TokenStandard    string  `json:"tokenStandard,omitempty"`
	Decimals         int     `json:"decimals,omitempty"`
	ContractAddress  string  `json:"contractAddress"`
	PredictedAddress string  `json:"predictedAddress,omitempty"`
	TotalSupply      string  `json:"totalSupply,omitempty"`
	TotalSupplyHuman string  `json:"totalSupplyHuman,omitempty"`
	SupplyHuman      string  `json:"supplyHuman"`
	WrapAmountHuman  string  `json:"wrapAmountHuman,omitempty"`
	OwnerBalance     string  `json:"ownerBalance,omitempty"`
	OwnerBalanceHuman string `json:"ownerBalanceHuman,omitempty"`
	OwnerAddress     string  `json:"ownerAddress,omitempty"`
	Transferable     bool    `json:"transferable"`
	PriceUSD         float64 `json:"priceUsd,omitempty"`
	ImpliedPriceUSD  float64 `json:"impliedPriceUsd,omitempty"`
	MarketCapUSD     float64 `json:"marketCapUsd,omitempty"`
	OnChainMarketCapUSD float64 `json:"onChainMarketCapUsd,omitempty"`
	ImpliedMarketCapUSD float64 `json:"impliedMarketCapUsd,omitempty"`
	LiquidityUSD     float64 `json:"liquidityUsd,omitempty"`
	HasLiquidity     bool    `json:"hasLiquidity,omitempty"`
	Holders          int     `json:"holders,omitempty"`
	DexID            string  `json:"dexId,omitempty"`
	PairAddress      string  `json:"pairAddress,omitempty"`
	TxHash           string  `json:"txHash,omitempty"`
	Explorer         string  `json:"explorer"`
	ExplorerTokenURL string  `json:"explorerTokenUrl,omitempty"`
	RPC              string  `json:"rpc,omitempty"`
	Status           string  `json:"status"`
	VerifiedOnChain  bool    `json:"verifiedOnChain"`
	DeployedAt       string  `json:"deployedAt,omitempty"`
}

type flashMirrorBook struct {
	Name               string                  `json:"name"`
	OriginToken        string                  `json:"originToken"`
	Symbol             string                  `json:"symbol"`
	OriginSupplyHuman  string                  `json:"originSupplyHuman,omitempty"`
	WrapAmountPerChain string                  `json:"wrapAmountPerChain,omitempty"`
	Decimals           int                     `json:"decimals,omitempty"`
	CanonicalAddress   string                  `json:"canonicalAddress,omitempty"`
	MirrorMode         string                  `json:"mirrorMode,omitempty"`
	UpdatedAt          string                  `json:"updatedAt"`
	PayloadReloaded    bool                    `json:"payloadReloaded,omitempty"`
	Deployments        []flashMirrorDeployment `json:"deployments"`
}

type chainMeta struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	NetworkID int64  `json:"networkId"`
	Explorer  string `json:"explorer"`
	RPC       string `json:"rpc"`
}

func (s *Server) flashMirrorPaths() (livePath, mirrorPath, configPath string) {
	dir := s.cfg.ConfigDir
	return filepath.Join(dir, "flash-coin-live-addresses.json"),
		filepath.Join(dir, "flash-coin-mirror-result.json"),
		filepath.Join(dir, "flash-coin-mirror.json")
}

func (s *Server) loadChainMeta() map[string]chainMeta {
	raw, err := os.ReadFile(filepath.Join(s.cfg.ConfigDir, "chains.json"))
	if err != nil {
		return nil
	}
	var list []chainMeta
	if json.Unmarshal(raw, &list) != nil {
		return nil
	}
	out := make(map[string]chainMeta, len(list))
	for _, c := range list {
		out[c.ID] = c
	}
	return out
}

func (s *Server) loadFlashMirrorBook() flashMirrorBook {
	livePath, mirrorPath, _ := s.flashMirrorPaths()
	chains := s.loadChainMeta()
	supplyHuman := s.loadWrapSupplyHuman()

	book := flashMirrorBook{Name: "Flash Coin", Symbol: "wFLASH", OriginToken: "FLASH", Deployments: []flashMirrorDeployment{}}
	byChain := map[string]flashMirrorDeployment{}

	if raw, err := os.ReadFile(mirrorPath); err == nil {
		book = mirrorResultToBook(raw, chains, supplyHuman)
		for _, d := range book.Deployments {
			byChain[d.ChainID] = d
		}
	}

	if raw, err := os.ReadFile(livePath); err == nil {
		var live flashMirrorBook
		if json.Unmarshal(raw, &live) == nil {
			if live.Name != "" {
				book.Name = live.Name
			}
			if live.Symbol != "" {
				book.Symbol = live.Symbol
			}
			if live.OriginToken != "" {
				book.OriginToken = live.OriginToken
			}
			if live.UpdatedAt != "" {
				book.UpdatedAt = live.UpdatedAt
			}
			if live.CanonicalAddress != "" {
				book.CanonicalAddress = live.CanonicalAddress
			}
			if live.MirrorMode != "" {
				book.MirrorMode = live.MirrorMode
			}
			for _, d := range live.Deployments {
				if d.ChainID == "" {
					continue
				}
				if meta, ok := chains[d.ChainID]; ok {
					if d.ChainName == "" {
						d.ChainName = meta.Name
					}
					if d.Explorer == "" {
						d.Explorer = meta.Explorer
					}
					if d.RPC == "" {
						d.RPC = meta.RPC
					}
				}
				if d.SupplyHuman == "" {
					d.SupplyHuman = supplyHuman
				}
				if pred, ok := byChain[d.ChainID]; ok && d.PredictedAddress == "" {
					d.PredictedAddress = pred.PredictedAddress
				}
				byChain[d.ChainID] = d
			}
		}
	}

	book.Deployments = make([]flashMirrorDeployment, 0, len(byChain))
	order := s.mirrorChainOrder()
	if len(order) == 0 {
		for _, d := range byChain {
			book.Deployments = append(book.Deployments, d)
		}
	} else {
		for _, id := range order {
			if d, ok := byChain[id]; ok {
				book.Deployments = append(book.Deployments, d)
			}
		}
		for id, d := range byChain {
			found := false
			for _, o := range order {
				if o == id {
					found = true
					break
				}
			}
			if !found {
				book.Deployments = append(book.Deployments, d)
			}
		}
	}
	s.enrichFlashDeployments(&book, chains, supplyHuman)
	cfg := s.loadMirrorConfig()
	book.OriginSupplyHuman = cfg.Supply
	book.Decimals = cfg.Decimals
	book.WrapAmountPerChain = cfg.WrapAmountPerChain
	s.ensureMirrorContractAddresses(&book)
	return book
}

func (s *Server) enrichFlashDeployments(book *flashMirrorBook, chains map[string]chainMeta, supplyHuman string) {
	if book == nil {
		return
	}
	mode := s.loadMirrorMode()
	if mode != "" {
		book.MirrorMode = mode
	}
	var canonical string
	for i := range book.Deployments {
		d := &book.Deployments[i]
		if meta, ok := chains[d.ChainID]; ok {
			d.ChainName = meta.Name
			if d.Explorer == "" {
				d.Explorer = meta.Explorer
			}
			if d.RPC == "" {
				d.RPC = meta.RPC
			}
		} else if d.ChainName == "" {
			d.ChainName = d.ChainID
		}
		if d.SupplyHuman == "" {
			d.SupplyHuman = supplyHuman
		}
		if d.PredictedAddress == "" {
			d.PredictedAddress = d.ContractAddress
		}
		addr := d.ContractAddress
		if addr == "" {
			addr = d.PredictedAddress
		}
		if canonical == "" && addr != "" {
			canonical = addr
		}
	}
	if book.CanonicalAddress == "" {
		book.CanonicalAddress = canonical
	}
}

func (s *Server) loadWrapSupplyHuman() string {
	_, _, cfgPath := s.flashMirrorPaths()
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		return "1000000000"
	}
	var cfg struct {
		WrapAmountPerChain string `json:"wrapAmountPerChain"`
	}
	if json.Unmarshal(raw, &cfg) != nil || cfg.WrapAmountPerChain == "" {
		return "1000000000"
	}
	return cfg.WrapAmountPerChain
}

func (s *Server) loadMirrorMode() string {
	_, _, cfgPath := s.flashMirrorPaths()
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		return ""
	}
	var cfg struct {
		MirrorMode string `json:"mirrorMode"`
	}
	if json.Unmarshal(raw, &cfg) != nil {
		return ""
	}
	return cfg.MirrorMode
}

func (s *Server) mirrorChainOrder() []string {
	_, _, cfgPath := s.flashMirrorPaths()
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil
	}
	var cfg struct {
		MirrorChains []string `json:"mirrorChains"`
	}
	if json.Unmarshal(raw, &cfg) != nil {
		return nil
	}
	return cfg.MirrorChains
}

func mirrorResultToBook(raw []byte, chains map[string]chainMeta, supplyHuman string) flashMirrorBook {
	var result struct {
		Name    string `json:"name"`
		Symbol  string `json:"symbol"`
		Steps   []struct {
			Chain  string `json:"chain"`
			Result struct {
				Wrapped *struct {
					ChainID         string `json:"chainId"`
					Symbol          string `json:"symbol"`
					ContractAddress string `json:"contractAddress"`
					DeployPayload   struct {
						Explorer string `json:"explorer"`
						RPC      string `json:"rpc"`
					} `json:"deployPayload"`
				} `json:"wrapped"`
			} `json:"result"`
		} `json:"steps"`
	}
	book := flashMirrorBook{Name: "Flash Coin", Symbol: "wFLASH", OriginToken: "FLASH", Deployments: []flashMirrorDeployment{}}
	if err := json.Unmarshal(raw, &result); err != nil {
		return book
	}
	if result.Name != "" {
		book.Name = result.Name
	}
	if result.Symbol != "" {
		book.OriginToken = result.Symbol
		book.Symbol = "w" + result.Symbol
	}
	for _, step := range result.Steps {
		if step.Chain == "onex-mainnet-1" || step.Result.Wrapped == nil {
			continue
		}
		w := step.Result.Wrapped
		chainKey := w.ChainID
		if chainKey == "" {
			chainKey = step.Chain
		}
		chainName := step.Chain
		explorer := w.DeployPayload.Explorer
		rpc := w.DeployPayload.RPC
		if meta, ok := chains[chainKey]; ok {
			chainName = meta.Name
			if explorer == "" {
				explorer = meta.Explorer
			}
			if rpc == "" {
				rpc = meta.RPC
			}
		}
		dep := flashMirrorDeployment{
			ChainID:          chainKey,
			ChainName:        chainName,
			Symbol:           w.Symbol,
			ContractAddress:  w.ContractAddress,
			PredictedAddress: w.ContractAddress,
			Explorer:         explorer,
			RPC:              rpc,
			SupplyHuman:      supplyHuman,
			Status:           "predicted",
			VerifiedOnChain:  false,
		}
		book.Deployments = append(book.Deployments, dep)
	}
	return book
}

func (s *Server) verifyFlashMirrorOnChain(ctx context.Context, book *flashMirrorBook) {
	if book == nil || book.CanonicalAddress == "" {
		return
	}
	addr := book.CanonicalAddress
	for i := range book.Deployments {
		d := &book.Deployments[i]
		rpcURL := d.RPC
		if rpcURL == "" {
			continue
		}
		ok, err := s.isContractOn(ctx, rpcURL, addr)
		if err != nil {
			continue
		}
		d.VerifiedOnChain = ok
		if ok {
			d.Status = "live"
			d.ContractAddress = addr
		}
	}
}

func (s *Server) saveFlashMirrorLive(book flashMirrorBook) error {
	livePath, _, _ := s.flashMirrorPaths()
	out := flashMirrorBook{
		Name:               book.Name,
		OriginToken:        book.OriginToken,
		Symbol:             book.Symbol,
		OriginSupplyHuman:  book.OriginSupplyHuman,
		WrapAmountPerChain: book.WrapAmountPerChain,
		Decimals:           book.Decimals,
		CanonicalAddress:   book.CanonicalAddress,
		MirrorMode:         book.MirrorMode,
		UpdatedAt:          book.UpdatedAt,
		PayloadReloaded:    book.PayloadReloaded,
		Deployments:        book.Deployments,
	}
	raw, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(livePath, raw, 0644)
}

func (s *Server) handleFlashMirror(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		book := s.loadFlashMirrorBook()
		chains := s.loadChainMeta()
		reload := r.URL.Query().Get("reload") == "1"
		verify := r.URL.Query().Get("verify") == "1"
		if verify || reload {
			ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
			defer cancel()
			if verify {
				s.verifyFlashMirrorOnChain(ctx, &book)
			}
			if reload {
				// Full contract details: RPC metadata, price, explorer — always on reload.
				s.reloadFlashMirrorPayload(ctx, &book, chains, true)
				book.PayloadReloaded = true
				book.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
				_ = s.saveFlashMirrorLive(book)
			} else {
				book.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			}
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, book)
	case http.MethodPost:
		if !s.limiter.Allow(clientIP(r)) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
		defer cancel()
		book := s.loadFlashMirrorBook()
		chains := s.loadChainMeta()
		s.verifyFlashMirrorOnChain(ctx, &book)
		s.reloadFlashMirrorPayload(ctx, &book, chains, true)
		book.PayloadReloaded = true
		book.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		_ = s.saveFlashMirrorLive(book)
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, book)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
