package bridge

import (
	"encoding/json"
	"net/http"

	"github.com/onex-blockchain/onex/internal/ledger"
)

func (s *Server) registerBridge7Routes(mux *http.ServeMux) {
	mux.HandleFunc("/bridge/bridge7/status", s.handleBridge7Status)
	mux.HandleFunc("/bridge/bridge7/ledgers", s.handleBridge7Ledgers)
	mux.HandleFunc("/bridge/bridge7/import", s.handleBridge7Import)
	mux.HandleFunc("/bridge/bridge7/sync", s.handleBridge7Sync)
}

func (s *Server) handleBridge7Status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, s.b.Bridge7Status())
}

func (s *Server) handleBridge7Ledgers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	cfg := ledger.LoadBridge7Config()
	sources := ledger.SummarizeBridge7Files(cfg)
	writeJSON(w, map[string]interface{}{"ledgers": sources, "count": len(sources)})
}

func (s *Server) handleBridge7Import(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Source string          `json:"source"` // local-ledger-2026 | ledger-pro | crypto-ledger | all
		Raw    json.RawMessage `json:"raw"`
		Active bool            `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.Raw) == 0 {
		writeJSON(w, map[string]string{"error": "raw JSON required"})
		return
	}
	evm := r.URL.Query().Get("evm")
	res, err := s.b.ImportActiveLedger(r.Context(), evm, ledger.ImportRequest{
		Raw: req.Raw, Active: true,
	})
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, res)
}

func (s *Server) handleBridge7Sync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	res, err := s.b.SyncBridge7(r.Context(), r.URL.Query().Get("evm"))
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, res)
}
