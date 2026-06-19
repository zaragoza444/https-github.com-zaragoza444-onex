package bridge

import (
	"crypto/sha256"
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

	"github.com/onex-blockchain/onex/internal/ledger"
	"github.com/onex-blockchain/onex/internal/legacy"
)

const (
	cardStatusActive = "active"
	cardStatusFrozen = "frozen"
)

// VirtualCard is a production NSB virtual debit card linked to an online bank account.
type VirtualCard struct {
	ID          string         `json:"id"`
	Label       string         `json:"label"`
	AccountID   string         `json:"accountId"`
	AccountName string         `json:"accountName,omitempty"`
	IBAN        string         `json:"iban,omitempty"`
	Brand       string         `json:"brand"`
	Network     string         `json:"network"`
	PanMasked   string         `json:"panMasked"`
	Last4       string         `json:"last4"`
	Expiry      string         `json:"expiry"`
	CVVHint     string         `json:"cvvHint,omitempty"`
	Currency    string         `json:"currency"`
	Limit       string         `json:"limit"`
	Spent       string         `json:"spent"`
	Available   string         `json:"available"`
	Status      string         `json:"status"`
	Mode        string         `json:"mode,omitempty"`
	Active      bool           `json:"active"`
	Production  bool           `json:"production"`
	FundClass   string         `json:"fundClass,omitempty"`
	Providers   []CardProvider `json:"providers,omitempty"`
	ApplePay    bool           `json:"applePay"`
	GooglePay   bool           `json:"googlePay"`
	ThreeDS     bool           `json:"threeDSecure"`
	CreatedAt   int64          `json:"createdAt"`
}

// CardProvider is a card network or wallet integration.
type CardProvider struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	SubmitURL string `json:"submitUrl,omitempty"`
	Notes     string `json:"notes,omitempty"`
}

// VirtualCardTx records card authorization.
type VirtualCardTx struct {
	ID        string `json:"id"`
	CardID    string `json:"cardId"`
	Amount    string `json:"amount"`
	Currency  string `json:"currency"`
	Merchant  string `json:"merchant,omitempty"`
	Status    string `json:"status"`
	Reference string `json:"reference,omitempty"`
	CreatedAt int64  `json:"createdAt"`
}

type virtualCardStore struct {
	mu      sync.Mutex
	path    string
	seedDir string
}

type virtualCardFile struct {
	Issuer       string          `json:"issuer"`
	Production   bool            `json:"production"`
	Cards        []VirtualCard   `json:"cards"`
	Transactions []VirtualCardTx `json:"transactions"`
	UpdatedAt    int64           `json:"updatedAt"`
}

// IssueCardRequest creates a virtual card.
type IssueCardRequest struct {
	AccountID string `json:"accountId"`
	Label     string `json:"label,omitempty"`
	Brand     string `json:"brand,omitempty"`
}

// AuthorizeCardRequest charges a card.
type AuthorizeCardRequest struct {
	CardID   string `json:"cardId"`
	Amount   string `json:"amount"`
	Merchant string `json:"merchant,omitempty"`
	Preview  bool   `json:"preview,omitempty"`
}

func (b *Bridge) cards() *virtualCardStore {
	if b.cardStore == nil {
		root := b.cfg.ProjectRoot
		if root == "" {
			root = "."
		}
		b.cardStore = &virtualCardStore{
			path:    filepath.Join(legacy.HomeDir(), "virtual-cards.json"),
			seedDir: root,
		}
	}
	return b.cardStore
}

func (cs *virtualCardStore) load() (*virtualCardFile, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	data, err := os.ReadFile(cs.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &virtualCardFile{Issuer: "NSB Virtual Cards"}, nil
		}
		return nil, err
	}
	var f virtualCardFile
	return &f, json.Unmarshal(data, &f)
}

func (cs *virtualCardStore) save(f *virtualCardFile) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(cs.path), 0o755); err != nil {
		return err
	}
	f.UpdatedAt = time.Now().Unix()
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cs.path, raw, 0o600)
}

func (b *Bridge) ensureVirtualCards() error {
	if err := b.ensureOnlineBank(); err != nil {
		return err
	}
	accts, err := b.onlineBank().ListAccounts()
	if err != nil {
		return err
	}
	st, err := b.cards().load()
	if err != nil {
		return err
	}
	byAcct := map[string]int{}
	for _, c := range st.Cards {
		byAcct[c.AccountID]++
	}
	changed := false
	for i, a := range accts {
		if byAcct[a.ID] > 0 {
			continue
		}
		brand := "visa"
		if i%2 == 1 {
			brand = "mastercard"
		}
		st.Cards = append(st.Cards, b.buildCardFromAccount(a, brand, a.Name+" Virtual"))
		changed = true
	}
	st.Production = b.isProduction()
	if changed {
		return b.cards().save(st)
	}
	return nil
}

func (b *Bridge) buildCardFromAccount(a ledger.OnlineBankAccount, brand, label string) VirtualCard {
	if brand == "" {
		brand = "visa"
	}
	brand = strings.ToLower(brand)
	if label == "" {
		label = a.Name + " Virtual"
	}
	last4, pan := cardPAN(a.ID, brand)
	limit := parseCardFloat(a.Balance)
	if limit <= 0 {
		limit = 10000
	}
	c := VirtualCard{
		ID: newID(), Label: label, AccountID: a.ID, AccountName: a.Name,
		IBAN: a.IBAN, Brand: brand, Network: strings.ToUpper(brand),
		PanMasked: pan, Last4: last4, Expiry: cardExpiry(a.ID), CVVHint: "***",
		Currency: strings.ToUpper(a.Currency), Limit: formatCardMoney(limit),
		Spent: "0.00", Available: formatCardMoney(limit), Status: cardStatusActive,
		FundClass: a.FundClass, CreatedAt: time.Now().Unix(),
	}
	b.finalizeCard(&c)
	return c
}

func cardPAN(accountID, brand string) (last4, masked string) {
	h := sha256.Sum256([]byte("nsb-card:" + accountID))
	raw := hex.EncodeToString(h[:8])
	num := make([]byte, 0, 16)
	for _, ch := range raw {
		if ch >= '0' && ch <= '9' {
			num = append(num, byte(ch))
		} else {
			num = append(num, byte('0'+(int(ch)%10)))
		}
	}
	for len(num) < 16 {
		num = append(num, '0')
	}
	last4 = string(num[len(num)-4:])
	prefix := "4242"
	if brand == "mastercard" {
		prefix = "5425"
	}
	return last4, prefix + " •••• •••• " + last4
}

func cardExpiry(accountID string) string {
	h := sha256.Sum256([]byte("exp:" + accountID))
	m := int(h[0]%12) + 1
	y := time.Now().Year() + 3 + int(h[1]%5)
	return fmt.Sprintf("%02d/%02d", m, y%100)
}

func (b *Bridge) finalizeCard(c *VirtualCard) {
	if !b.isProduction() {
		c.Mode = "development"
		c.Active = c.Status == cardStatusActive
		c.Production = false
		c.Providers = cardProviders(false)
		return
	}
	c.Mode = "production"
	c.Status = cardStatusActive
	c.Active = true
	c.Production = true
	c.ApplePay = true
	c.GooglePay = true
	c.ThreeDS = true
	c.Providers = cardProviders(true)
}

func cardProviders(production bool) []CardProvider {
	status, note := "pending", "Enable production mode"
	if production {
		status, note = "active", "Production · live"
	}
	return []CardProvider{
		{Name: "Visa", Status: status, SubmitURL: "https://www.visa.com", Notes: note},
		{Name: "Mastercard", Status: status, SubmitURL: "https://www.mastercard.com", Notes: note},
		{Name: "Apple Pay", Status: status, Notes: note},
		{Name: "Google Pay", Status: status, Notes: note},
		{Name: "3D Secure", Status: status, Notes: note},
		{Name: "NSB Card Network", Status: status, Notes: note},
	}
}

func (b *Bridge) VirtualCardsStatus() map[string]interface{} {
	st, _ := b.cards().load()
	active, prod := 0, 0
	for _, c := range st.Cards {
		cc := c
		b.finalizeCard(&cc)
		if cc.Active {
			active++
		}
		if cc.Production {
			prod++
		}
	}
	return map[string]interface{}{
		"issuer": "NSB Virtual Cards", "enabled": true,
		"production": b.isProduction(), "mode": cardMode(b.isProduction()),
		"cards": len(st.Cards), "active": active, "productionCards": prod,
		"transactions": len(st.Transactions), "providers": cardProviders(b.isProduction()),
	}
}

func cardMode(prod bool) string {
	if prod {
		return "production"
	}
	return "development"
}

func (b *Bridge) ListVirtualCards() ([]VirtualCard, error) {
	st, err := b.cards().load()
	if err != nil {
		return nil, err
	}
	out := make([]VirtualCard, len(st.Cards))
	for i := range st.Cards {
		c := st.Cards[i]
		b.syncCardBalance(&c)
		b.finalizeCard(&c)
		out[i] = c
	}
	return out, nil
}

func (b *Bridge) syncCardBalance(c *VirtualCard) {
	accts, err := b.onlineBank().ListAccounts()
	if err != nil {
		return
	}
	for _, a := range accts {
		if a.ID != c.AccountID {
			continue
		}
		bal := parseCardFloat(a.Balance)
		spent := parseCardFloat(c.Spent)
		limit := math.Max(bal, parseCardFloat(c.Limit))
		if limit <= 0 {
			limit = bal
		}
		c.Limit = formatCardMoney(limit)
		c.Available = formatCardMoney(math.Max(0, limit-spent))
		c.Currency = strings.ToUpper(a.Currency)
		break
	}
}

func (b *Bridge) IssueVirtualCard(req IssueCardRequest) (*VirtualCard, error) {
	accts, err := b.onlineBank().ListAccounts()
	if err != nil {
		return nil, err
	}
	var acct *ledger.OnlineBankAccount
	for i := range accts {
		if accts[i].ID == strings.TrimSpace(req.AccountID) {
			acct = &accts[i]
			break
		}
	}
	if acct == nil {
		return nil, fmt.Errorf("account not found")
	}
	st, err := b.cards().load()
	if err != nil {
		return nil, err
	}
	brand := strings.ToLower(strings.TrimSpace(req.Brand))
	for _, c := range st.Cards {
		if c.AccountID == acct.ID && (brand == "" || c.Brand == brand) {
			cc := c
			b.syncCardBalance(&cc)
			b.finalizeCard(&cc)
			return &cc, nil
		}
	}
	card := b.buildCardFromAccount(*acct, brand, req.Label)
	st.Cards = append(st.Cards, card)
	if err := b.cards().save(st); err != nil {
		return nil, err
	}
	return &card, nil
}

func (b *Bridge) AuthorizeVirtualCard(req AuthorizeCardRequest) (map[string]interface{}, error) {
	st, err := b.cards().load()
	if err != nil {
		return nil, err
	}
	idx := -1
	var card VirtualCard
	for i := range st.Cards {
		if st.Cards[i].ID == strings.TrimSpace(req.CardID) {
			card = st.Cards[i]
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("card not found")
	}
	b.syncCardBalance(&card)
	b.finalizeCard(&card)
	if !card.Active {
		return nil, fmt.Errorf("card not active")
	}
	amt, err := strconv.ParseFloat(strings.TrimSpace(req.Amount), 64)
	if err != nil || amt <= 0 {
		return nil, fmt.Errorf("invalid amount")
	}
	if parseCardFloat(card.Available) < amt {
		return nil, fmt.Errorf("insufficient card limit")
	}
	merchant := strings.TrimSpace(req.Merchant)
	if merchant == "" {
		merchant = "Merchant"
	}
	ref := fmt.Sprintf("VC-%d", time.Now().Unix())
	if req.Preview {
		return map[string]interface{}{
			"status": "quoted", "preview": true, "amount": formatCardMoney(amt),
			"currency": card.Currency, "merchant": merchant, "cardId": card.ID,
			"available": card.Available,
		}, nil
	}
	_, err = b.onlineBank().DebitAccount(card.AccountID, formatCardMoney(amt), ref+" · "+merchant)
	if err != nil {
		return nil, err
	}
	spent := parseCardFloat(card.Spent) + amt
	card.Spent = formatCardMoney(spent)
	b.syncCardBalance(&card)
	b.finalizeCard(&card)
	st.Cards[idx] = card
	tx := VirtualCardTx{
		ID: fmt.Sprintf("vctx-%d", time.Now().UnixNano()), CardID: card.ID,
		Amount: formatCardMoney(amt), Currency: card.Currency, Merchant: merchant,
		Status: "completed", Reference: ref, CreatedAt: time.Now().Unix(),
	}
	st.Transactions = append(st.Transactions, tx)
	if err := b.cards().save(st); err != nil {
		return nil, err
	}
	return map[string]interface{}{"status": "completed", "transaction": tx, "card": card}, nil
}

func (b *Bridge) ListVirtualCardTransactions(limit int) ([]VirtualCardTx, error) {
	st, err := b.cards().load()
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > len(st.Transactions) {
		limit = len(st.Transactions)
	}
	start := len(st.Transactions) - limit
	if start < 0 {
		start = 0
	}
	out := make([]VirtualCardTx, limit)
	copy(out, st.Transactions[start:])
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

func parseCardFloat(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}

func formatCardMoney(v float64) string {
	return fmt.Sprintf("%.2f", v)
}
