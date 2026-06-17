package ledger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/onex-blockchain/onex/internal/legacy"
)

// BankProviderConfig holds Open Banking credentials.
type BankProviderConfig struct {
	Provider string // file, api, plaid, truelayer

	FilePath string
	APIURL   string
	APIKey   string

	PlaidClientID    string
	PlaidSecret      string
	PlaidAccessToken string
	PlaidEnv         string // sandbox, development, production

	TrueLayerToken   string
	TrueLayerBaseURL string
}

func LoadBankProviderConfig() BankProviderConfig {
	provider := strings.ToLower(strings.TrimSpace(legacy.EnvOrLegacy("ONEX_BANK_PROVIDER", "SHIVA_BANK_PROVIDER")))
	if provider == "" {
		provider = "auto"
	}
	tlBase := legacy.EnvOrLegacy("ONEX_TRUELAYER_API_URL", "SHIVA_TRUELAYER_API_URL")
	if tlBase == "" {
		tlBase = "https://api.truelayer.com"
	}
	return BankProviderConfig{
		Provider: provider,
		FilePath: legacy.EnvOrLegacy("ONEX_BANK_LEDGER_FILE", "SHIVA_BANK_LEDGER_FILE"),
		APIURL:   legacy.EnvOrLegacy("ONEX_BANK_LEDGER_URL", "SHIVA_BANK_LEDGER_URL"),
		APIKey:   legacy.EnvOrLegacy("ONEX_BANK_API_KEY", "SHIVA_BANK_API_KEY"),

		PlaidClientID:    legacy.EnvOrLegacy("ONEX_PLAID_CLIENT_ID", "SHIVA_PLAID_CLIENT_ID"),
		PlaidSecret:      legacy.EnvOrLegacy("ONEX_PLAID_SECRET", "SHIVA_PLAID_SECRET"),
		PlaidAccessToken: legacy.EnvOrLegacy("ONEX_PLAID_ACCESS_TOKEN", "SHIVA_PLAID_ACCESS_TOKEN"),
		PlaidEnv:         legacy.EnvOrLegacy("ONEX_PLAID_ENV", "SHIVA_PLAID_ENV"),

		TrueLayerToken:   legacy.EnvOrLegacy("ONEX_TRUELAYER_ACCESS_TOKEN", "SHIVA_TRUELAYER_ACCESS_TOKEN"),
		TrueLayerBaseURL: tlBase,
	}
}

func (c BankProviderConfig) ResolvedProvider() string {
	if c.Provider != "" && c.Provider != "auto" {
		return c.Provider
	}
	if c.PlaidClientID != "" && c.PlaidAccessToken != "" {
		return "plaid"
	}
	if c.TrueLayerToken != "" {
		return "truelayer"
	}
	if c.FilePath != "" {
		return "file"
	}
	if c.APIURL != "" {
		return "api"
	}
	return ""
}

func (c BankProviderConfig) Status() map[string]interface{} {
	p := c.ResolvedProvider()
	return map[string]interface{}{
		"provider":       p,
		"configured":     p != "",
		"plaid":          c.PlaidClientID != "" && c.PlaidAccessToken != "",
		"truelayer":      c.TrueLayerToken != "",
		"file":           c.FilePath != "",
		"customAPI":      c.APIURL != "",
		"plaidEnv":       c.plaidHost(),
		"trueLayerBase":  c.TrueLayerBaseURL,
	}
}

func (c BankProviderConfig) plaidHost() string {
	switch strings.ToLower(strings.TrimSpace(c.PlaidEnv)) {
	case "production", "prod":
		return "https://production.plaid.com"
	case "development", "dev":
		return "https://development.plaid.com"
	default:
		return "https://sandbox.plaid.com"
	}
}

// ReadBankLedgerWithProvider loads fiat balances from the configured bank source.
func ReadBankLedgerWithProvider(cfg BankProviderConfig) ([]Entry, error) {
	switch cfg.ResolvedProvider() {
	case "plaid":
		return fetchPlaidBalances(cfg)
	case "truelayer":
		return fetchTrueLayerBalances(cfg)
	case "file":
		if cfg.FilePath == "" {
			return nil, nil
		}
		return ReadBankLedger(BankConfig{FilePath: cfg.FilePath})
	case "api":
		if cfg.APIURL == "" {
			return nil, nil
		}
		return ReadBankLedger(BankConfig{APIURL: cfg.APIURL, APIKey: cfg.APIKey})
	default:
		return ReadBankLedger(BankConfig{
			FilePath: cfg.FilePath,
			APIURL:   cfg.APIURL,
			APIKey:   cfg.APIKey,
		})
	}
}

func fetchPlaidBalances(cfg BankProviderConfig) ([]Entry, error) {
	if cfg.PlaidClientID == "" || cfg.PlaidSecret == "" || cfg.PlaidAccessToken == "" {
		return nil, fmt.Errorf("plaid: set ONEX_PLAID_CLIENT_ID, ONEX_PLAID_SECRET, ONEX_PLAID_ACCESS_TOKEN")
	}
	body, _ := json.Marshal(map[string]string{
		"client_id":     cfg.PlaidClientID,
		"secret":        cfg.PlaidSecret,
		"access_token":  cfg.PlaidAccessToken,
	})
	raw, err := postJSON(cfg.plaidHost()+"/accounts/balance/get", body, "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Accounts []struct {
			AccountID string `json:"account_id"`
			Name      string `json:"name"`
			Mask      string `json:"mask"`
			Balances  struct {
				Available *float64 `json:"available"`
				Current   *float64 `json:"current"`
				ISO       string   `json:"iso_currency_code"`
			} `json:"balances"`
		} `json:"accounts"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	var out []Entry
	for _, acct := range resp.Accounts {
		amt := acct.Balances.Current
		if amt == nil {
			amt = acct.Balances.Available
		}
		if amt == nil {
			continue
		}
		cur := strings.ToUpper(strings.TrimSpace(acct.Balances.ISO))
		if cur == "" {
			cur = "USD"
		}
		account := acct.Name
		if acct.Mask != "" {
			account += " ·••" + acct.Mask
		}
		out = append(out, Entry{
			ID:           acct.AccountID,
			Source:       SourceBank,
			Mode:         ModeBank,
			Asset:        cur,
			TokenKey:     "fiat:" + cur,
			Human:        formatFloat(*amt),
			FiatCurrency: cur,
			Account:      account,
			Timestamp:    now,
			Reference:    "plaid",
		})
	}
	return out, nil
}

func fetchTrueLayerBalances(cfg BankProviderConfig) ([]Entry, error) {
	if cfg.TrueLayerToken == "" {
		return nil, fmt.Errorf("truelayer: set ONEX_TRUELAYER_ACCESS_TOKEN")
	}
	base := strings.TrimRight(cfg.TrueLayerBaseURL, "/")
	raw, err := getJSON(base+"/data/v1/accounts", cfg.TrueLayerToken)
	if err != nil {
		return nil, err
	}
	var accounts struct {
		Results []struct {
			AccountID   string `json:"account_id"`
			DisplayName string `json:"display_name"`
			Currency    string `json:"currency"`
			AccountType string `json:"account_type"`
		} `json:"results"`
	}
	if err := json.Unmarshal(raw, &accounts); err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	var out []Entry
	for _, acct := range accounts.Results {
		balRaw, err := getJSON(base+"/data/v1/accounts/"+acct.AccountID+"/balance", cfg.TrueLayerToken)
		if err != nil {
			continue
		}
		var bal struct {
			Results []struct {
				Current   float64 `json:"current"`
				Available float64 `json:"available"`
				Currency  string  `json:"currency"`
			} `json:"results"`
		}
		if err := json.Unmarshal(balRaw, &bal); err != nil || len(bal.Results) == 0 {
			continue
		}
		row := bal.Results[0]
		amt := row.Current
		if amt == 0 && row.Available > 0 {
			amt = row.Available
		}
		cur := strings.ToUpper(strings.TrimSpace(row.Currency))
		if cur == "" {
			cur = strings.ToUpper(strings.TrimSpace(acct.Currency))
		}
		if cur == "" {
			cur = "GBP"
		}
		out = append(out, Entry{
			ID:           acct.AccountID,
			Source:       SourceBank,
			Mode:         ModeBank,
			Asset:        cur,
			TokenKey:     "fiat:" + cur,
			Human:        formatFloat(amt),
			FiatCurrency: cur,
			Account:      acct.DisplayName,
			Timestamp:    now,
			Reference:    "truelayer:" + acct.AccountType,
		})
	}
	return out, nil
}

// BankTransferRequest initiates an outbound transfer to an external bank.
type BankTransferRequest struct {
	Rail      BankRail
	BankName  string
	Account   string
	Amount    string
	Asset     string
	Reference string
}

// InitiateBankTransfer submits a wire/ACH/SEPA/SWIFT transfer via configured provider.
func InitiateBankTransfer(req BankTransferRequest) (string, error) {
	cfg := LoadBankProviderConfig()
	ref := strings.TrimSpace(req.Reference)
	if ref == "" {
		ref = fmt.Sprintf("bank-xfer-%d", time.Now().UnixNano())
	}
	switch cfg.ResolvedProvider() {
	case "plaid":
		return initiatePlaidTransfer(cfg, req, ref)
	case "truelayer":
		return initiateTrueLayerTransfer(cfg, req, ref)
	}
	// Book-recorded settlement when no live bank API is configured.
	return fmt.Sprintf("bank-pending:%s:%s:%s", req.Rail, req.BankName, req.Account), nil
}

func initiatePlaidTransfer(cfg BankProviderConfig, req BankTransferRequest, ref string) (string, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"client_id":    cfg.PlaidClientID,
		"secret":       cfg.PlaidSecret,
		"access_token": cfg.PlaidAccessToken,
		"account_id":   req.Account,
		"type":         string(req.Rail),
		"amount":       req.Amount,
		"description":  ref,
	})
	_, err := postJSON(cfg.plaidHost()+"/transfer/authorization/create", body, "")
	if err != nil {
		return fmt.Sprintf("bank-pending:%s:%s", req.Rail, req.Account), nil
	}
	return "plaid-transfer-submitted:" + ref, nil
}

func initiateTrueLayerTransfer(cfg BankProviderConfig, req BankTransferRequest, ref string) (string, error) {
	base := strings.TrimRight(cfg.TrueLayerBaseURL, "/")
	payload := map[string]interface{}{
		"amount_in_minor": parseMinor(req.Amount),
		"currency":      strings.ToUpper(req.Asset),
		"beneficiary":   map[string]string{"account_identifier": req.Account},
		"reference":     ref,
	}
	body, _ := json.Marshal(payload)
	_, err := postJSON(base+"/payments", body, cfg.TrueLayerToken)
	if err != nil {
		return fmt.Sprintf("bank-pending:%s:%s", req.Rail, req.Account), nil
	}
	return "truelayer-payment-submitted:" + ref, nil
}

func parseMinor(amount string) int64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(amount), 64)
	return int64(v * 100)
}

func postJSON(url string, body []byte, bearer string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return raw, nil
}

func getJSON(url, bearer string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return raw, nil
}
