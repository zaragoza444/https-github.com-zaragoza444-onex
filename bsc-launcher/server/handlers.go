package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Server struct {
	cfg      Config
	store    *TokenStore
	bscscan  *bscScanClient
	price    *priceClient
	limiter  *rateLimiter
}

func NewServer(cfg Config) *Server {
	return &Server{
		cfg:     cfg,
		store:   NewTokenStore(cfg.DataDir),
		bscscan: newBSCScanClient(cfg.BSCScanAPIKey),
		price:   newPriceClient(),
		limiter: newRateLimiter(cfg.RateLimitPerMin),
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
	writeJSON(w, map[string]interface{}{
		"chainId":              s.cfg.ChainID,
		"chainName":            "BNB Smart Chain",
		"rpcUrl":               s.cfg.RPCURL,
		"explorer":             s.cfg.Explorer,
		"nativeSymbol":         "BNB",
		"contractAbi":          abiArr,
		"contractBytecode":     contractBytecodeHex(),
		"backendDeployEnabled": s.cfg.DeployerKey != "",
		"apiKeyRequired":       s.cfg.APIKey != "",
		"env":                  s.cfg.Env,
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
	if !s.limiter.Allow(r.RemoteAddr) {
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

	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Minute)
	defer cancel()

	result, err := s.deployContract(ctx, req.Name, strings.ToUpper(req.Symbol), req.Decimals, supplyRaw, s.cfg.DeployerKey)
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
	}
	if err := s.store.Add(rec); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, map[string]interface{}{
		"token":   rec,
		"txHash":  result.TxHash,
		"explorerTokenUrl": s.cfg.Explorer + "/token/" + result.ContractAddress,
		"explorerTxUrl":    s.cfg.Explorer + "/tx/" + result.TxHash,
	})
}

type RegisterRequest struct {
	ContractAddress string `json:"contractAddress"`
	Name            string `json:"name"`
	Symbol          string `json:"symbol"`
	Decimals        int    `json:"decimals"`
	Supply          string `json:"supply"`
	TxHash          string `json:"txHash"`
	Creator         string `json:"creator"`
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

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	verified, err := s.verifyDeployTx(ctx, req.TxHash, req.ContractAddress)
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
	}
	if err := s.store.Add(rec); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, map[string]interface{}{
		"token": rec,
		"explorerTokenUrl": s.cfg.Explorer + "/token/" + rec.ContractAddress,
		"explorerTxUrl":    s.cfg.Explorer + "/tx/" + rec.TxHash,
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
	if regErr == nil {
		lookupAddr = rec.ContractAddress
	}

	info, err := s.tokenInfo(r.Context(), lookupAddr)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	quote, _ := s.price.Quote(lookupAddr)

	resp := map[string]interface{}{
		"bscscan": info,
		"price":   quote,
		"explorerTokenUrl": s.cfg.Explorer + "/token/" + lookupAddr,
	}
	if regErr == nil {
		resp["token"] = rec
		resp["explorerTxUrl"] = s.cfg.Explorer + "/tx/" + rec.TxHash
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
	info, err := s.tokenInfo(r.Context(), addr)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, info)
}

func (s *Server) tokenInfo(ctx context.Context, address string) (*BSCScanTokenInfo, error) {
	chain, chainErr := s.readOnChainToken(ctx, address)
	if chainErr != nil {
		if chain != nil && !chain.IsContract {
			return &BSCScanTokenInfo{
				ContractAddress: address,
				IsWallet:        true,
				Error:           chainErr.Error(),
			}, chainErr
		}
		return nil, chainErr
	}

	scan, _ := s.bscscan.TokenInfo(address)
	return mergeTokenInfo(scan, chain), nil
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
	quote, err := s.price.Quote(addr)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, quote)
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
