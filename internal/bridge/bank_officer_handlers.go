package bridge

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/onex-blockchain/onex/internal/ledger"
)

func (s *Server) registerBankOfficerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/bridge/bank/officer/status", s.handleBankOfficerStatus)
	mux.HandleFunc("/bridge/bank/officer/list", s.handleBankOfficerList)
	mux.HandleFunc("/bridge/bank/officer", s.handleBankOfficerGet)
	mux.HandleFunc("/bridge/bank/officer/verify", s.handleBankOfficerVerify)
	mux.HandleFunc("/bridge/bank/officer/transfer", s.handleBankOfficerTransfer)
	mux.HandleFunc("/bridge/bank/officer/ensure", s.handleBankOfficerEnsure)
	mux.HandleFunc("/bridge/bank/officer/credentials", s.handleBankOfficerCredentials)
}

func (b *Bridge) bankOfficers() *ledger.BankOfficerStore {
	return ledger.DefaultBankOfficerStore()
}

func (b *Bridge) officerSeedPath() string {
	seed := ledger.BankOfficerSeedFile()
	if !filepath.IsAbs(seed) {
		seed = filepath.Join(b.projectRoot(), seed)
	}
	return seed
}

func (b *Bridge) ensureBankOfficers() error {
	return b.bankOfficers().EnsureSeeded(b.officerSeedPath())
}

func (s *Server) handleBankOfficerStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	_ = s.b.ensureBankOfficers()
	st := s.b.bankOfficers().Status()
	st["enabled"] = true
	writeJSON(w, st)
}

func (s *Server) handleBankOfficerList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	if err := s.b.ensureBankOfficers(); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	list, err := s.b.bankOfficers().List()
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"officers": list, "count": len(list)})
}

func (s *Server) handleBankOfficerGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	if err := s.b.ensureBankOfficers(); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	o, err := s.b.bankOfficers().Get(id)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, o)
}

func (s *Server) handleBankOfficerVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if err := s.b.ensureBankOfficers(); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	var req ledger.OfficerAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res, err := s.b.bankOfficers().Verify(req)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, res)
}

func (s *Server) handleBankOfficerTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if err := s.b.ensureOnlineBank(); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	if err := s.b.ensureBankOfficers(); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	var req ledger.OfficerTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res, err := s.b.bankOfficers().AuthorizeTransfer(req, s.b.onlineBank())
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	if res.Status == "authorized" && res.Transfer != nil && !req.Preview {
		// Mirror online bank post-hooks when transfer succeeded.
		xfer := ledger.OnlineBankTransferRequest{
			FromAccount: req.FromAccount, ToAccount: req.ToAccount, Amount: req.Amount,
			Rail: req.Rail, ToBank: req.ToBank, ToIBAN: req.ToIBAN, Reference: req.Reference,
		}
		_ = s.b.applyFineractTransfer(xfer, res.Transfer)
		_ = s.b.applyHybrixTransfer(xfer, res.Transfer)
		_ = s.b.SyncLedgerBook(r.Context(), r.URL.Query().Get("evm"))
	}
	writeJSON(w, res)
}

func (s *Server) handleBankOfficerEnsure(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if err := ledger.RequireProductionOfficerSecrets(); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	if err := s.b.bankOfficers().SeedFromEnv(s.b.officerSeedPath()); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	list, _ := s.b.bankOfficers().List()
	writeJSON(w, map[string]interface{}{
		"status": "ok", "production": ledger.LoadConfig().Production(),
		"officers": list, "count": len(list),
	})
}

func (s *Server) handleBankOfficerCredentials(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if err := s.b.ensureBankOfficers(); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	var req ledger.OfficerCredentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if list, _ := s.b.bankOfficers().List(); len(list) == 0 {
		if err := s.b.bankOfficers().SeedFromEnv(s.b.officerSeedPath()); err != nil {
			writeJSON(w, map[string]string{"error": err.Error()})
			return
		}
	}
	pub, err := s.b.bankOfficers().SetCredentials(req)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"status": "updated", "officer": pub})
}
