package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/onex-blockchain/onex/internal/legacy"
)

const (
	defaultOnlineBankName  = "NSB — National Sovereign Bank"
	defaultOnlineBankSWIFT = "NSBKLAL2X"
)

// OnlineBankAccount is a live online banking account with IBAN.
type OnlineBankAccount struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	IBAN                 string `json:"iban,omitempty"`
	Currency             string `json:"currency"`
	Balance              string `json:"balance"`
	FundClass            string `json:"fundClass,omitempty"`
	Bank                 string `json:"bank,omitempty"`
	Status               string `json:"status,omitempty"`
	OfficerPINHash       string `json:"officerPinHash,omitempty"`
	OfficerPINRequired   bool   `json:"officerPinRequired,omitempty"`
	OfficerPINConfigured bool   `json:"officerPinConfigured,omitempty"`
}

// OnlineBankTransaction records a transfer on the online bank.
type OnlineBankTransaction struct {
	ID                string `json:"id"`
	Type              string `json:"type"` // internal, iban, sepa, swift, wire, ach, fps
	FromAccount       string `json:"fromAccount"`
	FromName          string `json:"fromName,omitempty"`
	ToAccount         string `json:"toAccount,omitempty"`
	ToName            string `json:"toName,omitempty"`
	ToIBAN            string `json:"toIban,omitempty"`
	ToBank            string `json:"toBank,omitempty"`
	Rail              string `json:"rail,omitempty"`
	Amount            string `json:"amount"`
	Currency          string `json:"currency"`
	Status            string `json:"status"` // completed, pending
	Reference         string `json:"reference,omitempty"`
	Settlement        string `json:"settlement,omitempty"`
	OfficerAuthorized bool   `json:"officerAuthorized,omitempty"`
	CreatedAt         int64  `json:"createdAt"`
}

// OnlineBankState is the persisted online bank book.
type OnlineBankState struct {
	Name         string                  `json:"name"`
	Online       bool                    `json:"online"`
	SWIFT        string                  `json:"swift"`
	Accounts     []OnlineBankAccount     `json:"accounts"`
	Transactions []OnlineBankTransaction `json:"transactions"`
	UpdatedAt    int64                   `json:"updatedAt"`
}

// OnlineBankTransferRequest moves funds internally or to an external IBAN.
type OnlineBankTransferRequest struct {
	FromAccount string `json:"fromAccount"`
	ToAccount   string `json:"toAccount,omitempty"`
	Amount      string `json:"amount"`
	Rail        string `json:"rail,omitempty"`
	ToBank      string `json:"toBank,omitempty"`
	ToIBAN      string `json:"toIban,omitempty"`
	Reference   string `json:"reference,omitempty"`
	OfficerPIN  string `json:"officerPin,omitempty"`
	PIN         string `json:"pin,omitempty"`
	Preview     bool   `json:"preview,omitempty"`
}

// OnlineBankTransferResult is returned after a transfer attempt.
type OnlineBankTransferResult struct {
	Status      string                 `json:"status"`
	Preview     bool                   `json:"preview,omitempty"`
	Transaction *OnlineBankTransaction `json:"transaction,omitempty"`
	FromBalance string                 `json:"fromBalance,omitempty"`
	ToBalance   string                 `json:"toBalance,omitempty"`
}

// OnlineBankDepositRequest credits an online bank account.
type OnlineBankDepositRequest struct {
	ToAccount string `json:"toAccount"`
	Amount    string `json:"amount"`
	Source    string `json:"source,omitempty"` // wire, sepa, swift, cash, ach
	Reference string `json:"reference,omitempty"`
	Preview   bool   `json:"preview,omitempty"`
}

// OnlineBankDepositResult is returned after a deposit.
type OnlineBankDepositResult struct {
	Status      string                 `json:"status"`
	Preview     bool                   `json:"preview,omitempty"`
	Transaction *OnlineBankTransaction `json:"transaction,omitempty"`
	ToBalance   string                 `json:"toBalance,omitempty"`
}

// OnlineBankLedgerSnapshot is the bank ledger view for UI and API.
type OnlineBankLedgerSnapshot struct {
	Name         string                  `json:"name"`
	SWIFT        string                  `json:"swift"`
	TotalUSD     float64                 `json:"totalUsd,omitempty"`
	Entries      []Entry                 `json:"entries"`
	Transactions []OnlineBankTransaction `json:"transactions"`
	ByFundClass  map[string]float64      `json:"byFundUsd,omitempty"`
	At           int64                   `json:"at"`
}

// OnlineBankStore persists live bank accounts and transactions.
type OnlineBankStore struct {
	mu   sync.Mutex
	path string
}

func OnlineBankEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(legacy.EnvOrLegacy("ONEX_ONLINE_BANK", "SHIVA_ONLINE_BANK")))
	if v == "0" || v == "false" || v == "off" {
		return false
	}
	if v == "1" || v == "true" || v == "on" {
		return true
	}
	cfg := LoadConfig()
	return cfg.Production()
}

func DefaultOnlineBankStore() *OnlineBankStore {
	return &OnlineBankStore{path: filepath.Join(legacy.HomeDir(), "online-bank.json")}
}

func (s *OnlineBankStore) load() (*OnlineBankState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &OnlineBankState{
				Name: defaultOnlineBankName, Online: true, SWIFT: defaultOnlineBankSWIFT,
				Accounts: []OnlineBankAccount{}, Transactions: []OnlineBankTransaction{},
			}, nil
		}
		return nil, err
	}
	var st OnlineBankState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	if st.Name == "" {
		st.Name = defaultOnlineBankName
	}
	if st.SWIFT == "" {
		st.SWIFT = defaultOnlineBankSWIFT
	}
	return &st, nil
}

func (s *OnlineBankStore) save(st *OnlineBankState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	st.UpdatedAt = time.Now().Unix()
	raw, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0o600)
}

// EnsureSeeded imports accounts from the bank ledger file when the store is empty.
func (s *OnlineBankStore) EnsureSeeded(bankFile string) error {
	st, err := s.load()
	if err != nil {
		return err
	}
	if len(st.Accounts) > 0 {
		return nil
	}
	path := strings.TrimSpace(bankFile)
	if path == "" {
		path = LoadBankProviderConfig().FilePath
	}
	if path == "" {
		path = LoadConfig().BankFile
	}
	if path == "" {
		return nil
	}
	seen := map[string]bool{}
	if accts, err := readBankAccountsFile(path); err == nil && len(accts) > 0 {
		for i, acct := range accts {
			id := strings.TrimSpace(acct.ID)
			if id == "" {
				id = fmt.Sprintf("bank-%d", i)
			}
			if seen[id] {
				continue
			}
			seen[id] = true
			cur := strings.ToUpper(strings.TrimSpace(acct.Currency))
			if cur == "" {
				cur = "USD"
			}
			bal := strings.TrimSpace(acct.Balance)
			if bal == "" {
				continue
			}
			name := strings.TrimSpace(acct.Name)
			if name == "" {
				name = id
			}
			pinHash := strings.TrimSpace(acct.OfficerPINHash)
			if pinHash == "" {
				pinHash = hashOfficerPIN(acct.OfficerPIN)
			}
			st.Accounts = append(st.Accounts, OnlineBankAccount{
				ID: id, Name: name, IBAN: normalizeIBAN(acct.IBAN),
				Currency: cur, Balance: bal, FundClass: resolveBankFundClass(acct, id),
				Bank: strings.TrimSpace(acct.Bank), Status: "active",
				OfficerPINHash: pinHash, OfficerPINRequired: acct.OfficerPINRequired || pinHash != "",
			})
		}
		st.Online = true
		return s.save(st)
	}
	entries, err := ReadBankLedger(BankConfig{FilePath: path})
	if err != nil {
		return err
	}
	for _, e := range entries {
		id := e.ID
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		iban := extractIBAN(e.Account)
		st.Accounts = append(st.Accounts, OnlineBankAccount{
			ID: id, Name: accountNameOnly(e.Account), IBAN: iban,
			Currency: e.Asset, Balance: e.Human, FundClass: e.FundClass,
			Bank: "nsb", Status: "active",
		})
	}
	st.Online = true
	return s.save(st)
}

func extractIBAN(account string) string {
	parts := strings.Split(account, "·")
	for _, p := range parts {
		p = normalizeIBAN(p)
		if len(p) >= 15 && ibanRe.MatchString(p) {
			return p
		}
	}
	return ""
}

func normalizeIBAN(s string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(s), " ", ""))
}

func accountNameOnly(account string) string {
	if idx := strings.Index(account, "·"); idx > 0 {
		return strings.TrimSpace(account[:idx])
	}
	return strings.TrimSpace(account)
}

func sanitizeOnlineBankAccount(acct OnlineBankAccount) OnlineBankAccount {
	acct.OfficerPINConfigured = strings.TrimSpace(acct.OfficerPINHash) != ""
	acct.OfficerPINHash = ""
	return acct
}

func hashOfficerPIN(pin string) string {
	pin = strings.TrimSpace(pin)
	if pin == "" {
		return ""
	}
	sum := sha256.Sum256([]byte("officer-pin:" + pin))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func verifyOfficerPIN(supplied, stored string) bool {
	supplied = strings.TrimSpace(supplied)
	stored = strings.TrimSpace(stored)
	if supplied == "" || stored == "" {
		return false
	}
	if strings.HasPrefix(stored, "sha256:") {
		return stored == hashOfficerPIN(supplied)
	}
	return stored == supplied
}

func requestedOfficerPIN(req OnlineBankTransferRequest) string {
	return firstNonEmpty(req.OfficerPIN, req.PIN)
}

func validateOfficerPIN(acct OnlineBankAccount, req OnlineBankTransferRequest) (bool, error) {
	if !acct.OfficerPINRequired && strings.TrimSpace(acct.OfficerPINHash) == "" {
		return false, nil
	}
	if strings.TrimSpace(acct.OfficerPINHash) == "" {
		return false, fmt.Errorf("officer PIN is not configured for %s", acct.Name)
	}
	if requestedOfficerPIN(req) == "" {
		return false, fmt.Errorf("officer PIN required for %s", acct.Name)
	}
	if !verifyOfficerPIN(requestedOfficerPIN(req), acct.OfficerPINHash) {
		return false, fmt.Errorf("invalid officer PIN for %s", acct.Name)
	}
	return true, nil
}

func (s *OnlineBankStore) Status() map[string]interface{} {
	st, err := s.load()
	if err != nil {
		return map[string]interface{}{"online": false, "error": err.Error()}
	}
	total := map[string]float64{}
	for _, a := range st.Accounts {
		bal, _ := strconv.ParseFloat(strings.TrimSpace(a.Balance), 64)
		total[strings.ToUpper(a.Currency)] += bal
	}
	return map[string]interface{}{
		"online":       st.Online && OnlineBankEnabled(),
		"name":         st.Name,
		"swift":        st.SWIFT,
		"accounts":     len(st.Accounts),
		"transactions": len(st.Transactions),
		"totals":       total,
		"updatedAt":    st.UpdatedAt,
	}
}

func (s *OnlineBankStore) ListAccounts() ([]OnlineBankAccount, error) {
	st, err := s.load()
	if err != nil {
		return nil, err
	}
	out := make([]OnlineBankAccount, len(st.Accounts))
	for i, acct := range st.Accounts {
		out[i] = sanitizeOnlineBankAccount(acct)
	}
	return out, nil
}

func (s *OnlineBankStore) ListTransactions(limit int) ([]OnlineBankTransaction, error) {
	return s.ListTransactionsFiltered(limit, "", "")
}

// ListTransactionsFiltered returns recent transactions, optionally filtered by account or type.
func (s *OnlineBankStore) ListTransactionsFiltered(limit int, accountID, txType string) ([]OnlineBankTransaction, error) {
	st, err := s.load()
	if err != nil {
		return nil, err
	}
	accountID = strings.TrimSpace(accountID)
	txType = strings.ToLower(strings.TrimSpace(txType))
	var filtered []OnlineBankTransaction
	for i := len(st.Transactions) - 1; i >= 0; i-- {
		tx := st.Transactions[i]
		if accountID != "" && tx.FromAccount != accountID && tx.ToAccount != accountID {
			continue
		}
		if txType != "" && !strings.EqualFold(tx.Type, txType) {
			continue
		}
		filtered = append(filtered, tx)
		if limit > 0 && len(filtered) >= limit {
			break
		}
	}
	return filtered, nil
}

func (s *OnlineBankStore) GetAccount(id string) (*OnlineBankAccount, error) {
	st, err := s.load()
	if err != nil {
		return nil, err
	}
	acct, _ := s.findAccount(st, strings.TrimSpace(id))
	if acct == nil {
		return nil, fmt.Errorf("account not found")
	}
	out := sanitizeOnlineBankAccount(*acct)
	return &out, nil
}

// OnlineBankWireInstructions is how to receive funds into an account.
type OnlineBankWireInstructions struct {
	AccountID   string `json:"accountId"`
	AccountName string `json:"accountName"`
	IBAN        string `json:"iban,omitempty"`
	SWIFT       string `json:"swift"`
	BankName    string `json:"bankName"`
	Currency    string `json:"currency"`
	Reference   string `json:"reference,omitempty"`
}

func (s *OnlineBankStore) WireInstructions(accountID string) (*OnlineBankWireInstructions, error) {
	st, err := s.load()
	if err != nil {
		return nil, err
	}
	acct, _ := s.findAccount(st, strings.TrimSpace(accountID))
	if acct == nil {
		return nil, fmt.Errorf("account not found")
	}
	ref := "NSB-" + strings.ToUpper(acct.ID)
	return &OnlineBankWireInstructions{
		AccountID: acct.ID, AccountName: acct.Name, IBAN: acct.IBAN,
		SWIFT: st.SWIFT, BankName: st.Name, Currency: acct.Currency, Reference: ref,
	}, nil
}

// ExportTransactionsCSV returns a CSV statement for one account or all accounts.
func (s *OnlineBankStore) ExportTransactionsCSV(accountID string) (string, error) {
	txs, err := s.ListTransactionsFiltered(0, accountID, "")
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("date,type,status,amount,currency,from,to,reference,settlement\n")
	for _, t := range txs {
		to := t.ToName
		if to == "" {
			to = t.ToIBAN
		}
		if to == "" {
			to = t.ToAccount
		}
		fmt.Fprintf(&b, "%s,%s,%s,%s,%s,%s,%s,%s,%s\n",
			time.Unix(t.CreatedAt, 0).UTC().Format("2006-01-02 15:04:05"),
			t.Type, t.Status, t.Amount, t.Currency,
			escapeCSV(t.FromName), escapeCSV(to),
			escapeCSV(t.Reference), escapeCSV(t.Settlement),
		)
	}
	return b.String(), nil
}

func escapeCSV(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

func (s *OnlineBankStore) findAccount(st *OnlineBankState, id string) (*OnlineBankAccount, int) {
	for i := range st.Accounts {
		if st.Accounts[i].ID == id {
			return &st.Accounts[i], i
		}
	}
	return nil, -1
}

func (s *OnlineBankStore) findAccountByIBAN(st *OnlineBankState, iban string) *OnlineBankAccount {
	iban = normalizeIBAN(iban)
	if iban == "" {
		return nil
	}
	for i := range st.Accounts {
		if normalizeIBAN(st.Accounts[i].IBAN) == iban {
			return &st.Accounts[i]
		}
	}
	return nil
}

// EnsureSystemAccount creates a zero-balance system account when missing.
func (s *OnlineBankStore) EnsureSystemAccount(id, name, currency, fundClass string) error {
	st, err := s.load()
	if err != nil {
		return err
	}
	for _, a := range st.Accounts {
		if a.ID == id {
			return nil
		}
	}
	st.Accounts = append(st.Accounts, OnlineBankAccount{
		ID: id, Name: name, Currency: strings.ToUpper(currency),
		Balance: "0.00", FundClass: fundClass, Bank: "nsb", Status: "active",
	})
	return s.save(st)
}

// Transfer moves funds between online bank accounts or to an external IBAN.
func (s *OnlineBankStore) Transfer(req OnlineBankTransferRequest) (*OnlineBankTransferResult, error) {
	st, err := s.load()
	if err != nil {
		return nil, err
	}
	from, fi := s.findAccount(st, strings.TrimSpace(req.FromAccount))
	if from == nil {
		return nil, fmt.Errorf("from account not found")
	}
	amt, err := strconv.ParseFloat(strings.TrimSpace(req.Amount), 64)
	if err != nil || amt <= 0 {
		return nil, fmt.Errorf("invalid amount")
	}
	fromBal, _ := strconv.ParseFloat(strings.TrimSpace(from.Balance), 64)
	if fromBal < amt {
		return nil, fmt.Errorf("insufficient balance")
	}

	txType := "internal"
	status := "completed"
	settlement := ""
	toName := ""
	toIBAN := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(req.ToIBAN), " ", ""))
	toBank := strings.TrimSpace(req.ToBank)
	var to *OnlineBankAccount
	rail := BankRail(strings.ToLower(strings.TrimSpace(req.Rail)))
	if toIBAN != "" {
		txType = "send"
		if rail == "" {
			rail = RailIBAN
		}
		if _, err := ParseExternalDestination(fmt.Sprintf("bank:%s:%s:%s", orDefault(toBank, "generic"), rail, toIBAN)); err != nil {
			return nil, err
		}
		to = s.findAccountByIBAN(st, toIBAN)
		status = "pending"
	} else if strings.TrimSpace(req.ToAccount) != "" {
		to, _ = s.findAccount(st, strings.TrimSpace(req.ToAccount))
		if to == nil {
			return nil, fmt.Errorf("to account not found")
		}
		if !strings.EqualFold(to.Currency, from.Currency) {
			return nil, fmt.Errorf("currency mismatch: %s → %s", from.Currency, to.Currency)
		}
		toName = to.Name
	} else {
		return nil, fmt.Errorf("toAccount or toIban required")
	}
	officerAuthorized := false
	for _, acct := range []*OnlineBankAccount{from, to} {
		if acct == nil {
			continue
		}
		ok, err := validateOfficerPIN(*acct, req)
		if err != nil {
			return nil, err
		}
		officerAuthorized = officerAuthorized || ok
	}

	ref := strings.TrimSpace(req.Reference)
	if ref == "" {
		ref = fmt.Sprintf("NSB-%d", time.Now().Unix())
	}

	preview := req.Preview
	res := &OnlineBankTransferResult{
		Status:  "quoted",
		Preview: preview,
		Transaction: &OnlineBankTransaction{
			Type: txType, FromAccount: from.ID, FromName: from.Name,
			ToAccount: req.ToAccount, ToName: toName, ToIBAN: toIBAN, ToBank: toBank,
			Rail: string(rail), Amount: formatFloat(amt), Currency: from.Currency,
			Status: status, Reference: ref, OfficerAuthorized: officerAuthorized,
		},
	}

	if preview {
		res.FromBalance = from.Balance
		return res, nil
	}

	from.Balance = formatFloat(fromBal - amt)
	st.Accounts[fi] = *from

	if txType == "internal" {
		to, ti := s.findAccount(st, strings.TrimSpace(req.ToAccount))
		toBal, _ := strconv.ParseFloat(strings.TrimSpace(to.Balance), 64)
		to.Balance = formatFloat(toBal + amt)
		st.Accounts[ti] = *to
		res.ToBalance = to.Balance
		res.Transaction.Status = "completed"
	} else {
		settlement, _ = InitiateBankTransfer(BankTransferRequest{
			Rail: rail, BankName: toBank, Account: toIBAN,
			Amount: formatFloat(amt), Asset: from.Currency, Reference: ref,
		})
		res.Transaction.Settlement = settlement
		if strings.HasPrefix(settlement, "plaid-") || strings.HasPrefix(settlement, "truelayer-") ||
			strings.HasPrefix(settlement, "hybx-") {
			res.Transaction.Status = "completed"
		}
	}

	res.FromBalance = from.Balance
	res.Transaction.CreatedAt = time.Now().Unix()
	res.Transaction.ID = fmt.Sprintf("obtx-%d", time.Now().UnixNano())
	st.Transactions = append(st.Transactions, *res.Transaction)
	if err := s.save(st); err != nil {
		return nil, err
	}
	res.Status = res.Transaction.Status
	res.Preview = false
	return res, nil
}

func orDefault(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return strings.TrimSpace(v)
}

// LedgerEntries converts live online bank accounts to ledger entries.
func (s *OnlineBankStore) LedgerEntries() ([]Entry, error) {
	st, err := s.load()
	if err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	out := make([]Entry, 0, len(st.Accounts))
	for _, acct := range st.Accounts {
		bal := strings.TrimSpace(acct.Balance)
		if bal == "" {
			continue
		}
		account := acct.Name
		if acct.IBAN != "" {
			if account != "" {
				account += " · "
			}
			account += acct.IBAN
		}
		fc := resolveBankFundClass(BankAccount{
			ID: acct.ID, FundClass: acct.FundClass, Bank: acct.Bank, Currency: acct.Currency,
		}, acct.ID)
		out = append(out, Entry{
			ID: acct.ID, Source: SourceBank, Mode: ModeBank,
			Asset: strings.ToUpper(acct.Currency), TokenKey: "fiat:" + strings.ToUpper(acct.Currency),
			Human: bal, FiatCurrency: strings.ToUpper(acct.Currency),
			FundClass: fc, Account: account, Timestamp: now, Reference: "online-bank",
		})
	}
	return out, nil
}

// ReadOnlineBankLedger loads accounts when online bank mode is active.
func ReadOnlineBankLedger(bankFile string) ([]Entry, error) {
	if !OnlineBankEnabled() {
		return nil, nil
	}
	store := DefaultOnlineBankStore()
	if err := store.EnsureSeeded(bankFile); err != nil {
		return nil, err
	}
	return store.LedgerEntries()
}

// Deposit credits an online bank account and records the deposit in the bank ledger.
func (s *OnlineBankStore) Deposit(req OnlineBankDepositRequest) (*OnlineBankDepositResult, error) {
	st, err := s.load()
	if err != nil {
		return nil, err
	}
	to, ti := s.findAccount(st, strings.TrimSpace(req.ToAccount))
	if to == nil {
		return nil, fmt.Errorf("account not found")
	}
	amt, err := strconv.ParseFloat(strings.TrimSpace(req.Amount), 64)
	if err != nil || amt <= 0 {
		return nil, fmt.Errorf("invalid amount")
	}
	src := strings.TrimSpace(req.Source)
	if src == "" {
		src = "wire"
	}
	ref := strings.TrimSpace(req.Reference)
	if ref == "" {
		ref = fmt.Sprintf("DEP-%d", time.Now().Unix())
	}
	toBal, _ := strconv.ParseFloat(strings.TrimSpace(to.Balance), 64)
	res := &OnlineBankDepositResult{
		Status:  "quoted",
		Preview: req.Preview,
		Transaction: &OnlineBankTransaction{
			Type: "deposit", ToAccount: to.ID, ToName: to.Name,
			Amount: formatFloat(amt), Currency: to.Currency,
			Status: "completed", Reference: ref, Rail: src,
		},
	}
	if req.Preview {
		res.ToBalance = to.Balance
		return res, nil
	}
	to.Balance = formatFloat(toBal + amt)
	st.Accounts[ti] = *to
	res.Transaction.FromName = src + " deposit"
	res.Transaction.CreatedAt = time.Now().Unix()
	res.Transaction.ID = fmt.Sprintf("obdep-%d", time.Now().UnixNano())
	st.Transactions = append(st.Transactions, *res.Transaction)
	if err := s.save(st); err != nil {
		return nil, err
	}
	res.Status = "completed"
	res.ToBalance = to.Balance
	res.Preview = false
	return res, nil
}

// Send is an alias for outbound transfer (internal or IBAN).
func (s *OnlineBankStore) Send(req OnlineBankTransferRequest) (*OnlineBankTransferResult, error) {
	return s.Transfer(req)
}

// BankLedger returns accounts as ledger entries plus transaction history.
func (s *OnlineBankStore) BankLedger(prices map[string]PriceQuote, limit int) (*OnlineBankLedgerSnapshot, error) {
	st, err := s.load()
	if err != nil {
		return nil, err
	}
	entries, err := s.LedgerEntries()
	if err != nil {
		return nil, err
	}
	txs, err := s.ListTransactions(limit)
	if err != nil {
		return nil, err
	}
	byFund := map[string]float64{}
	var totalUSD float64
	valued := make([]Entry, len(entries))
	copy(valued, entries)
	for i := range valued {
		amt := parseFloatSafe(valued[i].Human)
		usd := amt * unitUSD(valued[i].Asset, prices, "USD")
		valued[i].FiatUSD = usd
		totalUSD += usd
		if valued[i].FundClass != "" {
			byFund[valued[i].FundClass] += usd
		}
	}
	return &OnlineBankLedgerSnapshot{
		Name: st.Name, SWIFT: st.SWIFT, Entries: valued,
		Transactions: txs, ByFundClass: byFund, TotalUSD: totalUSD,
		At: time.Now().Unix(),
	}, nil
}

// DebitAccount removes funds from an online bank account (card spend, fees).
func (s *OnlineBankStore) DebitAccount(accountID, amount, ref string) (string, error) {
	st, err := s.load()
	if err != nil {
		return "", err
	}
	acct, idx := s.findAccount(st, strings.TrimSpace(accountID))
	if acct == nil {
		return "", fmt.Errorf("account not found")
	}
	amt, err := strconv.ParseFloat(strings.TrimSpace(amount), 64)
	if err != nil || amt <= 0 {
		return "", fmt.Errorf("invalid amount")
	}
	bal, _ := strconv.ParseFloat(strings.TrimSpace(acct.Balance), 64)
	if bal < amt {
		return "", fmt.Errorf("insufficient balance")
	}
	acct.Balance = formatFloat(bal - amt)
	st.Accounts[idx] = *acct
	tx := OnlineBankTransaction{
		ID:   fmt.Sprintf("obdebit-%d", time.Now().UnixNano()),
		Type: "send", FromAccount: acct.ID, FromName: acct.Name,
		Amount: formatFloat(amt), Currency: acct.Currency,
		Status: "completed", Reference: ref, CreatedAt: time.Now().Unix(),
	}
	st.Transactions = append(st.Transactions, tx)
	if err := s.save(st); err != nil {
		return "", err
	}
	return acct.Balance, nil
}

func parseFloatSafe(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}
