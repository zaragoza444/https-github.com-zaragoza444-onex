package ledger

import "testing"

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
