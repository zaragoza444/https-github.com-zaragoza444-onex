package ledger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeNationalSovereignBank(t *testing.T) {
	for _, raw := range []string{"national_sovereign_bank", "National Sovereign Bank", "sovereign_bank"} {
		if got := NormalizeFundClass(raw); got != FundNSB {
			t.Fatalf("%q => %q want nsb", raw, got)
		}
	}
}

func TestParseBankM0M1NSB(t *testing.T) {
	data := []byte(`{
	  "accounts": [
	    {"id":"m0-usd","fundClass":"m0","currency":"USD","balance":"1000"},
	    {"id":"m1-eur","moneySupply":"m1","currency":"EUR","balance":"2000"},
	    {"id":"nsb-gbp","bank":"nsb","currency":"GBP","balance":"3000"}
	  ]
	}`)
	entries, err := parseBankJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries got %d", len(entries))
	}
	byFC := map[string]string{}
	for _, e := range entries {
		byFC[e.FundClass] = e.Asset
	}
	if byFC[FundM0] != "USD" || byFC[FundM1] != "EUR" || byFC[FundNSB] != "GBP" {
		t.Fatalf("unexpected fund classes %+v", byFC)
	}
}

func TestFilterEntriesByFundClass(t *testing.T) {
	entries := []Entry{
		{FundClass: FundM0, FiatUSD: 100},
		{FundClass: FundM1, FiatUSD: 200},
		{FundClass: FundNSB, FiatUSD: 300},
	}
	m0 := filterEntriesByFundClass(entries, FundM0)
	if len(m0) != 1 || m0[0].FiatUSD != 100 {
		t.Fatalf("m0 filter failed %+v", m0)
	}
}

func TestSummarizeByFundClass(t *testing.T) {
	snap := Summarize([]Entry{
		{FundClass: FundM0, FiatUSD: 10, FiatCurrency: "USD", FiatValue: 10},
		{FundClass: FundM1, FiatUSD: 20, FiatCurrency: "USD", FiatValue: 20},
		{FundClass: FundNSB, FiatUSD: 30, FiatCurrency: "USD", FiatValue: 30},
	}, "production")
	if snap.ByFundClass[FundM0] != 10 || snap.ByFundClass[FundNSB] != 30 {
		t.Fatalf("byFundUsd %+v", snap.ByFundClass)
	}
}

func TestBookConvertFromM1(t *testing.T) {
	dir := t.TempDir()
	store := NewBookStore(dir)
	store.data = &Book{Accounts: map[string]*BookAccount{
		"m1-usd": {ID: "m1-usd", Asset: "USD", Balance: "10000", FundClass: FundM1, Source: SourceBank, Mode: ModeBank},
	}}
	_ = store.save()

	prices := map[string]PriceQuote{"BTC": {USD: 50000}}
	conv, err := store.ConvertActive(ConvertRequest{
		FromAccount: "m1-usd",
		FromAsset:   "USD",
		ToAsset:     "BTC",
		Amount:      "1000",
	}, prices, nil)
	if err != nil {
		t.Fatal(err)
	}
	if conv.FundClass != FundM1 {
		t.Fatalf("expected m1 fund class got %s", conv.FundClass)
	}
}

func TestParseZBankLedgerSeedM1toM4(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	path := filepath.Join(root, "configs", "bank-ledger.zbank.example.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read zbank seed: %v", err)
	}
	entries, err := parseBankJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]string{}
	for _, e := range entries {
		byID[e.ID] = e.FundClass
	}
	want := map[string]string{
		"zbank-usd-cash":        FundM0,
		"zbank-usd-checking":    FundM1,
		"zbank-eur-checking":    FundM1,
		"zbank-usd-safeguarded": FundM2,
		"zbank-usd-treasury":    FundM3,
		"zbank-usd-wholesale":   FundM4,
		"zbank-gbp-wholesale":   FundM4,
	}
	for id, fc := range want {
		if byID[id] != fc {
			t.Fatalf("%s fundClass=%q want %q (got map %+v)", id, byID[id], fc, byID)
		}
	}
	for id := range byID {
		if len(id) >= 5 && id[:5] == "nova-" {
			t.Fatalf("zbank seed must not include nova account %s", id)
		}
	}
}

func TestLedgerStatusExposesM2M3M4(t *testing.T) {
	st := Config{Mode: "demo"}.Status()
	classes, _ := st["fundClasses"].([]string)
	want := map[string]bool{FundM0: true, FundM1: true, FundM2: true, FundM3: true, FundM4: true, FundNSB: true}
	for _, c := range classes {
		delete(want, c)
	}
	if len(want) != 0 {
		t.Fatalf("missing fund classes in status: %v (got %v)", want, classes)
	}
	labels, _ := st["fundClassLabels"].(map[string]string)
	if labels[FundM2] == "" || labels[FundM3] == "" || labels[FundM4] == "" {
		t.Fatalf("missing M2–M4 labels: %+v", labels)
	}
}
