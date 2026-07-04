package bridge

import (
	"encoding/json"
	"net/http"

	"github.com/onex-blockchain/onex/internal/ledger"
)

func (s *Server) registerHybrixBankRoutes(mux *http.ServeMux) {
	registerHybxRoute := func(path string, h http.HandlerFunc) {
		mux.HandleFunc("/bridge/bank/hybx"+path, h)
		mux.HandleFunc("/bridge/bank/hybrix"+path, h) // legacy alias
	}
	registerHybxRoute("/status", s.handleHybrixStatus)
	registerHybxRoute("/assets", s.handleHybrixAssets)
	registerHybxRoute("/mirrors", s.handleHybrixMirrors)
	registerHybxRoute("/sync", s.handleHybrixSync)
	registerHybxRoute("/convert", s.handleHybrixConvert)
	registerHybxRoute("/cards", s.handleHybxCardsList)
	registerHybxRoute("/cards/issue", s.handleHybxCardsIssue)
	registerHybxRoute("/middleware/status", s.handleHybxMiddlewareStatus)
	registerHybxRoute("/exchange/routes", s.handleHybxExchangeRoutes)
	registerHybxRoute("/exchange/quote", s.handleHybxExchangeQuote)
	registerHybxRoute("/exchange", s.handleHybxExchange)
	registerHybxRoute("/settle", s.handleHybxSettle)
	registerHybxRoute("/federation", s.handleHybxFederation)
}

func (s *Server) handleHybrixStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, s.b.HybrixStatus())
}

func (s *Server) handleHybrixAssets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	assets, err := s.b.HybrixAssets()
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"assets": assets, "count": len(assets)})
}

func (s *Server) handleHybrixMirrors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	mirrors, err := s.b.HybrixMirrors()
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"mirrors": mirrors, "count": len(mirrors)})
}

func (s *Server) handleHybrixSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if err := s.b.ensureOnlineBank(); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	mirrors, err := s.b.SyncHybrixMirrors()
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	_ = s.b.ensureVirtualCards()
	_ = s.b.SyncHybxMiddleware(r.Context(), r.URL.Query().Get("evm"))
	writeJSON(w, map[string]interface{}{"status": "synced", "mirrors": mirrors, "count": len(mirrors)})
}

func (s *Server) handleHybrixConvert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if err := s.b.ensureOnlineBank(); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	var req struct {
		Direction  string `json:"direction"`
		NSBAccount string `json:"nsbAccount"`
		Amount     string `json:"amount"`
		Preview    bool   `json:"preview"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res, err := s.b.HybrixConvert(req.Direction, req.NSBAccount, req.Amount, req.Preview)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	if !req.Preview {
		_ = s.b.SyncLedgerBook(r.Context(), r.URL.Query().Get("evm"))
	}
	writeJSON(w, res)
}

func (s *Server) handleHybxCardsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	if err := s.b.ensureVirtualCards(); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	cards, err := s.b.ListVirtualCardsByIssuer("hybx")
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"issuer": "hybx", "cards": cards, "count": len(cards), "production": s.b.isProduction()})
}

func (s *Server) handleHybxCardsIssue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	cards, err := s.b.IssueHybxVirtualCards()
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	_ = s.b.SyncLedgerBook(r.Context(), r.URL.Query().Get("evm"))
	writeJSON(w, map[string]interface{}{"status": "issued", "cards": cards, "count": len(cards)})
}

func (s *Server) handleHybxMiddlewareStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, s.b.HybxMiddlewareStatus())
}

func (s *Server) handleHybxExchangeRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	routes := s.b.HybxExchangeRoutes()
	writeJSON(w, map[string]interface{}{"routes": routes, "count": len(routes)})
}

func (s *Server) handleHybxExchangeQuote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req ledger.HybxExchangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res, err := s.b.QuoteHybxExchange(req)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, res)
}

func (s *Server) handleHybxExchange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req ledger.HybxExchangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res, err := s.b.HybxExchange(req)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	if !req.Preview {
		_ = s.b.SyncHybxMiddleware(r.Context(), r.URL.Query().Get("evm"))
	}
	writeJSON(w, res)
}

func (s *Server) handleHybxSettle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req ledger.HybxExchangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res, err := s.b.HybxSettle(req)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	if !req.Preview {
		_ = s.b.SyncHybxMiddleware(r.Context(), r.URL.Query().Get("evm"))
	}
	writeJSON(w, res)
}

func (s *Server) handleHybxFederation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	recs, err := ledger.ListHybxFederationRecords(50)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"records": recs, "count": len(recs)})
}
