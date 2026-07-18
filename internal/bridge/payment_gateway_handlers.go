package bridge

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/onex-blockchain/onex/internal/ledger"
)

func (s *Server) registerPaymentGatewayRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/bridge/payments/status", s.handlePaymentGatewayStatus)
	mux.HandleFunc("/bridge/payments/config", s.handlePaymentGatewayConfig)
	mux.HandleFunc("/bridge/payments/pages", s.handlePaymentGatewayPages)
	mux.HandleFunc("/bridge/payments/page", s.handlePaymentGatewayPage)
	mux.HandleFunc("/bridge/payments/destinations", s.handlePaymentGatewayDestinations)
	mux.HandleFunc("/bridge/payments/session", s.handlePaymentGatewaySessionCreate)
	mux.HandleFunc("/bridge/payments/session/get", s.handlePaymentGatewaySessionGet)
	mux.HandleFunc("/bridge/payments/confirm", s.handlePaymentGatewayConfirm)
	mux.HandleFunc("/bridge/payments/sessions", s.handlePaymentGatewaySessionsList)
	mux.HandleFunc("/bridge/payments/dashboard", s.handlePaymentGatewayDashboard)
	mux.HandleFunc("/bridge/payments/webhook", s.handlePaymentGatewayWebhook)
}

func (s *Server) paymentGatewayStore() *ledger.PaymentGatewayStore {
	return ledger.DefaultPaymentGatewayStore()
}

func (s *Server) handlePaymentGatewayStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	cfg := ledger.LoadPaymentGatewayConfig()
	st := cfg.Status()
	st["onlineBank"] = ledger.OnlineBankEnabled()
	writeJSON(w, st)
}

func (s *Server) handlePaymentGatewayConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	cfg := ledger.LoadPaymentGatewayConfig()
	writeJSON(w, map[string]interface{}{
		"framework":       cfg.Framework,
		"displayName":     cfg.DisplayName,
		"logoUrl":         cfg.LogoURL,
		"provider":        cfg.Provider,
		"acceptedCards":   cfg.AcceptedCards,
		"processingFee":   cfg.ProcessingFee,
		"stripePublicKey": cfg.StripePublishableKey,
	})
}

func (s *Server) handlePaymentGatewayPages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	cfg := ledger.LoadPaymentGatewayConfig()
	writeJSON(w, map[string]interface{}{
		"pages": cfg.PublicPages(),
		"count": len(cfg.PublicPages()),
	})
}

func (s *Server) handlePaymentGatewayPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	slug := strings.TrimSpace(r.URL.Query().Get("slug"))
	if slug == "" {
		http.Error(w, "slug required", http.StatusBadRequest)
		return
	}
	cfg := ledger.LoadPaymentGatewayConfig()
	page, err := cfg.FindPage(slug)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	dest, _ := cfg.FindDestination(page.SettlementDestination)
	pub := map[string]interface{}{
		"page": *page,
	}
	if dest != nil {
		pub["settlement"] = map[string]string{
			"id":    dest.ID,
			"label": dest.Label,
			"bank":  dest.BankName,
		}
	}
	writeJSON(w, pub)
}

func (s *Server) handlePaymentGatewayDestinations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	cfg := ledger.LoadPaymentGatewayConfig()
	writeJSON(w, map[string]interface{}{
		"destinations": cfg.PublicDestinations(),
		"count":        len(cfg.PublicDestinations()),
	})
}

func (s *Server) handlePaymentGatewaySessionCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if !ledger.PaymentGatewayEnabled() {
		writeJSON(w, map[string]string{"error": "payment gateway disabled"})
		return
	}
	var req ledger.CreatePaymentSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = s.b.ensureOnlineBank()
	sess, err := s.paymentGatewayStore().CreateSession(req, s.b.onlineBank())
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, sess)
}

func (s *Server) handlePaymentGatewaySessionGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	sess, err := s.paymentGatewayStore().GetSession(id)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, sess)
}

func (s *Server) handlePaymentGatewayConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req ledger.ConfirmPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.SessionID == "" {
		http.Error(w, "sessionId required", http.StatusBadRequest)
		return
	}
	_ = s.b.ensureOnlineBank()
	sess, err := s.paymentGatewayStore().ConfirmSession(req, s.b.onlineBank())
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, sess)
}

func (s *Server) handlePaymentGatewayDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	cfg := ledger.LoadPaymentGatewayConfig()
	stats, err := s.paymentGatewayStore().DashboardStats(25)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{
		"status":       cfg.Status(),
		"stats":        stats,
		"pages":        cfg.PublicPages(),
		"destinations": cfg.PublicDestinations(),
	})
}

func (s *Server) handlePaymentGatewaySessionsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	sessions, err := s.paymentGatewayStore().ListSessions(limit, r.URL.Query().Get("flow"), r.URL.Query().Get("status"))
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"sessions": sessions, "count": len(sessions)})
}

func (s *Server) handlePaymentGatewayWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	cfg := ledger.LoadPaymentGatewayConfig()
	payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	provider := ledger.ResolvePaymentProvider(cfg)
	providerRef, status, err := provider.VerifyWebhook(payload, r.Header.Get("Stripe-Signature"), cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if status != ledger.PaymentStatusSucceeded {
		writeJSON(w, map[string]string{"status": status})
		return
	}
	store := s.paymentGatewayStore()
	st, err := store.ListSessions(100, "", "")
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	var sessionID string
	for _, sess := range st {
		if sess.ProviderRef == providerRef {
			sessionID = sess.ID
			break
		}
	}
	if sessionID == "" {
		writeJSON(w, map[string]string{"status": "ignored", "reason": "session not found"})
		return
	}
	_ = s.b.ensureOnlineBank()
	sess, err := store.ConfirmSession(ledger.ConfirmPaymentRequest{
		SessionID:   sessionID,
		ProviderRef: providerRef,
	}, s.b.onlineBank())
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"status": "ok", "session": sess})
}
