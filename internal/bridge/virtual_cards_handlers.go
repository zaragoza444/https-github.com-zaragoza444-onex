package bridge

import (
	"encoding/json"
	"net/http"
)

func (s *Server) registerCardRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/bridge/cards/status", s.handleVirtualCardsStatus)
	mux.HandleFunc("/bridge/cards", s.handleVirtualCardsList)
	mux.HandleFunc("/bridge/cards/issue", s.handleVirtualCardIssue)
	mux.HandleFunc("/bridge/cards/authorize", s.handleVirtualCardAuthorize)
	mux.HandleFunc("/bridge/cards/transactions", s.handleVirtualCardTransactions)
}

func (s *Server) handleVirtualCardsStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	_ = s.b.ensureVirtualCards()
	writeJSON(w, s.b.VirtualCardsStatus())
}

func (s *Server) handleVirtualCardsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	if err := s.b.ensureVirtualCards(); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	cards, err := s.b.ListVirtualCards()
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"cards": cards, "count": len(cards), "production": s.b.isProduction()})
}

func (s *Server) handleVirtualCardIssue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if err := s.b.ensureVirtualCards(); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	var req IssueCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	card, err := s.b.IssueVirtualCard(req)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, card)
}

func (s *Server) handleVirtualCardAuthorize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if err := s.b.ensureVirtualCards(); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	var req AuthorizeCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res, err := s.b.AuthorizeVirtualCard(req)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	if !req.Preview {
		_ = s.b.SyncLedgerBook(r.Context(), r.URL.Query().Get("evm"))
	}
	writeJSON(w, res)
}

func (s *Server) handleVirtualCardTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	txs, err := s.b.ListVirtualCardTransactions(50)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"transactions": txs, "count": len(txs)})
}
