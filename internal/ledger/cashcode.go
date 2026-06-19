package ledger

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/onex-blockchain/onex/internal/legacy"
)

const (
	CashCodeEscrowAccountID = "nsb-cashcode-escrow"
	cashCodeStatusActive    = "active"
	cashCodeStatusRedeemed  = "redeemed"
	cashCodeStatusCancelled = "cancelled"
	cashCodeStatusExpired   = "expired"
)

var cashCodeAlphabet = []byte("ABCDEFGHJKMNPQRSTUVWXYZ23456789")

// CashCode is a one-time redeemable cash pickup code.
type CashCode struct {
	ID            string `json:"id"`
	CodeLast4     string `json:"codeLast4"`
	CodeHash      string `json:"codeHash,omitempty"`
	PinHash       string `json:"pinHash,omitempty"`
	HasPIN        bool   `json:"hasPin"`
	Amount        string `json:"amount"`
	Currency      string `json:"currency"`
	Status        string `json:"status"`
	IssuerAccount string `json:"issuerAccount"`
	IssuerName    string `json:"issuerName,omitempty"`
	RedeemAccount string `json:"redeemAccount,omitempty"`
	Memo          string `json:"memo,omitempty"`
	EscrowTxID    string `json:"escrowTxId,omitempty"`
	RedeemTxID    string `json:"redeemTxId,omitempty"`
	ExpiresAt     int64  `json:"expiresAt,omitempty"`
	CreatedAt     int64  `json:"createdAt"`
	RedeemedAt    int64  `json:"redeemedAt,omitempty"`
	CancelledAt   int64  `json:"cancelledAt,omitempty"`
}

// CashCodeIssueRequest creates a new cash code.
type CashCodeIssueRequest struct {
	FromAccount    string `json:"fromAccount"`
	Amount         string `json:"amount"`
	Currency       string `json:"currency,omitempty"`
	PIN            string `json:"pin,omitempty"`
	Memo           string `json:"memo,omitempty"`
	ExpiresInHours int    `json:"expiresInHours,omitempty"`
	Preview        bool   `json:"preview,omitempty"`
}

// CashCodeIssueResult returns the plaintext code once at issue time.
type CashCodeIssueResult struct {
	Status    string    `json:"status"`
	Preview   bool      `json:"preview,omitempty"`
	Code      string    `json:"code,omitempty"`
	CashCode  *CashCode `json:"cashCode,omitempty"`
	EscrowTx  string    `json:"escrowTxId,omitempty"`
}

// CashCodeRedeemRequest redeems a cash code into a bank account.
type CashCodeRedeemRequest struct {
	Code      string `json:"code"`
	PIN       string `json:"pin,omitempty"`
	ToAccount string `json:"toAccount"`
	Preview   bool   `json:"preview,omitempty"`
}

// CashCodeRedeemResult is returned after redemption.
type CashCodeRedeemResult struct {
	Status     string    `json:"status"`
	Preview    bool      `json:"preview,omitempty"`
	CashCode   *CashCode `json:"cashCode,omitempty"`
	RedeemTxID string    `json:"redeemTxId,omitempty"`
	ToBalance  string    `json:"toBalance,omitempty"`
}

// CashCodeVerifyResult previews a code without redeeming.
type CashCodeVerifyResult struct {
	Valid     bool   `json:"valid"`
	Status    string `json:"status"`
	Amount    string `json:"amount,omitempty"`
	Currency  string `json:"currency,omitempty"`
	CodeLast4 string `json:"codeLast4,omitempty"`
	HasPIN    bool   `json:"hasPin"`
	ExpiresAt int64  `json:"expiresAt,omitempty"`
	Memo      string `json:"memo,omitempty"`
}

type cashCodeFile struct {
	Production bool       `json:"production"`
	Codes      []CashCode `json:"codes"`
	UpdatedAt  int64      `json:"updatedAt"`
}

// CashCodeStore persists issued cash codes.
type CashCodeStore struct {
	mu   sync.Mutex
	path string
}

func CashCodeEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(legacy.EnvOrLegacy("ONEX_CASHCODE_ENABLED", "SHIVA_CASHCODE_ENABLED")))
	if v == "0" || v == "false" || v == "off" {
		return false
	}
	if v == "1" || v == "true" || v == "on" {
		return true
	}
	return LoadConfig().Production()
}

func DefaultCashCodeStore() *CashCodeStore {
	return &CashCodeStore{path: filepath.Join(legacy.HomeDir(), "cash-codes.json")}
}

func (s *CashCodeStore) load() (*cashCodeFile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &cashCodeFile{Production: LoadConfig().Production(), Codes: []CashCode{}}, nil
		}
		return nil, err
	}
	var f cashCodeFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return &f, nil
}

func (s *CashCodeStore) save(f *cashCodeFile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	f.UpdatedAt = time.Now().Unix()
	f.Production = LoadConfig().Production()
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0o600)
}

func (s *CashCodeStore) Status() map[string]interface{} {
	f, err := s.load()
	if err != nil {
		return map[string]interface{}{"enabled": CashCodeEnabled(), "error": err.Error()}
	}
	active, redeemed := 0, 0
	var escrowHeld float64
	for _, c := range f.Codes {
		switch c.Status {
		case cashCodeStatusActive:
			active++
			amt, _ := parseCashAmount(c.Amount)
			escrowHeld += amt
		case cashCodeStatusRedeemed:
			redeemed++
		}
	}
	return map[string]interface{}{
		"enabled":    CashCodeEnabled(),
		"production": f.Production,
		"service":    "onex-cashcode",
		"total":      len(f.Codes),
		"active":     active,
		"redeemed":   redeemed,
		"escrowHeld": formatCashAmount(escrowHeld),
		"escrowAcct": CashCodeEscrowAccountID,
		"updatedAt":  f.UpdatedAt,
	}
}

func (s *CashCodeStore) List(accountID string) ([]CashCode, error) {
	f, err := s.load()
	if err != nil {
		return nil, err
	}
	accountID = strings.TrimSpace(accountID)
	out := make([]CashCode, 0, len(f.Codes))
	for _, c := range f.Codes {
		if accountID == "" || c.IssuerAccount == accountID || c.RedeemAccount == accountID {
			c.CodeHash = ""
			c.PinHash = ""
			out = append(out, c)
		}
	}
	return out, nil
}

func (s *CashCodeStore) GetActive(code, pin string) (*CashCode, error) {
	f, idx, err := s.findByCode(code)
	if err != nil {
		return nil, err
	}
	if idx < 0 {
		return nil, fmt.Errorf("invalid cash code")
	}
	c := f.Codes[idx]
	if err := validateCashCodeActive(&c, pin); err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *CashCodeStore) findByCode(code string) (*cashCodeFile, int, error) {
	f, err := s.load()
	if err != nil {
		return nil, -1, err
	}
	hash := hashCashCode(normalizeCashCode(code))
	for i := range f.Codes {
		if f.Codes[i].CodeHash == hash {
			return f, i, nil
		}
	}
	return f, -1, nil
}

func (s *CashCodeStore) Issue(req CashCodeIssueRequest, issuerName string) (*CashCodeIssueResult, error) {
	if !CashCodeEnabled() {
		return nil, fmt.Errorf("cash code system disabled")
	}
	amt, err := parseCashAmount(req.Amount)
	if err != nil {
		return nil, err
	}
	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if currency == "" {
		currency = "USD"
	}
	expiresAt := int64(0)
	hours := req.ExpiresInHours
	if hours <= 0 {
		hours = defaultCashCodeTTLHours()
	}
	if hours > 0 {
		expiresAt = time.Now().Add(time.Duration(hours) * time.Hour).Unix()
	}

	code := generateCashCode()
	last4 := code[len(code)-4:]
	now := time.Now().Unix()
	cc := CashCode{
		ID: newCashCodeID(), CodeLast4: last4, CodeHash: hashCashCode(code),
		Amount: formatCashAmount(amt), Currency: currency, Status: cashCodeStatusActive,
		IssuerAccount: strings.TrimSpace(req.FromAccount), IssuerName: issuerName,
		Memo: strings.TrimSpace(req.Memo), ExpiresAt: expiresAt, CreatedAt: now,
	}
	if pin := strings.TrimSpace(req.PIN); pin != "" {
		if len(pin) < 4 || len(pin) > 8 {
			return nil, fmt.Errorf("pin must be 4-8 digits")
		}
		cc.HasPIN = true
		cc.PinHash = hashCashPIN(pin, cc.CodeHash)
	}

	res := &CashCodeIssueResult{
		Status: "quoted", Preview: req.Preview, CashCode: &cc,
	}
	if req.Preview {
		res.Code = maskCashCode(code)
		return res, nil
	}

	f, err := s.load()
	if err != nil {
		return nil, err
	}
	f.Codes = append(f.Codes, cc)
	if err := s.save(f); err != nil {
		return nil, err
	}
	res.Status = "issued"
	res.Preview = false
	res.Code = code
	return res, nil
}

func (s *CashCodeStore) MarkEscrow(id, txID string) error {
	f, err := s.load()
	if err != nil {
		return err
	}
	for i := range f.Codes {
		if f.Codes[i].ID == id {
			f.Codes[i].EscrowTxID = txID
			return s.save(f)
		}
	}
	return fmt.Errorf("cash code not found")
}

func (s *CashCodeStore) Verify(code, pin string) (*CashCodeVerifyResult, error) {
	f, idx, err := s.findByCode(code)
	if err != nil {
		return nil, err
	}
	if idx < 0 {
		return &CashCodeVerifyResult{Valid: false, Status: "not_found"}, nil
	}
	c := f.Codes[idx]
	if err := validateCashCodeActive(&c, pin); err != nil {
		return &CashCodeVerifyResult{Valid: false, Status: c.Status, Amount: c.Amount, Currency: c.Currency, CodeLast4: c.CodeLast4, HasPIN: c.HasPIN, ExpiresAt: c.ExpiresAt, Memo: c.Memo}, nil
	}
	return &CashCodeVerifyResult{
		Valid: true, Status: c.Status, Amount: c.Amount, Currency: c.Currency,
		CodeLast4: c.CodeLast4, HasPIN: c.HasPIN, ExpiresAt: c.ExpiresAt, Memo: c.Memo,
	}, nil
}

func (s *CashCodeStore) Redeem(req CashCodeRedeemRequest) (*CashCodeRedeemResult, error) {
	if !CashCodeEnabled() {
		return nil, fmt.Errorf("cash code system disabled")
	}
	f, idx, err := s.findByCode(req.Code)
	if err != nil {
		return nil, err
	}
	if idx < 0 {
		return nil, fmt.Errorf("invalid cash code")
	}
	c := f.Codes[idx]
	if err := validateCashCodeActive(&c, req.PIN); err != nil {
		return nil, err
	}
	res := &CashCodeRedeemResult{
		Status: "quoted", Preview: req.Preview, CashCode: &c,
	}
	if req.Preview {
		return res, nil
	}
	now := time.Now().Unix()
	c.Status = cashCodeStatusRedeemed
	c.RedeemAccount = strings.TrimSpace(req.ToAccount)
	c.RedeemedAt = now
	f.Codes[idx] = c
	if err := s.save(f); err != nil {
		return nil, err
	}
	res.Status = "redeemed"
	res.Preview = false
	res.CashCode = &c
	return res, nil
}

func (s *CashCodeStore) MarkRedeemed(id, txID, toAccount string) error {
	f, err := s.load()
	if err != nil {
		return err
	}
	for i := range f.Codes {
		if f.Codes[i].ID == id {
			f.Codes[i].RedeemTxID = txID
			f.Codes[i].RedeemAccount = toAccount
			return s.save(f)
		}
	}
	return fmt.Errorf("cash code not found")
}

func (s *CashCodeStore) Cancel(id, issuerAccount string) (*CashCode, error) {
	f, err := s.load()
	if err != nil {
		return nil, err
	}
	for i := range f.Codes {
		c := &f.Codes[i]
		if c.ID != id {
			continue
		}
		if issuerAccount != "" && c.IssuerAccount != issuerAccount {
			return nil, fmt.Errorf("not authorized to cancel this code")
		}
		if c.Status != cashCodeStatusActive {
			return nil, fmt.Errorf("code is %s", c.Status)
		}
		if c.ExpiresAt > 0 && time.Now().Unix() > c.ExpiresAt {
			c.Status = cashCodeStatusExpired
		} else {
			c.Status = cashCodeStatusCancelled
		}
		c.CancelledAt = time.Now().Unix()
		if err := s.save(f); err != nil {
			return nil, err
		}
		return c, nil
	}
	return nil, fmt.Errorf("cash code not found")
}

func validateCashCodeActive(c *CashCode, pin string) error {
	if c.Status == cashCodeStatusRedeemed {
		return fmt.Errorf("cash code already redeemed")
	}
	if c.Status == cashCodeStatusCancelled {
		return fmt.Errorf("cash code cancelled")
	}
	if c.Status == cashCodeStatusExpired {
		return fmt.Errorf("cash code expired")
	}
	if c.ExpiresAt > 0 && time.Now().Unix() > c.ExpiresAt {
		return fmt.Errorf("cash code expired")
	}
	if c.HasPIN {
		if strings.TrimSpace(pin) == "" {
			return fmt.Errorf("pin required")
		}
		if hashCashPIN(pin, c.CodeHash) != c.PinHash {
			return fmt.Errorf("invalid pin")
		}
	}
	return nil
}

func generateCashCode() string {
	var parts []string
	for p := 0; p < 3; p++ {
		var b strings.Builder
		for i := 0; i < 4; i++ {
			n, err := rand.Int(rand.Reader, big.NewInt(int64(len(cashCodeAlphabet))))
			if err != nil {
				b.WriteByte(cashCodeAlphabet[time.Now().UnixNano()%int64(len(cashCodeAlphabet))])
				continue
			}
			b.WriteByte(cashCodeAlphabet[n.Int64()])
		}
		parts = append(parts, b.String())
	}
	return strings.Join(parts, "-")
}

func normalizeCashCode(code string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), " ", ""))
}

func maskCashCode(code string) string {
	code = normalizeCashCode(code)
	if len(code) < 4 {
		return "****"
	}
	return "****-****-" + code[len(code)-4:]
}

func hashCashCode(code string) string {
	sum := sha256.Sum256([]byte("onex-cashcode-v1:" + normalizeCashCode(code)))
	return hex.EncodeToString(sum[:])
}

func hashCashPIN(pin, codeHash string) string {
	sum := sha256.Sum256([]byte("onex-cashcode-pin:" + strings.TrimSpace(pin) + ":" + codeHash))
	return hex.EncodeToString(sum[:])
}

func newCashCodeID() string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("cc-%d", time.Now().UnixNano())))
	return "cc-" + hex.EncodeToString(sum[:6])
}

func defaultCashCodeTTLHours() int {
	v := strings.TrimSpace(legacy.EnvOrLegacy("ONEX_CASHCODE_TTL_HOURS", "SHIVA_CASHCODE_TTL_HOURS"))
	if v == "" {
		return 72
	}
	var n int
	fmt.Sscanf(v, "%d", &n)
	if n <= 0 {
		return 0
	}
	return n
}

func parseCashAmount(s string) (float64, error) {
	amt, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || amt <= 0 {
		return 0, fmt.Errorf("invalid amount")
	}
	return amt, nil
}

func formatCashAmount(amt float64) string {
	return fmt.Sprintf("%.2f", amt)
}
