package ledger

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/onex-blockchain/onex/internal/legacy"
)

const (
	PaymentFlowDonation   = "donation"
	PaymentFlowPayment    = "payment"
	PaymentFlowCollection = "collection"

	FrameworkNova  = "nova"
	FrameworkZBank = "zbank"
	FrameworkNSB   = "nsb"

	SettlementTypeInternal = "internal"
	SettlementTypeExternal = "external"

	PaymentStatusPending   = "pending"
	PaymentStatusProcessing = "processing"
	PaymentStatusSucceeded = "succeeded"
	PaymentStatusFailed    = "failed"
	PaymentStatusRefunded  = "refunded"

	CardVisa       = "visa"
	CardMastercard = "mastercard"
	CardAmex       = "amex"
)

// ProcessingFeeConfig is an optional gateway-level administration fee.
type ProcessingFeeConfig struct {
	Enabled  bool   `json:"enabled"`
	Percent  string `json:"percent,omitempty"`  // e.g. "1.5" = 1.5%
	Fixed    string `json:"fixed,omitempty"`    // flat amount in fee currency
	Currency string `json:"currency,omitempty"` // defaults to session currency
}

// SettlementDestination routes cleared funds to an internal or external bank account.
type SettlementDestination struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Type      string `json:"type"` // internal | external
	AccountID string `json:"accountId,omitempty"`
	BankName  string `json:"bankName,omitempty"`
	IBAN      string `json:"iban,omitempty"`
	SWIFT     string `json:"swift,omitempty"`
	AccountNo string `json:"accountNo,omitempty"`
	RoutingNo string `json:"routingNo,omitempty"`
	Currency  string `json:"currency"`
	Rail      string `json:"rail,omitempty"` // ach, wire, sepa, swift, fps, internal
	Country   string `json:"country,omitempty"`
	Active    bool   `json:"active"`
}

// PaymentPage defines a hosted donation, payment, or collection page.
type PaymentPage struct {
	Slug                   string               `json:"slug"`
	Flow                   string               `json:"flow"` // donation | payment | collection
	Title                  string               `json:"title"`
	Description            string               `json:"description,omitempty"`
	SettlementDestination  string               `json:"settlementDestination"`
	Currency               string               `json:"currency"`
	SuggestedAmounts       []float64            `json:"suggestedAmounts,omitempty"`
	AllowCustomAmount      bool                 `json:"allowCustomAmount"`
	MinAmount              string               `json:"minAmount,omitempty"`
	MaxAmount              string               `json:"maxAmount,omitempty"`
	ProcessingFee          *ProcessingFeeConfig `json:"processingFee,omitempty"`
	ReferencePrefix        string               `json:"referencePrefix,omitempty"`
	Active                 bool                 `json:"active"`
}

// PaymentGatewayConfig is the gateway configuration loaded from file + env.
type PaymentGatewayConfig struct {
	Framework                    string                  `json:"framework"` // nova | zbank | nsb
	DisplayName                  string                  `json:"displayName"`
	Provider                     string                  `json:"provider"` // mock | stripe
	DefaultSettlementDestination string                  `json:"defaultSettlementDestination"`
	ProcessingFee                ProcessingFeeConfig     `json:"processingFee"`
	AcceptedCards                []string                `json:"acceptedCards"`
	SettlementDestinations       []SettlementDestination `json:"settlementDestinations"`
	Pages                        []PaymentPage           `json:"pages"`

	StripeSecretKey      string `json:"-"`
	StripePublishableKey string `json:"-"`
	StripeWebhookSecret  string `json:"-"`
}

// PaymentSession is a single card payment attempt.
type PaymentSession struct {
	ID                    string  `json:"id"`
	PageSlug              string  `json:"pageSlug,omitempty"`
	Flow                  string  `json:"flow"`
	Amount                string  `json:"amount"`
	Currency              string  `json:"currency"`
	ProcessingFee         string  `json:"processingFee,omitempty"`
	TotalCharged          string  `json:"totalCharged"`
	SettlementDestination string  `json:"settlementDestination"`
	SettlementLabel       string  `json:"settlementLabel,omitempty"`
	CardBrand             string  `json:"cardBrand,omitempty"`
	Status                string  `json:"status"`
	Provider              string  `json:"provider"`
	ProviderRef           string  `json:"providerRef,omitempty"`
	ClientSecret          string  `json:"clientSecret,omitempty"`
	PayerEmail            string  `json:"payerEmail,omitempty"`
	PayerName             string  `json:"payerName,omitempty"`
	Reference             string  `json:"reference,omitempty"`
	Memo                  string  `json:"memo,omitempty"`
	SettlementRef         string  `json:"settlementRef,omitempty"`
	LedgerDepositID       string  `json:"ledgerDepositId,omitempty"`
	Framework             string  `json:"framework"`
	CreatedAt             int64   `json:"createdAt"`
	CompletedAt           int64   `json:"completedAt,omitempty"`
}

// PaymentGatewayState is persisted gateway state (sessions + config snapshot).
type PaymentGatewayState struct {
	Sessions  []PaymentSession `json:"sessions"`
	UpdatedAt int64            `json:"updatedAt"`
}

// CreatePaymentSessionRequest initiates a hosted payment.
type CreatePaymentSessionRequest struct {
	PageSlug    string  `json:"pageSlug,omitempty"`
	Flow        string  `json:"flow,omitempty"`
	Amount      string  `json:"amount"`
	Currency    string  `json:"currency,omitempty"`
	Destination string  `json:"settlementDestination,omitempty"`
	PayerEmail  string  `json:"payerEmail,omitempty"`
	PayerName   string  `json:"payerName,omitempty"`
	Reference   string  `json:"reference,omitempty"`
	Memo        string  `json:"memo,omitempty"`
	ApplyFee    *bool   `json:"applyFee,omitempty"`
	CardBrand   string  `json:"cardBrand,omitempty"`
	Preview     bool    `json:"preview,omitempty"`
}

// ConfirmPaymentRequest completes a payment (mock or post-3DS).
type ConfirmPaymentRequest struct {
	SessionID     string `json:"sessionId"`
	ProviderRef   string `json:"providerRef,omitempty"`
	CardLast4     string `json:"cardLast4,omitempty"`
	CardBrand     string `json:"cardBrand,omitempty"`
	PaymentMethod string `json:"paymentMethodId,omitempty"`
}

// PaymentGatewayStore persists payment sessions.
type PaymentGatewayStore struct {
	mu   sync.Mutex
	path string
}

func PaymentGatewayEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(legacy.EnvOrLegacy("ONEX_PAYMENT_GATEWAY", "SHIVA_PAYMENT_GATEWAY")))
	if v == "0" || v == "false" || v == "off" {
		return false
	}
	if v == "1" || v == "true" || v == "on" {
		return true
	}
	cfg := LoadConfig()
	return cfg.Production() || OnlineBankEnabled()
}

func DefaultPaymentGatewayStore() *PaymentGatewayStore {
	return &PaymentGatewayStore{path: filepath.Join(legacy.HomeDir(), "payment-gateway.json")}
}

func LoadPaymentGatewayConfig() PaymentGatewayConfig {
	cfg := PaymentGatewayConfig{
		Framework:     FrameworkNova,
		DisplayName:   "Nova Bank Payment Gateway",
		Provider:      "mock",
		AcceptedCards: []string{CardVisa, CardMastercard, CardAmex},
		ProcessingFee: ProcessingFeeConfig{Enabled: false, Percent: "1.5", Fixed: "0.30", Currency: "USD"},
	}
	file := strings.TrimSpace(legacy.EnvOrLegacy("ONEX_PAYMENT_GATEWAY_FILE", "SHIVA_PAYMENT_GATEWAY_FILE"))
	if file == "" {
		file = filepath.Join(legacy.HomeDir(), "payment-gateway.config.json")
	}
	if data, err := os.ReadFile(file); err == nil {
		_ = json.Unmarshal(data, &cfg)
	}
	if p := strings.TrimSpace(legacy.EnvOrLegacy("ONEX_PAYMENT_GATEWAY_PROVIDER", "SHIVA_PAYMENT_GATEWAY_PROVIDER")); p != "" {
		cfg.Provider = p
	}
	if f := strings.TrimSpace(legacy.EnvOrLegacy("ONEX_PAYMENT_GATEWAY_FRAMEWORK", "SHIVA_PAYMENT_GATEWAY_FRAMEWORK")); f != "" {
		cfg.Framework = f
	}
	cfg.StripeSecretKey = legacy.EnvOrLegacy("ONEX_STRIPE_SECRET_KEY", "SHIVA_STRIPE_SECRET_KEY")
	cfg.StripePublishableKey = legacy.EnvOrLegacy("ONEX_STRIPE_PUBLISHABLE_KEY", "SHIVA_STRIPE_PUBLISHABLE_KEY")
	cfg.StripeWebhookSecret = legacy.EnvOrLegacy("ONEX_STRIPE_WEBHOOK_SECRET", "SHIVA_STRIPE_WEBHOOK_SECRET")
	if cfg.StripeSecretKey != "" && cfg.Provider == "mock" {
		cfg.Provider = "stripe"
	}
	if strings.EqualFold(cfg.Provider, "stripe") && cfg.StripeSecretKey == "" {
		cfg.Provider = "mock"
	}
	cfg.applyFrameworkDefaults()
	return cfg
}

func (c *PaymentGatewayConfig) applyFrameworkDefaults() {
	switch strings.ToLower(strings.TrimSpace(c.Framework)) {
	case FrameworkZBank:
		if c.DisplayName == "" || c.DisplayName == "Nova Bank Payment Gateway" {
			c.DisplayName = "Z Bank Payment Gateway"
		}
	case FrameworkNSB:
		if c.DisplayName == "" || c.DisplayName == "Nova Bank Payment Gateway" {
			c.DisplayName = "NSB Payment Gateway"
		}
	default:
		c.Framework = FrameworkNova
		if c.DisplayName == "" {
			c.DisplayName = "Nova Bank Payment Gateway"
		}
	}
}

func (c PaymentGatewayConfig) Status() map[string]interface{} {
	dests := 0
	for _, d := range c.SettlementDestinations {
		if d.Active {
			dests++
		}
	}
	pages := 0
	for _, p := range c.Pages {
		if p.Active {
			pages++
		}
	}
	return map[string]interface{}{
		"enabled":              PaymentGatewayEnabled(),
		"framework":            c.Framework,
		"displayName":          c.DisplayName,
		"provider":             c.Provider,
		"stripeConfigured":     c.StripeSecretKey != "",
		"stripeLiveReady":      c.StripeSecretKey != "" && c.StripePublishableKey != "" && c.StripeWebhookSecret != "",
		"acceptedCards":        c.AcceptedCards,
		"processingFee":        c.ProcessingFee,
		"settlementDestinations": dests,
		"activePages":          pages,
		"defaultSettlement":    c.DefaultSettlementDestination,
	}
}

func (c PaymentGatewayConfig) FindPage(slug string) (*PaymentPage, error) {
	slug = strings.TrimSpace(slug)
	for i := range c.Pages {
		p := &c.Pages[i]
		if p.Active && strings.EqualFold(p.Slug, slug) {
			return p, nil
		}
	}
	return nil, fmt.Errorf("payment page not found: %s", slug)
}

func (c PaymentGatewayConfig) FindDestination(id string) (*SettlementDestination, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		id = c.DefaultSettlementDestination
	}
	for i := range c.SettlementDestinations {
		d := &c.SettlementDestinations[i]
		if d.Active && strings.EqualFold(d.ID, id) {
			return d, nil
		}
	}
	return nil, fmt.Errorf("settlement destination not found: %s", id)
}

func (c PaymentGatewayConfig) ResolveFee(amount, currency string, page *PaymentPage, applyFee *bool) (fee string, enabled bool) {
	feeCfg := c.ProcessingFee
	if page != nil && page.ProcessingFee != nil {
		feeCfg = *page.ProcessingFee
	}
	if applyFee != nil {
		enabled = *applyFee
	} else {
		enabled = feeCfg.Enabled
	}
	if !enabled {
		return "0", false
	}
	amt, err := parseMoney(amount)
	if err != nil || amt <= 0 {
		return "0", false
	}
	pct, _ := parseMoney(feeCfg.Percent)
	fixed, _ := parseMoney(feeCfg.Fixed)
	feeAmt := amt*pct/100 + fixed
	if feeAmt < 0 {
		feeAmt = 0
	}
	return formatMoney(feeAmt), true
}

func (c PaymentGatewayConfig) CardAccepted(brand string) bool {
	brand = normalizeCardBrand(brand)
	if brand == "" {
		return true
	}
	if len(c.AcceptedCards) == 0 {
		return brand == CardVisa || brand == CardMastercard || brand == CardAmex
	}
	for _, a := range c.AcceptedCards {
		if normalizeCardBrand(a) == brand {
			return true
		}
	}
	return false
}

func (s *PaymentGatewayStore) load() (*PaymentGatewayState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &PaymentGatewayState{Sessions: []PaymentSession{}}, nil
		}
		return nil, err
	}
	var st PaymentGatewayState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	if st.Sessions == nil {
		st.Sessions = []PaymentSession{}
	}
	return &st, nil
}

func (s *PaymentGatewayStore) save(st *PaymentGatewayState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	st.UpdatedAt = time.Now().Unix()
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

func (s *PaymentGatewayStore) GetSession(id string) (*PaymentSession, error) {
	st, err := s.load()
	if err != nil {
		return nil, err
	}
	for i := range st.Sessions {
		if st.Sessions[i].ID == id {
			copy := st.Sessions[i]
			return &copy, nil
		}
	}
	return nil, fmt.Errorf("payment session not found")
}

func (s *PaymentGatewayStore) ListSessions(limit int, flow, status string) ([]PaymentSession, error) {
	st, err := s.load()
	if err != nil {
		return nil, err
	}
	out := make([]PaymentSession, 0, len(st.Sessions))
	for i := len(st.Sessions) - 1; i >= 0; i-- {
		sess := st.Sessions[i]
		if flow != "" && !strings.EqualFold(sess.Flow, flow) {
			continue
		}
		if status != "" && !strings.EqualFold(sess.Status, status) {
			continue
		}
		out = append(out, sess)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *PaymentGatewayStore) CreateSession(req CreatePaymentSessionRequest, bank *OnlineBankStore) (*PaymentSession, error) {
	cfg := LoadPaymentGatewayConfig()
	var page *PaymentPage
	flow := strings.TrimSpace(req.Flow)
	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	destID := strings.TrimSpace(req.Destination)

	if req.PageSlug != "" {
		p, err := cfg.FindPage(req.PageSlug)
		if err != nil {
			return nil, err
		}
		page = p
		flow = p.Flow
		if currency == "" {
			currency = strings.ToUpper(p.Currency)
		}
		if destID == "" {
			destID = p.SettlementDestination
		}
	}
	if flow == "" {
		flow = PaymentFlowPayment
	}
	if currency == "" {
		currency = "USD"
	}
	amount, err := parseMoney(req.Amount)
	if err != nil || amount <= 0 {
		return nil, fmt.Errorf("invalid amount")
	}
	if page != nil {
		if min, _ := parseMoney(page.MinAmount); min > 0 && amount < min {
			return nil, fmt.Errorf("amount below minimum %s", page.MinAmount)
		}
		if max, _ := parseMoney(page.MaxAmount); max > 0 && amount > max {
			return nil, fmt.Errorf("amount above maximum %s", page.MaxAmount)
		}
	}
	dest, err := cfg.FindDestination(destID)
	if err != nil {
		return nil, err
	}
	if dest.Type == SettlementTypeInternal && bank != nil {
		if _, err := bank.GetAccount(dest.AccountID); err != nil {
			return nil, fmt.Errorf("settlement account unavailable: %w", err)
		}
	}
	feeStr, _ := cfg.ResolveFee(req.Amount, currency, page, req.ApplyFee)
	feeAmt, _ := parseMoney(feeStr)
	total := amount + feeAmt

	if req.CardBrand != "" && !cfg.CardAccepted(req.CardBrand) {
		return nil, fmt.Errorf("card brand not accepted: %s", req.CardBrand)
	}

	ref := strings.TrimSpace(req.Reference)
	if ref == "" && page != nil && page.ReferencePrefix != "" {
		ref = page.ReferencePrefix + "-" + newPaymentID()[4:12]
	}

	sess := PaymentSession{
		ID:                    newPaymentID(),
		PageSlug:              req.PageSlug,
		Flow:                  flow,
		Amount:                formatMoney(amount),
		Currency:              currency,
		ProcessingFee:         feeStr,
		TotalCharged:          formatMoney(total),
		SettlementDestination: dest.ID,
		SettlementLabel:       dest.Label,
		Status:                PaymentStatusPending,
		Provider:              cfg.Provider,
		PayerEmail:            strings.TrimSpace(req.PayerEmail),
		PayerName:             strings.TrimSpace(req.PayerName),
		Reference:             ref,
		Memo:                  strings.TrimSpace(req.Memo),
		CardBrand:             normalizeCardBrand(req.CardBrand),
		Framework:             cfg.Framework,
		CreatedAt:             time.Now().Unix(),
	}

	if req.Preview {
		return &sess, nil
	}

	provider := ResolvePaymentProvider(cfg)
	intent, err := provider.CreateIntent(sess, cfg)
	if err != nil {
		return nil, err
	}
	sess.ProviderRef = intent.ProviderRef
	sess.ClientSecret = intent.ClientSecret
	sess.Status = PaymentStatusProcessing

	st, err := s.load()
	if err != nil {
		return nil, err
	}
	st.Sessions = append(st.Sessions, sess)
	if err := s.save(st); err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *PaymentGatewayStore) ConfirmSession(req ConfirmPaymentRequest, bank *OnlineBankStore) (*PaymentSession, error) {
	st, err := s.load()
	if err != nil {
		return nil, err
	}
	idx := -1
	for i := range st.Sessions {
		if st.Sessions[i].ID == req.SessionID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("payment session not found")
	}
	sess := st.Sessions[idx]
	if sess.Status == PaymentStatusSucceeded {
		return &sess, nil
	}
	cfg := LoadPaymentGatewayConfig()
	provider := ResolvePaymentProvider(cfg)
	ref := firstNonEmpty(req.ProviderRef, sess.ProviderRef)
	if err := provider.ConfirmPayment(ref, req); err != nil {
		sess.Status = PaymentStatusFailed
		_ = s.save(st)
		return nil, err
	}
	if req.CardBrand != "" {
		sess.CardBrand = normalizeCardBrand(req.CardBrand)
	}
	sess.Status = PaymentStatusSucceeded
	sess.CompletedAt = time.Now().Unix()

	settleRef, depID, err := s.settleToDestination(sess, cfg, bank)
	if err != nil {
		sess.Status = PaymentStatusFailed
		st.Sessions[idx] = sess
		_ = s.save(st)
		return nil, fmt.Errorf("settlement failed: %w", err)
	}
	sess.SettlementRef = settleRef
	sess.LedgerDepositID = depID
	st.Sessions[idx] = sess
	if err := s.save(st); err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *PaymentGatewayStore) settleToDestination(sess PaymentSession, cfg PaymentGatewayConfig, bank *OnlineBankStore) (settlementRef, depositID string, err error) {
	dest, err := cfg.FindDestination(sess.SettlementDestination)
	if err != nil {
		return "", "", err
	}
	settlementRef = fmt.Sprintf("PGW-%s-%s", dest.ID, sess.ID[4:12])
	source := "card:" + firstNonEmpty(sess.CardBrand, "card")

	switch dest.Type {
	case SettlementTypeInternal:
		if bank == nil {
			return settlementRef, "", fmt.Errorf("online bank unavailable")
		}
		res, err := bank.Deposit(OnlineBankDepositRequest{
			ToAccount: dest.AccountID,
			Amount:    sess.Amount,
			Source:    source,
			Reference: firstNonEmpty(sess.Reference, settlementRef),
		})
		if err != nil {
			return "", "", err
		}
		if res.Transaction != nil {
			depositID = res.Transaction.ID
		}
		return settlementRef, depositID, nil

	case SettlementTypeExternal:
		// Credit gateway clearing account then queue external settlement via middleware.
		clearingID := cfg.defaultClearingAccount(dest.Currency)
		if bank != nil && clearingID != "" {
			if _, err := bank.GetAccount(clearingID); err == nil {
				res, depErr := bank.Deposit(OnlineBankDepositRequest{
					ToAccount: clearingID,
					Amount:    sess.Amount,
					Source:    source,
					Reference: settlementRef,
				})
				if depErr == nil && res.Transaction != nil {
					depositID = res.Transaction.ID
				}
			}
		}
		// Record external settlement intent (real payout via acquiring bank / processor settlement).
		_ = dest
		return settlementRef, depositID, nil
	default:
		return "", "", fmt.Errorf("unknown settlement type: %s", dest.Type)
	}
}

func (c PaymentGatewayConfig) defaultClearingAccount(currency string) string {
	cur := strings.ToUpper(currency)
	for _, d := range c.SettlementDestinations {
		if d.Active && d.Type == SettlementTypeInternal && strings.EqualFold(d.Currency, cur) {
			return d.AccountID
		}
	}
	return c.DefaultSettlementDestination
}

func (c PaymentGatewayConfig) PublicPages() []PaymentPage {
	out := make([]PaymentPage, 0, len(c.Pages))
	for _, p := range c.Pages {
		if p.Active {
			out = append(out, p)
		}
	}
	return out
}

func (c PaymentGatewayConfig) PublicDestinations() []SettlementDestination {
	out := make([]SettlementDestination, 0, len(c.SettlementDestinations))
	for _, d := range c.SettlementDestinations {
		if d.Active {
			pub := d
			pub.IBAN = maskIBAN(d.IBAN)
			pub.AccountNo = maskAccount(d.AccountNo)
			out = append(out, pub)
		}
	}
	return out
}

func newPaymentID() string {
	return fmt.Sprintf("pay_%d_%s", time.Now().UnixNano(), randomHex(4))
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%04x", time.Now().UnixNano()%0xffff)
	}
	return hex.EncodeToString(b)
}

func normalizeCardBrand(brand string) string {
	b := strings.ToLower(strings.TrimSpace(brand))
	switch b {
	case "visa", "vi":
		return CardVisa
	case "mastercard", "mc", "master", "master card":
		return CardMastercard
	case "amex", "american express", "americanexpress":
		return CardAmex
	default:
		return b
	}
}

func parseMoney(s string) (float64, error) {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", ""))
	if s == "" {
		return 0, fmt.Errorf("empty amount")
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("invalid amount")
	}
	return v, nil
}

func formatMoney(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}

func maskIBAN(iban string) string {
	iban = strings.TrimSpace(iban)
	if len(iban) <= 8 {
		return iban
	}
	return iban[:4] + strings.Repeat("*", len(iban)-8) + iban[len(iban)-4:]
}

func maskAccount(acct string) string {
	acct = strings.TrimSpace(acct)
	if len(acct) <= 4 {
		return acct
	}
	return strings.Repeat("*", len(acct)-4) + acct[len(acct)-4:]
}
