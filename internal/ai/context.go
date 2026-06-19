package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

func summarizeContext(ctx string) string {
	if ctx == "" {
		return ""
	}
	var parts []string

	for _, line := range strings.Split(ctx, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, ": ")
		if idx < 0 {
			continue
		}
		key := line[:idx]
		raw := line[idx+2:]
		switch key {
		case "real_ledger":
			if s := summarizeRealLedger(raw); s != "" {
				parts = append(parts, s)
			}
		case "ledger_accounts":
			if s := summarizeLedgerAccounts(raw); s != "" {
				parts = append(parts, s)
			}
		case "saved_destinations":
			if s := summarizeSavedDestinations(raw); s != "" {
				parts = append(parts, s)
			}
		case "ledger_status":
			if s := summarizeLedgerStatus(raw); s != "" {
				parts = append(parts, s)
			}
		case "portfolio":
			if s := summarizePortfolio(raw); s != "" {
				parts = append(parts, s)
			}
		case "virtual_cards":
			if s := summarizeVirtualCards(raw); s != "" {
				parts = append(parts, s)
			}
		}
	}

	if len(parts) == 0 {
		if len(ctx) > 1200 {
			return ctx[:1200] + "…"
		}
		return ctx
	}
	return strings.Join(parts, "\n")
}

func summarizeRealLedger(raw string) string {
	var m struct {
		TotalUsd    float64            `json:"totalUsd"`
		Mode        string             `json:"mode"`
		BySourceUsd map[string]float64 `json:"bySourceUsd"`
		ByFundUsd   map[string]float64 `json:"byFundUsd"`
		EntryCount  int                `json:"entryCount"`
		Entries     []struct {
			Asset     string  `json:"asset"`
			Amount    string  `json:"amount"`
			Source    string  `json:"source"`
			FundClass string  `json:"fundClass"`
			Account   string  `json:"account"`
			USD       float64 `json:"usd"`
		} `json:"entries"`
	}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Real ledger total: $%.2f (%s mode, %d lines)", m.TotalUsd, m.Mode, m.EntryCount)
	if len(m.ByFundUsd) > 0 {
		b.WriteString("\nFund classes:")
		for k, v := range m.ByFundUsd {
			fmt.Fprintf(&b, " %s $%.0f", k, v)
		}
	}
	if len(m.Entries) > 0 {
		b.WriteString("\nTop holdings:")
		limit := min(len(m.Entries), 6)
		for i := 0; i < limit; i++ {
			e := m.Entries[i]
			fc := ""
			if e.FundClass != "" {
				fc = " [" + e.FundClass + "]"
			}
			acct := ""
			if e.Account != "" {
				acct = " · " + e.Account
			}
			fmt.Fprintf(&b, "\n• %s %s (%s)$%.0f%s%s", e.Amount, e.Asset, e.Source, e.USD, fc, acct)
		}
	}
	return b.String()
}

func summarizeLedgerAccounts(raw string) string {
	var m struct {
		Count    int `json:"count"`
		Accounts []struct {
			Asset   string `json:"asset"`
			Balance string `json:"balance"`
			Source  string `json:"source"`
			Account string `json:"account"`
		} `json:"accounts"`
	}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Active ledger book: %d accounts", m.Count)
	for i, a := range m.Accounts {
		if i >= 5 {
			break
		}
		acct := ""
		if a.Account != "" {
			acct = " · " + a.Account
		}
		fmt.Fprintf(&b, "\n• %s %s (%s)%s", a.Balance, a.Asset, a.Source, acct)
	}
	return b.String()
}

func summarizeSavedDestinations(raw string) string {
	var m struct {
		WalletCount int `json:"walletCount"`
		BankCount   int `json:"bankCount"`
		Wallets     []struct {
			Label   string `json:"label"`
			ChainID string `json:"chainId"`
			Address string `json:"address"`
		} `json:"wallets"`
		Banks []struct {
			Label  string `json:"label"`
			BankID string `json:"bankId"`
			Rail   string `json:"rail"`
			IBAN   string `json:"iban"`
		} `json:"banks"`
	}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Saved destinations: %d wallets, %d bank IBAN accounts", m.WalletCount, m.BankCount)
	for _, w := range m.Wallets {
		addr := w.Address
		short := addr
		if len(addr) > 10 {
			short = addr[:6] + "…" + addr[len(addr)-4:]
		}
		fmt.Fprintf(&b, "\n• Wallet %s on %s (%s)", w.Label, w.ChainID, short)
	}
	for _, bk := range m.Banks {
		iban := bk.IBAN
		if len(iban) > 8 {
			iban = iban[:4] + "…" + iban[len(iban)-4:]
		}
		fmt.Fprintf(&b, "\n• Bank %s %s %s (%s)", bk.Label, bk.BankID, strings.ToUpper(bk.Rail), iban)
	}
	return b.String()
}

func summarizeLedgerStatus(raw string) string {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return ""
	}
	mode, _ := m["mode"].(string)
	bankReady, _ := m["bankReady"].(bool)
	active, _ := m["active"].(bool)
	return fmt.Sprintf("Ledger middleware: mode=%s bank=%v active=%v", mode, bankReady, active)
}

func summarizePortfolio(raw string) string {
	var m struct {
		Address  string            `json:"address"`
		Balances map[string]string `json:"balances"`
	}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return ""
	}
	if len(m.Balances) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Simulated portfolio tokens:")
	n := 0
	for k, v := range m.Balances {
		if n >= 5 {
			break
		}
		fmt.Fprintf(&b, " %s=%s", k, v)
		n++
	}
	return b.String()
}

func summarizeVirtualCards(raw string) string {
	var m struct {
		Production bool   `json:"production"`
		Mode       string `json:"mode"`
		Cards      int    `json:"cards"`
		Active     int    `json:"active"`
	}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return ""
	}
	if m.Cards == 0 {
		return ""
	}
	mode := m.Mode
	if m.Production {
		mode = "production"
	}
	return fmt.Sprintf("Virtual cards: %d active of %d (%s · Apple Pay · Google Pay · 3DS)", m.Active, m.Cards, mode)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
