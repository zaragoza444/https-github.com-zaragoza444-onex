package bridge

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/onex-blockchain/onex/internal/ledger"
)

func (s *Server) registerLedgerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/bridge/ledger/status", s.handleLedgerStatus)
	mux.HandleFunc("/bridge/ledger/real", s.handleLedgerReal)
	mux.HandleFunc("/bridge/ledger/read", s.handleLedgerRead)
	mux.HandleFunc("/bridge/ledger/convert", s.handleLedgerConvert)
	mux.HandleFunc("/bridge/ledger/import", s.handleLedgerImport)
	mux.HandleFunc("/bridge/ledger/accounts", s.handleLedgerAccounts)
	mux.HandleFunc("/bridge/ledger/transfer", s.handleLedgerTransfer)
	mux.HandleFunc("/bridge/ledger/transfers", s.handleLedgerTransfers)
	mux.HandleFunc("/bridge/ledger/destinations", s.handleLedgerDestinations)
	// Legacy Shiva paths
	mux.HandleFunc("/bridge/shiva-ledger/status", s.handleLedgerStatus)
	mux.HandleFunc("/bridge/shiva-ledger/real", s.handleLedgerReal)
	mux.HandleFunc("/bridge/shiva-ledger/read", s.handleLedgerRead)
	mux.HandleFunc("/bridge/shiva-ledger/convert", s.handleLedgerConvert)
	mux.HandleFunc("/bridge/shiva-ledger/import", s.handleLedgerImport)
	mux.HandleFunc("/bridge/shiva-ledger/accounts", s.handleLedgerAccounts)
	mux.HandleFunc("/bridge/shiva-ledger/transfer", s.handleLedgerTransfer)
	mux.HandleFunc("/bridge/shiva-ledger/transfers", s.handleLedgerTransfers)
	mux.HandleFunc("/bridge/shiva-ledger/destinations", s.handleLedgerDestinations)
}

func (s *Server) handleLedgerStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, s.b.LedgerStatus())
}

func (s *Server) handleLedgerReal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	evm := r.URL.Query().Get("evm")
	snap, err := s.b.ReadRealLedger(r.Context(), "all", evm, s.b.LoadLatestImport())
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	_ = s.b.SyncLedgerBook(r.Context(), evm)
	writeJSON(w, snap)
}

func (s *Server) handleLedgerRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	source := r.URL.Query().Get("source")
	evm := r.URL.Query().Get("evm")
	snap, err := s.b.ReadRealLedger(r.Context(), source, evm, s.b.LoadLatestImport())
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, snap)
}

func (s *Server) handleLedgerConvert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req ledger.ConvertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res, err := s.b.ConvertLedger(r.Context(), r.URL.Query().Get("evm"), req)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, res)
}

func (s *Server) handleLedgerImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	entries, path, err := s.b.ImportAnyLedger(body)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	evm := r.URL.Query().Get("evm")
	_ = s.b.SyncLedgerBook(r.Context(), evm)
	snap, _ := s.b.ReadRealLedger(r.Context(), "all", evm, body)
	writeJSON(w, map[string]interface{}{
		"status":   "imported",
		"path":     path,
		"entries":  len(entries),
		"parsed":   len(entries),
		"snapshot": snap,
	})
}

func (s *Server) handleLedgerAccounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	evm := r.URL.Query().Get("evm")
	accts, err := s.b.ListLedgerAccounts(r.Context(), evm)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"accounts": accts, "count": len(accts)})
}

func (s *Server) handleLedgerTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req ledger.TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res, err := s.b.TransferLedger(r.Context(), r.URL.Query().Get("evm"), req)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, res)
}

func (s *Server) handleLedgerTransfers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	list, err := s.b.ListLedgerTransfers(r.Context(), r.URL.Query().Get("evm"), 25)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"transfers": list, "count": len(list)})
}

func (s *Server) handleLedgerDestinations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, s.b.ListExternalDestinations())
}
