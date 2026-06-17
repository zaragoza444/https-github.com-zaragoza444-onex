package ledger

import (
	"path/filepath"
	"testing"
)

func TestParseAnyLedgerBalanceMap(t *testing.T) {
	data := []byte(`{"balances":{"BTC":"0.25","ETH":"1.5","USD":"9000"}}`)
	entries, err := ParseAnyLedger(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries got %d", len(entries))
	}
}

func TestBookTransferWithConvert(t *testing.T) {
	dir := t.TempDir()
	store := NewBookStore(dir)
	store.data = &Book{Accounts: map[string]*BookAccount{
		"usd-acct": {ID: "usd-acct", Asset: "USD", Balance: "10000", Source: SourceBank, Mode: ModeBank},
	}}
	_ = store.save()

	prices := map[string]PriceQuote{"BTC": {USD: 50000}}
	res, err := store.Transfer(TransferRequest{
		FromAccount: "usd-acct",
		ToAccount:   "btc-wallet",
		Amount:      "5000",
		ConvertTo:   "BTC",
	}, prices, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Convert == nil || res.Convert.ToAmount == "" {
		t.Fatal("expected conversion result")
	}
	from, _ := store.GetAccount("usd-acct")
	if parseHuman(from.Balance) != 5000 {
		t.Fatalf("expected 5000 USD left got %s", from.Balance)
	}
	to, _ := store.GetAccount("btc-wallet")
	if parseHuman(to.Balance) <= 0 {
		t.Fatalf("expected btc credit got %s", to.Balance)
	}
}

func TestBookStorePath(t *testing.T) {
	dir := t.TempDir()
	s := NewBookStore(dir)
	if s.path != filepath.Join(dir, "ledger-book.json") {
		t.Fatalf("unexpected path %s", s.path)
	}
}
