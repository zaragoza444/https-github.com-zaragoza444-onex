package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type Server struct {
	cfg         Config
	store       *TokenStore
	liquidity   *LiquidityStore
	bscscan     *bscScanClient
	price       *priceClient
	market      *marketClient
	limiter     *rateLimiter
	dexRegistry *DexRegistry
}

func NewServer(cfg Config) *Server {
	reg, _ := loadDexRegistry(cfg.ConfigDir)
	return &Server{
		cfg:         cfg,
		store:       NewTokenStore(cfg.DataDir),
		liquidity:   NewLiquidityStore(cfg.DataDir),
		bscscan:     newBSCScanClient(cfg.BSCScanAPIKey),
		price:       newPriceClient(),
		market:      newMarketClient(),
		limiter:     newRateLimiter(cfg.RateLimitPerMin),
		dexRegistry: reg,
	}
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	abiArr, err := contractABIArray()
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	dex, err := s.dexConfigFor("bsc", "pancake-v2")
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	def := defaultChain(s.cfg)
	writeJSON(w, map[string]interface{}{
		"chainId":              def.ChainID,
		"chainSlug":            def.Slug,
		"chainName":            def.Name,
		"rpcUrl":               def.RPCURL,
		"explorer":             def.Explorer,
		"nativeSymbol":         def.NativeSymbol,
		"chains":               supportedChains(),
		"contractAbi":          abiArr,
		"contractBytecode":     contractBytecodeHex(),
		"backendDeployEnabled": s.cfg.DeployerKey != "",
		"deployerAddress":      s.cfg.DeployerAddress,
		"poolLiveMode":         poolLiveMode(s.cfg.DeployerKey),
		"apiKeyRequired":       s.cfg.APIKey != "",
		"env":                  s.cfg.Env,
		"build":                s.cfg.BuildVersion,
		"configDir":            s.cfg.ConfigDir,
		"dex":                  dex,
		"dexRegistry":          s.dexRegistry != nil,
	})
}

func (s *Server) handleDeploy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if s.cfg.DeployerKey == "" {
		writeJSON(w, map[string]string{"error": "backend deploy disabled: set BSC_DEPLOYER_PRIVATE_KEY"})
		return
	}
	if !s.limiter.Allow(clientIP(r)) {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	var req DeployRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validateDeployRequest(req); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}

	supplyRaw, err := parseSupply(req.Supply, req.Decimals)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}

	chainSlug := resolveChainSlug(req.Chain, req.Features.Chain)
	chain, err := deployChain(chainSlug)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Minute)
	defer cancel()

	init, err := buildInitParams(req.Name, strings.ToUpper(req.Symbol), req.Decimals, supplyRaw, req.Features, "")
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	if init.Owner == (common.Address{}) {
		keyHex := strings.TrimPrefix(strings.TrimSpace(s.cfg.DeployerKey), "0x")
		if key, kerr := crypto.HexToECDSA(keyHex); kerr == nil {
			pub := key.Public().(*ecdsa.PublicKey)
			init.Owner = crypto.PubkeyToAddress(*pub)
			if init.Recipient == (common.Address{}) {
				init.Recipient = init.Owner
			}
		}
	}

	result, err := s.deployContract(ctx, init, s.cfg.DeployerKey, chain)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}

	rec := TokenRecord{
		ContractAddress: result.ContractAddress,
		Name:            req.Name,
		Symbol:          strings.ToUpper(req.Symbol),
		Decimals:        req.Decimals,
		Supply:          req.Supply,
		TxHash:          result.TxHash,
		Creator:         result.Creator,
		DeployMethod:    "backend",
		ChainID:         chain.ChainID,
		ChainSlug:       chain.Slug,
		ChainName:       chain.Name,
		Explorer:        chain.Explorer,
	}
	if err := s.store.Add(rec); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, map[string]interface{}{
		"token":            rec,
		"txHash":           result.TxHash,
		"chain":            chain,
		"explorerTokenUrl": explorerTokenURL(chain.Explorer, result.ContractAddress),
		"explorerTxUrl":    explorerTxURL(chain.Explorer, result.TxHash),
	})
}

type RegisterRequest struct {
	ContractAddress string        `json:"contractAddress"`
	Name            string        `json:"name"`
	Symbol          string        `json:"symbol"`
	Decimals        int           `json:"decimals"`
	Supply          string        `json:"supply"`
	TxHash          string        `json:"txHash"`
	Creator         string        `json:"creator"`
	Chain           string        `json:"chain"`
	Features        TokenFeatures `json:"features"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.ContractAddress == "" || req.TxHash == "" {
		writeJSON(w, map[string]string{"error": "contractAddress and txHash required"})
		return
	}
	if err := validateDeployRequest(DeployRequest{
		Name: req.Name, Symbol: req.Symbol, Decimals: req.Decimals, Supply: req.Supply,
	}); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}

	chainSlug := resolveChainSlug(req.Chain, req.Features.Chain)
	chain, err := deployChain(chainSlug)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	verified, err := s.verifyDeployTx(ctx, req.TxHash, req.ContractAddress, chain.RPCURL, chain.ChainID)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	if req.Creator == "" {
		req.Creator = verified.Creator
	}

	rec := TokenRecord{
		ContractAddress: verified.ContractAddress,
		Name:            req.Name,
		Symbol:          strings.ToUpper(req.Symbol),
		Decimals:        req.Decimals,
		Supply:          req.Supply,
		TxHash:          req.TxHash,
		Creator:         req.Creator,
		DeployMethod:    "metamask",
		ChainID:         chain.ChainID,
		ChainSlug:       chain.Slug,
		ChainName:       chain.Name,
		Explorer:        chain.Explorer,
	}
	if err := s.store.Add(rec); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, map[string]interface{}{
		"token":            rec,
		"chain":            chain,
		"explorerTokenUrl": explorerTokenURL(chain.Explorer, rec.ContractAddress),
		"explorerTxUrl":    explorerTxURL(chain.Explorer, rec.TxHash),
	})
}

func (s *Server) handleTokens(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	list, err := s.store.Load()
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	list = s.mergeFlashCoinToken(list)
	writeJSON(w, list)
}

func (s *Server) handleTokenDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	addr := strings.TrimPrefix(r.URL.Path, "/api/tokens/")
	if addr == "" || strings.Contains(addr, "/") {
		http.Error(w, "address required", http.StatusBadRequest)
		return
	}

	rec, regErr := s.store.Find(addr)
	lookupAddr := addr
	chain := defaultChain(s.cfg)
	if regErr == nil {
		lookupAddr = rec.ContractAddress
		if rec.ChainSlug != "" {
			if c, err := chainBySlug(rec.ChainSlug); err == nil {
				chain = c
			}
		} else if rec.ChainID != 0 {
			if c, err := chainByID(rec.ChainID); err == nil {
				chain = c
			}
		}
		if rec.Explorer != "" {
			chain.Explorer = rec.Explorer
		}
	} else if q := r.URL.Query().Get("chain"); q != "" {
		if c, err := chainBySlug(q); err == nil {
			chain = c
		}
	}

	info, err := s.tokenInfoForChain(r.Context(), chain, lookupAddr)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	quote, _ := s.price.Quote(chain.DexChainID, lookupAddr)

	resp := map[string]interface{}{
		"bscscan":          info,
		"price":            quote,
		"chain":            chain,
		"explorerTokenUrl": explorerTokenURL(chain.Explorer, lookupAddr),
	}
	if regErr == nil {
		resp["token"] = rec
		resp["explorerTxUrl"] = explorerTxURL(chain.Explorer, rec.TxHash)
	}
	writeJSON(w, resp)
}

func (s *Server) handleBSCScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	addr := strings.TrimPrefix(r.URL.Path, "/api/bscscan/")
	if addr == "" {
		http.Error(w, "address required", http.StatusBadRequest)
		return
	}
	chain := defaultChain(s.cfg)
	if q := r.URL.Query().Get("chain"); q != "" {
		if c, err := chainBySlug(q); err == nil {
			chain = c
		}
	}
	info, err := s.tokenInfoForChain(r.Context(), chain, addr)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, info)
}

func (s *Server) handlePrice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	addr := strings.TrimPrefix(r.URL.Path, "/api/price/")
	if addr == "" {
		http.Error(w, "address required", http.StatusBadRequest)
		return
	}
	chain := defaultChain(s.cfg)
	if q := r.URL.Query().Get("chain"); q != "" {
		if c, err := chainBySlug(q); err == nil {
			chain = c
		}
	}
	quote, err := s.price.Quote(chain.DexChainID, addr)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, quote)
}

func (s *Server) handleMarketBNB(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	usd, err := s.market.BNBUSD()
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"symbol": "BNB", "priceUsd": usd})
}

func (s *Server) handleLiquidityQuote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	tokenAmount, err := parsePositiveFloat(r.URL.Query().Get("tokenAmount"))
	if err != nil {
		writeJSON(w, map[string]string{"error": "tokenAmount required"})
		return
	}
	targetUSD, err := parsePositiveFloat(r.URL.Query().Get("targetUsd"))
	if err != nil || targetUSD == 0 {
		targetUSD = 1
	}
	chainSlug := r.URL.Query().Get("chain")
	if chainSlug == "" {
		chainSlug = "bsc"
	}
	quote := strings.ToLower(r.URL.Query().Get("quote"))
	if quote == "" {
		quote = "usdt"
	}

	var quoteAmount float64
	var quoteSymbol string
	switch quote {
	case "usdt", "usdc":
		quoteAmount = tokenAmount * targetUSD
		quoteSymbol = strings.ToUpper(quote)
	default:
		nativeUSD, err := s.market.NativeUSD(chainSlug)
		if err != nil {
			writeJSON(w, map[string]string{"error": err.Error()})
			return
		}
		quoteAmount = (tokenAmount * targetUSD) / nativeUSD
		if s.dexRegistry != nil {
			if q, e := s.dexRegistry.quoteToken(chainSlug, quote); e == nil {
				quoteSymbol = q.Symbol
			}
		}
		if quoteSymbol == "" {
			quoteSymbol = strings.ToUpper(quote)
		}
	}

	explorerNote := explorerName(chainSlug) + " shows USD price from indexed DEX liquidity. USDT/USDC pair is best for a $1 listing."
	marketCap := tokenAmount * targetUSD
	writeJSON(w, map[string]interface{}{
		"tokenAmount":   tokenAmount,
		"targetUsd":     targetUSD,
		"chain":         normalizeRegistryChain(chainSlug),
		"quoteId":       quote,
		"quoteSymbol":   quoteSymbol,
		"quoteAmount":   formatAmount(quoteAmount),
		"impliedPrice":  targetUSD,
		"marketCapUsd":  marketCap,
		"explorerNote":  explorerNote,
		"bscscanNote":   fmt.Sprintf("BSCScan market cap ≈ $%.0f (%.0f tokens × $%.2f)", marketCap, tokenAmount, targetUSD),
		"recommendUsdt": quote != "usdt" && quote != "usdc",
	})
}

func parsePositiveFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	var v float64
	if _, err := fmt.Sscanf(s, "%f", &v); err != nil {
		return 0, err
	}
	if v <= 0 {
		return 0, fmt.Errorf("must be > 0")
	}
	return v, nil
}

func formatAmount(v float64) string {
	if v >= 1 {
		return fmt.Sprintf("%.6f", v)
	}
	return fmt.Sprintf("%.10f", v)
}

func (s *Server) handleLiquidityPair(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	token := r.URL.Query().Get("token")
	quote := r.URL.Query().Get("quote")
	chainSlug := r.URL.Query().Get("chain")
	dexID := r.URL.Query().Get("dex")
	if token == "" {
		http.Error(w, "token required", http.StatusBadRequest)
		return
	}
	if chainSlug == "" {
		chainSlug = "bsc"
	}
	if quote == "" {
		quote = "usdt"
	}
	if dexID == "" && s.dexRegistry != nil {
		if _, d, err := s.dexRegistry.dex(chainSlug, ""); err == nil {
			dexID = d.ID
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	dexCfg, err := s.dexConfigFor(chainSlug, dexID)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	pair, err := s.getPairAddress(ctx, chainSlug, dexID, token, quote)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	_, regDex, _ := s.dexRegistry.dex(chainSlug, dexID)
	swapURL := dexPairURL(regDex, pair)
	writeJSON(w, map[string]interface{}{
		"tokenAddress": token,
		"chain":        normalizeRegistryChain(chainSlug),
		"dexId":        dexID,
		"dexName":      dexCfg.Name,
		"quoteId":      quote,
		"pairAddress":  pair,
		"exists":       pair != "",
		"liquidityMode": dexCfg.LiquidityMode,
		"liquidityUrl": dexCfg.LiquidityURL,
		"swapUrl":      dexCfg.SwapURL,
		"dexUrl":       swapURL,
	})
}

func (s *Server) handleLiquidityList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	list, err := s.liquidity.Load()
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, list)
}

type LiquidityRegisterRequest struct {
	ChainSlug    string `json:"chainSlug"`
	DexID        string `json:"dexId"`
	TokenAddress string `json:"tokenAddress"`
	QuoteID      string `json:"quoteId"`
	PairAddress  string `json:"pairAddress"`
	TokenAmount  string `json:"tokenAmount"`
	QuoteAmount  string `json:"quoteAmount"`
	TxHash       string `json:"txHash"`
	Creator      string `json:"creator"`
}

func (s *Server) handleLiquidityRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req LiquidityRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.TokenAddress == "" || req.TxHash == "" {
		writeJSON(w, map[string]string{"error": "tokenAddress and txHash required"})
		return
	}
	if req.ChainSlug == "" {
		req.ChainSlug = "bsc"
	}
	if req.QuoteID == "" {
		req.QuoteID = "usdt"
	}
	if req.DexID == "" {
		req.DexID = "pancake-v2"
	}

	chain, _ := chainBySlug(req.ChainSlug)
	explorer := chain.Explorer
	if explorer == "" && s.dexRegistry != nil {
		if cfg, e := s.dexRegistry.chainConfig(req.ChainSlug); e == nil {
			explorer = cfg.Explorer
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	if req.PairAddress == "" {
		pair, err := s.getPairAddress(ctx, req.ChainSlug, req.DexID, req.TokenAddress, req.QuoteID)
		if err == nil && pair != "" {
			req.PairAddress = pair
		}
	}

	rec := LiquidityRecord{
		ChainSlug:    normalizeRegistryChain(req.ChainSlug),
		DexID:        req.DexID,
		TokenAddress: req.TokenAddress,
		QuoteID:      req.QuoteID,
		PairAddress:  req.PairAddress,
		TokenAmount:  req.TokenAmount,
		QuoteAmount:  req.QuoteAmount,
		TxHash:       req.TxHash,
		Creator:      req.Creator,
	}
	if err := s.liquidity.Add(rec); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	_, regDex, _ := s.dexRegistry.dex(req.ChainSlug, req.DexID)
	dexURL := dexPairURL(regDex, rec.PairAddress)
	dexChain := chain.DexChainID
	if dexChain == "" {
		dexChain = normalizeRegistryChain(req.ChainSlug)
	}
	writeJSON(w, map[string]interface{}{
		"liquidity":        rec,
		"explorerTxUrl":    explorerTxURL(explorer, rec.TxHash),
		"explorerTokenUrl": explorerTokenURL(explorer, rec.TokenAddress),
		"dexUrl":           dexURL,
		"dexscreenerUrl":   fmt.Sprintf("https://dexscreener.com/%s/%s", dexChain, rec.PairAddress),
		"explorerNote":     explorerName(req.ChainSlug) + " price updates within ~5–15 minutes after the pool is indexed.",
	})
}

func poolLiveMode(deployerKey string) string {
	if deployerKey != "" {
		return "cli"
	}
	return "metamask"
}

func validateDeployRequest(req DeployRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("name required")
	}
	if strings.TrimSpace(req.Symbol) == "" {
		return fmt.Errorf("symbol required")
	}
	if req.Decimals < 0 || req.Decimals > 18 {
		return fmt.Errorf("decimals must be 0-18")
	}
	if strings.TrimSpace(req.Supply) == "" {
		return fmt.Errorf("supply required")
	}
	return nil
}

type rateLimiter struct {
	perMin int
	mu     sync.Mutex
	hits   map[string][]time.Time
}

func newRateLimiter(perMin int) *rateLimiter {
	return &rateLimiter{perMin: perMin, hits: make(map[string][]time.Time)}
}

func (rl *rateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-time.Minute)
	var kept []time.Time
	for _, t := range rl.hits[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= rl.perMin {
		rl.hits[key] = kept
		return false
	}
	rl.hits[key] = append(kept, now)
	return true
}
