package ledger

import "testing"

func TestResolveSettlementKind(t *testing.T) {
	k := ResolveSettlementKind(SettlementRequest{
		FromAccount: "m1-usd",
		Amount:      "100",
		ExternalTo:  "bsc:0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
	})
	if k != SettlementRealCrypto {
		t.Fatalf("want real_crypto got %s", k)
	}
	k = ResolveSettlementKind(SettlementRequest{
		ExternalTo: "bank:hsbc:swift:GB82WEST12345698765432",
	})
	if k != SettlementRealFiat {
		t.Fatalf("want real_fiat got %s", k)
	}
	k = ResolveSettlementKind(SettlementRequest{
		ToAccount: "vault-bnb",
		Amount:    "10",
	})
	if k != SettlementInternal {
		t.Fatalf("want internal got %s", k)
	}
}

func TestSettleVaultConvert(t *testing.T) {
	dir := t.TempDir()
	store := NewBookStore(dir)
	store.data = &Book{Accounts: map[string]*BookAccount{
		"m1-usd": {ID: "m1-usd", Asset: "USD", Balance: "5000", FundClass: FundM1, Source: SourceBank, Mode: ModeBank},
	}}
	_ = store.save()

	prices := map[string]PriceQuote{"BNB": {USD: 600}}
	res, err := store.Settle(SettlementRequest{
		FromAccount: "m1-usd",
		Amount:      "100",
		PayoutAsset: "BNB",
		Kind:        "vault",
	}, prices, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "completed" {
		t.Fatalf("status %s", res.Status)
	}
	if res.Settlement.Kind != SettlementVault {
		t.Fatalf("kind %s", res.Settlement.Kind)
	}
	list := store.ListSettlements(5)
	if len(list) != 1 {
		t.Fatalf("expected 1 settlement got %d", len(list))
	}
}

func TestQuoteSettlement(t *testing.T) {
	res, err := QuoteSettlement(SettlementRequest{
		FromAccount: "m0-usd",
		Amount:      "1000",
		PayoutAsset: "USDT",
		ExternalTo:  "bsc:0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
		Preview:     true,
	}, map[string]PriceQuote{"USDT": {USD: 1}}, nil, "USD")
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "preview" {
		t.Fatalf("status %s", res.Status)
	}
	if len(res.Settlement.Steps) < 2 {
		t.Fatalf("steps %+v", res.Settlement.Steps)
	}
}
