package ledger

import "testing"

func TestImportStableID(t *testing.T) {
	id := importStableID("BTC", "", 0)
	if id != "import-btc" {
		t.Fatalf("got %s", id)
	}
	id2 := importStableID("USD", "acct1", 0)
	if id2 != "import-usd-acct1" {
		t.Fatalf("got %s", id2)
	}
}

func TestSyncImportEntries(t *testing.T) {
	dir := t.TempDir()
	store := NewBookStore(dir)
	valued := []Entry{
		{ID: "import-btc", Source: SourceImport, Asset: "BTC", Human: "0.5", Mode: ModeReal, FiatUSD: 25000},
		{ID: "import-usd", Source: SourceImport, Asset: "USD", Human: "1000", Mode: ModeFiat, FiatUSD: 1000},
	}
	n, err := store.SyncImportEntries(valued)
	if err != nil || n != 2 {
		t.Fatalf("synced %d err %v", n, err)
	}
	accts, err := store.ListAccounts()
	if err != nil || len(accts) != 2 {
		t.Fatalf("accounts %+v err %v", accts, err)
	}
}

func TestValueImportEntries(t *testing.T) {
	entries := []Entry{{Source: SourceImport, Asset: "BTC", Human: "1"}}
	out := ValueImportEntries(entries, map[string]PriceQuote{"BTC": {USD: 50000}}, "USD")
	if out[0].FiatUSD != 50000 {
		t.Fatalf("fiat usd %v", out[0].FiatUSD)
	}
}
