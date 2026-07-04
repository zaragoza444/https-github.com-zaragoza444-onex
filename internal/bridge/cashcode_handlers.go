package bridge

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/onex-blockchain/onex/internal/ledger"
)

func (s *Server) registerCashCodeRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/bridge/cashcode/status", s.handleCashCodeStatus)
	mux.HandleFunc("/bridge/cashcode/list", s.handleCashCodeList)
	mux.HandleFunc("/bridge/cashcode/issue", s.handleCashCodeIssue)
	mux.HandleFunc("/bridge/cashcode/redeem", s.handleCashCodeRedeem)
	mux.HandleFunc("/bridge/cashcode/verify", s.handleCashCodeVerify)
	mux.HandleFunc("/bridge/cashcode/cancel", s.handleCashCodeCancel)
}

func (s *Server) handleCashCodeStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, s.b.CashCodeStatus())
}

func (s *Server) handleCashCodeList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	accountID := strings.TrimSpace(r.URL.Query().Get("accountId"))
	codes, err := s.b.ListCashCodes(accountID)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"codes": codes, "count": len(codes)})
}

func (s *Server) handleCashCodeIssue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if err := s.b.ensureOnlineBank(); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	var req ledger.CashCodeIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]string{"error": "invalid json"})
		return
	}
	res, err := s.b.IssueCashCode(req)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, res)
}

func (s *Server) handleCashCodeRedeem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if err := s.b.ensureOnlineBank(); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	var req ledger.CashCodeRedeemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]string{"error": "invalid json"})
		return
	}
	res, err := s.b.RedeemCashCode(req)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, res)
}

func (s *Server) handleCashCodeVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Code string `json:"code"`
		PIN  string `json:"pin,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]string{"error": "invalid json"})
		return
	}
	res, err := s.b.VerifyCashCode(req.Code, req.PIN)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, res)
}

func (s *Server) handleCashCodeCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID            string `json:"id"`
		IssuerAccount string `json:"issuerAccount,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]string{"error": "invalid json"})
		return
	}
	cc, err := s.b.CancelCashCode(req.ID, req.IssuerAccount)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"status": "cancelled", "cashCode": cc})
}
