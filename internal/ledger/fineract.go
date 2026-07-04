package ledger

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/onex-blockchain/onex/internal/legacy"
)

const (
	defaultFineractURL     = "https://fineract.hybxfinance.com/fineract-provider"
	defaultFineractTenant  = "default"
	fineractSwaggerPath    = "/swagger-ui/index.html"
	fineractAccountPrefix  = "fineract-"
)

// FineractConfig holds Apache Fineract / Mifos core banking credentials.
type FineractConfig struct {
	Enabled  bool
	BaseURL  string
	Tenant   string
	Username string
	Password string
}

func LoadFineractConfig() FineractConfig {
	base := strings.TrimRight(strings.TrimSpace(legacy.EnvOrLegacy("ONEX_FINERACT_URL", "ONEX_FINERACT_BASE_URL")), "/")
	if base == "" {
		base = defaultFineractURL
	}
	tenant := strings.TrimSpace(legacy.EnvOrLegacy("ONEX_FINERACT_TENANT", "ONEX_FINERACT_TENANT_ID"))
	if tenant == "" {
		tenant = defaultFineractTenant
	}
	v := strings.ToLower(strings.TrimSpace(legacy.EnvOrLegacy("ONEX_FINERACT_ENABLED", "")))
	enabled := OnlineBankEnabled()
	if v == "0" || v == "false" || v == "off" {
		enabled = false
	}
	if v == "1" || v == "true" || v == "on" {
		enabled = true
	}
	return FineractConfig{
		Enabled:  enabled,
		BaseURL:  base,
		Tenant:   tenant,
		Username: legacy.EnvOrLegacy("ONEX_FINERACT_USERNAME", "ONEX_FINERACT_USER"),
		Password: legacy.EnvOrLegacy("ONEX_FINERACT_PASSWORD", "ONEX_FINERACT_PASS"),
	}
}

func (c FineractConfig) Configured() bool {
	return c.BaseURL != "" && c.Username != "" && c.Password != ""
}

func (c FineractConfig) SwaggerURL() string {
	return strings.TrimRight(c.BaseURL, "/") + fineractSwaggerPath
}

func (c FineractConfig) APIRoot() string {
	return strings.TrimRight(c.BaseURL, "/") + "/api/v1"
}

// FineractSavingsAccount is a normalized savings account from Fineract.
type FineractSavingsAccount struct {
	ID               int64   `json:"id"`
	AccountNo        string  `json:"accountNo"`
	ClientID         int64   `json:"clientId"`
	ClientName       string  `json:"clientName"`
	ProductName      string  `json:"productName"`
	Currency         string  `json:"currency"`
	Status           string  `json:"status"`
	AccountBalance   float64 `json:"accountBalance"`
	AvailableBalance float64 `json:"availableBalance"`
}

// FineractClient wraps the Apache Fineract REST API.
type FineractClient struct {
	cfg    FineractConfig
	client *http.Client
}

func NewFineractClient() *FineractClient {
	return &FineractClient{
		cfg:    LoadFineractConfig(),
		client: &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *FineractClient) Status() map[string]interface{} {
	st := map[string]interface{}{
		"enabled":    c.cfg.Enabled,
		"provider":   "Apache Fineract",
		"name":       "HYBX Fineract Core Banking",
		"baseUrl":    c.cfg.BaseURL,
		"swaggerUrl": c.cfg.SwaggerURL(),
		"tenant":     c.cfg.Tenant,
		"configured": c.cfg.Configured(),
		"online":     false,
		"accounts":   0,
	}
	if !c.cfg.Enabled {
		return st
	}
	if !c.cfg.Configured() {
		st["detail"] = "set ONEX_FINERACT_USERNAME and ONEX_FINERACT_PASSWORD"
		return st
	}
	accts, err := c.ListSavingsAccounts()
	if err != nil {
		st["error"] = err.Error()
		return st
	}
	st["online"] = true
	st["accounts"] = len(accts)
	return st
}

func (c *FineractClient) ListSavingsAccounts() ([]FineractSavingsAccount, error) {
	if !c.cfg.Configured() {
		return nil, fmt.Errorf("fineract not configured")
	}
	var page struct {
		PageItems []json.RawMessage `json:"pageItems"`
	}
	if err := c.getJSON("/savingsaccounts?limit=200", &page); err != nil {
		return nil, err
	}
	out := make([]FineractSavingsAccount, 0, len(page.PageItems))
	for _, raw := range page.PageItems {
		if acct, err := parseFineractSavings(raw); err == nil {
			out = append(out, acct)
		}
	}
	return out, nil
}

func (c *FineractClient) GetSavingsAccount(id int64) (*FineractSavingsAccount, error) {
	var raw json.RawMessage
	if err := c.getJSON(fmt.Sprintf("/savingsaccounts/%d", id), &raw); err != nil {
		return nil, err
	}
	acct, err := parseFineractSavings(raw)
	if err != nil {
		return nil, err
	}
	return &acct, nil
}

func (c *FineractClient) Deposit(accountID int64, amount float64, reference string) (map[string]interface{}, error) {
	return c.postTransaction(accountID, "deposit", amount, reference)
}

func (c *FineractClient) Withdraw(accountID int64, amount float64, reference string) (map[string]interface{}, error) {
	return c.postTransaction(accountID, "withdrawal", amount, reference)
}

func (c *FineractClient) postTransaction(accountID int64, command string, amount float64, reference string) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"transactionDate":   fineractDate(time.Now()),
		"transactionAmount": formatFloat(amount),
		"locale":            "en",
		"dateFormat":        "dd MMMM yyyy",
	}
	if reference != "" {
		body["note"] = reference
	}
	var out map[string]interface{}
	path := fmt.Sprintf("/savingsaccounts/%d/transactions?command=%s", accountID, command)
	if err := c.postJSON(path, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SyncFineractToOnlineBank imports Fineract savings accounts into the online bank store.
func SyncFineractToOnlineBank(bank *OnlineBankStore, client *FineractClient) ([]OnlineBankAccount, error) {
	if client == nil {
		client = NewFineractClient()
	}
	if !client.cfg.Enabled || !client.cfg.Configured() {
		return nil, fmt.Errorf("fineract not configured")
	}
	accts, err := client.ListSavingsAccounts()
	if err != nil {
		return nil, err
	}
	st, err := bank.load()
	if err != nil {
		return nil, err
	}
	byID := map[string]int{}
	for i, a := range st.Accounts {
		byID[a.ID] = i
	}
	synced := make([]OnlineBankAccount, 0, len(accts))
	for _, fa := range accts {
		id := FineractOnlineBankID(fa.ID)
		name := strings.TrimSpace(fa.ClientName)
		if name == "" {
			name = fa.ProductName
		}
		if name == "" {
			name = "Fineract Savings"
		}
		bal := formatFloat(fa.AccountBalance)
		if fa.AvailableBalance > 0 && fa.AvailableBalance != fa.AccountBalance {
			bal = formatFloat(fa.AvailableBalance)
		}
		acct := OnlineBankAccount{
			ID: id, Name: name + " · " + fa.ProductName,
			IBAN: fa.AccountNo, Currency: fa.Currency, Balance: bal,
			Bank: "fineract", Status: fa.Status,
		}
		if idx, ok := byID[id]; ok {
			st.Accounts[idx] = acct
		} else {
			st.Accounts = append(st.Accounts, acct)
		}
		synced = append(synced, acct)
	}
	if err := bank.save(st); err != nil {
		return nil, err
	}
	return synced, nil
}

func FineractOnlineBankID(fineractID int64) string {
	return fineractAccountPrefix + strconv.FormatInt(fineractID, 10)
}

func ParseFineractOnlineBankID(id string) (int64, bool) {
	id = strings.TrimSpace(id)
	if !strings.HasPrefix(id, fineractAccountPrefix) {
		return 0, false
	}
	n, err := strconv.ParseInt(strings.TrimPrefix(id, fineractAccountPrefix), 10, 64)
	return n, err == nil && n > 0
}

func IsFineractOnlineBankAccount(id string) bool {
	_, ok := ParseFineractOnlineBankID(id)
	return ok
}

func fineractDate(t time.Time) string {
	return t.Format("02 January 2006")
}

func parseFineractSavings(raw json.RawMessage) (FineractSavingsAccount, error) {
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return FineractSavingsAccount{}, err
	}
	acct := FineractSavingsAccount{
		ID:         jsonInt64(m["id"]),
		AccountNo:  jsonString(m["accountNo"]),
		ClientID:   jsonInt64(m["clientId"]),
		ClientName: jsonString(m["clientName"]),
		ProductName: jsonString(m["productName"]),
	}
	if cur, ok := m["currency"].(map[string]interface{}); ok {
		acct.Currency = jsonString(cur["code"])
	}
	if st, ok := m["status"].(map[string]interface{}); ok {
		acct.Status = jsonString(st["value"])
		if acct.Status == "" {
			acct.Status = jsonString(st["code"])
		}
	}
	if sum, ok := m["summary"].(map[string]interface{}); ok {
		acct.AccountBalance = jsonFloat(sum["accountBalance"])
		acct.AvailableBalance = jsonFloat(sum["availableBalance"])
	} else {
		acct.AccountBalance = jsonFloat(m["accountBalance"])
		acct.AvailableBalance = jsonFloat(m["availableBalance"])
	}
	if acct.Currency == "" {
		acct.Currency = "USD"
	}
	if acct.Status == "" {
		acct.Status = "active"
	}
	return acct, nil
}

func jsonString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatInt(int64(t), 10)
	default:
		return ""
	}
}

func jsonInt64(v interface{}) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int64:
		return t
	case json.Number:
		n, _ := t.Int64()
		return n
	default:
		return 0
	}
}

func jsonFloat(v interface{}) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case json.Number:
		f, _ := t.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(t, 64)
		return f
	default:
		return 0
	}
}

func (c *FineractClient) getJSON(path string, out interface{}) error {
	req, err := http.NewRequest(http.MethodGet, c.cfg.APIRoot()+path, nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *FineractClient) postJSON(path string, body interface{}, out interface{}) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, c.cfg.APIRoot()+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, out)
}

func (c *FineractClient) do(req *http.Request, out interface{}) error {
	req.Header.Set("Fineract-Platform-TenantId", c.cfg.Tenant)
	req.Header.Set("Accept", "application/json")
	token := base64.StdEncoding.EncodeToString([]byte(c.cfg.Username + ":" + c.cfg.Password))
	req.Header.Set("Authorization", "Basic "+token)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		msg := strings.TrimSpace(string(data))
		if len(msg) > 240 {
			msg = msg[:240] + "…"
		}
		return fmt.Errorf("fineract HTTP %d: %s", resp.StatusCode, msg)
	}
	if out == nil {
		return nil
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, out)
}
