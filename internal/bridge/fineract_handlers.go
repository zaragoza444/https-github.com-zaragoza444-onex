package bridge

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func (s *Server) registerFineractBankRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/bridge/bank/fineract/status", s.handleFineractStatus)
	mux.HandleFunc("/bridge/bank/fineract/accounts", s.handleFineractAccounts)
	mux.HandleFunc("/bridge/bank/fineract/sync", s.handleFineractSync)
	mux.HandleFunc("/bridge/bank/fineract/deposit", s.handleFineractDeposit)
	mux.HandleFunc("/bridge/bank/fineract/withdraw", s.handleFineractWithdraw)
}

func (s *Server) handleFineractStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, s.b.FineractStatus())
}

func (s *Server) handleFineractAccounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	accts, err := s.b.FineractAccounts()
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"accounts": accts, "count": len(accts)})
}

func (s *Server) handleFineractSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if err := s.b.ensureOnlineBank(); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	synced, err := s.b.SyncFineractAccounts()
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	_ = s.b.SyncHybxMiddleware(r.Context(), r.URL.Query().Get("evm"))
	writeJSON(w, map[string]interface{}{"status": "synced", "accounts": synced, "count": len(synced)})
}

func (s *Server) handleFineractDeposit(w http.ResponseWriter, r *http.Request) {
	s.handleFineractMoney(w, r, true)
}

func (s *Server) handleFineractWithdraw(w http.ResponseWriter, r *http.Request) {
	s.handleFineractMoney(w, r, false)
}

func (s *Server) handleFineractMoney(w http.ResponseWriter, r *http.Request, deposit bool) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if err := s.b.ensureOnlineBank(); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	var req struct {
		AccountID string `json:"accountId"`
		Amount    string `json:"amount"`
		Reference string `json:"reference"`
		Preview   bool   `json:"preview"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	amt, err := strconv.ParseFloat(req.Amount, 64)
	if err != nil || amt <= 0 {
		writeJSON(w, map[string]string{"error": "invalid amount"})
		return
	}
	var res map[string]interface{}
	if deposit {
		res, err = s.b.FineractDeposit(req.AccountID, amt, req.Reference, req.Preview)
	} else {
		res, err = s.b.FineractWithdraw(req.AccountID, amt, req.Reference, req.Preview)
	}
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	if !req.Preview {
		_ = s.b.SyncLedgerBook(r.Context(), r.URL.Query().Get("evm"))
	}
	writeJSON(w, res)
}
