package ledger

import "strings"

// Money-supply / bank fund classes for fiat ledger conversion.
const (
	FundM0  = "m0"
	FundM1  = "m1"
	FundNSB = "nsb"
)

// NormalizeFundClass maps aliases to m0, m1, or nsb.
func NormalizeFundClass(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "m0", "mb", "base", "narrow", "cash", "physical", "central":
		return FundM0
	case "m1", "broad", "demand", "checking", "transactional":
		return FundM1
	case "nsb", "national_sovereign_bank", "nationalsovereignbank", "sovereign_bank", "sovereign",
		"national_savings", "nationalsavings", "savings_bank":
		return FundNSB
	}
	if strings.Contains(s, "nsb") || strings.Contains(s, "national sovereign") ||
		strings.Contains(s, "sovereign bank") || strings.Contains(s, "national savings") {
		return FundNSB
	}
	return s
}

func isFundClassSource(src string) bool {
	switch NormalizeFundClass(src) {
	case FundM0, FundM1, FundNSB:
		return true
	}
	return false
}

func filterEntriesByFundClass(entries []Entry, fc string) []Entry {
	want := NormalizeFundClass(fc)
	if want == "" {
		return entries
	}
	out := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if strings.EqualFold(e.FundClass, want) {
			out = append(out, e)
		}
	}
	return out
}

func resolveBankFundClass(acct BankAccount, id string) string {
	for _, raw := range []string{acct.FundClass, acct.MoneySupply, acct.Bank, acct.Type, id} {
		if fc := NormalizeFundClass(raw); fc == FundM0 || fc == FundM1 || fc == FundNSB {
			return fc
		}
	}
	lower := strings.ToLower(id)
	switch {
	case strings.HasPrefix(lower, "m0-"), strings.Contains(lower, ":m0"), strings.Contains(lower, "_m0"):
		return FundM0
	case strings.HasPrefix(lower, "m1-"), strings.Contains(lower, ":m1"), strings.Contains(lower, "_m1"):
		return FundM1
	case strings.HasPrefix(lower, "nsb-"), strings.Contains(lower, "nsb"):
		return FundNSB
	}
	if strings.EqualFold(acct.Bank, "nsb") ||
		strings.Contains(strings.ToLower(acct.Name), "nsb") ||
		strings.Contains(strings.ToLower(acct.Name), "sovereign") {
		return FundNSB
	}
	return ""
}
