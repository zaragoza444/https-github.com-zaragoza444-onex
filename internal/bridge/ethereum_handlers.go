package bridge

import (
	"encoding/json"
	"net/http"
)

func (s *Server) registerEthereumRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/bridge/ethereum/status", s.handleEthereumStatus)
	mux.HandleFunc("/bridge/ethereum/transfer", s.handleEthereumTransfer)
	// Legacy alias
	mux.HandleFunc("/bridge/quiknode/status", s.handleEthereumStatus)
	mux.HandleFunc("/bridge/quiknode/transfer", s.handleEthereumTransfer)
}

func (s *Server) handleEthereumStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, s.b.EthereumStatus(r.Context()))
}

func (s *Server) handleEthereumTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req EthereumTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res, err := s.b.EthereumTransfer(r.Context(), req)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, res)
}
