package bridge

import (
	"encoding/json"
	"net/http"

	"github.com/onex-blockchain/onex/internal/ai"
)

func (s *Server) registerAIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/bridge/ai/status", s.handleAIStatus)
	mux.HandleFunc("/bridge/ai/chat", s.handleAIChat)
}

func (s *Server) handleAIStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	a := ai.NewAssistant()
	writeJSON(w, a.Status())
}

func (s *Server) handleAIChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req ai.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Context == "" {
		evm := r.URL.Query().Get("evm")
		req.Context = s.buildWalletAIContext(r.Context(), evm)
	}
	out := ai.NewAssistant().Chat(req)
	writeJSON(w, out)
}
