package bridge

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

func (s *Server) registerListingRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/bridge/market/token", s.handleTokenMarket)
	mux.HandleFunc("/bridge/listings/bridge", s.handleListingBridge)
	mux.HandleFunc("/bridge/listings/platform", s.handlePlatformListings)
}

func (s *Server) handleTokenMarket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	chain := r.URL.Query().Get("chain")
	addr := r.URL.Query().Get("address")
	if addr == "" {
		addr = r.URL.Query().Get("token")
	}
	market, err := s.b.TokenMarket(chain, addr)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, market)
}

func (s *Server) handleListingBridge(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		chain := r.URL.Query().Get("chain")
		token := r.URL.Query().Get("token")
		if token == "" {
			token = r.URL.Query().Get("address")
		}
		name := r.URL.Query().Get("name")
		symbol := r.URL.Query().Get("symbol")
		if token == "" {
			http.Error(w, "token required", http.StatusBadRequest)
			return
		}
		ctx, cancel := contextWithTimeout(r, 30*time.Second)
		defer cancel()
		res, err := s.b.BuildListingBridge(ctx, chain, token, name, symbol, r.URL.Query().Get("supply"))
		if err != nil {
			writeJSON(w, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, res)
	case http.MethodPost:
		var req struct {
			ChainID  string `json:"chainId"`
			Chain    string `json:"chainSlug"`
			Token    string `json:"tokenAddress"`
			Name     string `json:"name"`
			Symbol   string `json:"symbol"`
			Supply   string `json:"supply"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		chain := req.ChainID
		if chain == "" {
			chain = req.Chain
		}
		ctx, cancel := contextWithTimeout(r, 45*time.Second)
		defer cancel()
		res, err := s.b.BuildListingBridge(ctx, chain, req.Token, req.Name, req.Symbol, req.Supply)
		if err != nil {
			writeJSON(w, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, res)
	default:
		http.Error(w, "GET or POST only", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handlePlatformListings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := contextWithTimeout(r, 60*time.Second)
	defer cancel()
	list, err := s.b.PlatformTokensWithMarket(ctx)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"tokens": list, "count": len(list)})
}

func contextWithTimeout(r *http.Request, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), d)
}
