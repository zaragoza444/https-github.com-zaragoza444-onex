package bridge

import (
	"encoding/json"
	"net/http"

	"github.com/onex-blockchain/onex/internal/ledger"
)

func (b *Bridge) omnlF20Folder() *ledger.OMNLF20FolderStore {
	return ledger.DefaultOMNLF20FolderStore()
}

func (b *Bridge) OMNLF20Status() map[string]interface{} {
	st := b.omnlF20Folder().Status()
	st["onlineBank"] = ledger.OnlineBankEnabled()
	return st
}

func (b *Bridge) OrderOMNLF20(req ledger.OMNLF20FolderOrderRequest) (*ledger.OMNLF20FolderItem, error) {
	if err := b.ensureOnlineBank(); err != nil {
		return nil, err
	}
	return b.omnlF20Folder().Order(req)
}

func (b *Bridge) LocateOMNLF20(req ledger.OMNLF20FolderLocateRequest) (*ledger.OMNLF20FolderItem, error) {
	if err := b.ensureOnlineBank(); err != nil {
		return nil, err
	}
	return b.omnlF20Folder().Locate(req)
}

func (b *Bridge) ReleaseOMNLF20(req ledger.OMNLF20FolderReleaseRequest) (*ledger.OMNLF20FolderReleaseResult, error) {
	if err := b.ensureOnlineBank(); err != nil {
		return nil, err
	}
	res, err := b.omnlF20Folder().Release(b.onlineBank(), req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *Server) handleOMNLF20Status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, s.b.OMNLF20Status())
}

func (s *Server) handleOMNLF20Order(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req ledger.OMNLF20FolderOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	item, err := s.b.OrderOMNLF20(req)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"status": item.Status, "item": item})
}

func (s *Server) handleOMNLF20Locate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "GET or POST only", http.StatusMethodNotAllowed)
		return
	}
	var req ledger.OMNLF20FolderLocateRequest
	if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		req.F20Number = r.URL.Query().Get("f20")
		req.OutputMessageNumber = r.URL.Query().Get("output")
	}
	item, err := s.b.LocateOMNLF20(req)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"status": item.Status, "item": item})
}

func (s *Server) handleOMNLF20Release(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req ledger.OMNLF20FolderReleaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res, err := s.b.ReleaseOMNLF20(req)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, res)
}
