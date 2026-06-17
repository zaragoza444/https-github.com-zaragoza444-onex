package ledger

import "testing"

func TestParseExternalChain(t *testing.T) {
	dest, err := ParseExternalDestination("ethereum:0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0")
	if err != nil {
		t.Fatal(err)
	}
	if dest.Kind != ExternalEVM || dest.ChainID != "ethereum" {
		t.Fatalf("unexpected %+v", dest)
	}
}

func TestParseExternalBank(t *testing.T) {
	dest, err := ParseExternalDestination("bank:hsbc:swift:GB82WEST12345698765432")
	if err != nil {
		t.Fatal(err)
	}
	if dest.Kind != ExternalBank || dest.BankRail != RailSWIFT {
		t.Fatalf("unexpected %+v", dest)
	}
}

func TestParseExternalBitcoin(t *testing.T) {
	dest, err := ParseExternalDestination("bitcoin:bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh")
	if err != nil {
		t.Fatal(err)
	}
	if dest.Kind != ExternalBitcoin {
		t.Fatalf("expected bitcoin got %s", dest.Kind)
	}
}

func TestSupportedExternals(t *testing.T) {
	ext := SupportedExternals()
	if len(ext["chains"].([]SupportedChain)) < 10 {
		t.Fatal("expected many chains")
	}
	if len(ext["banks"].([]SupportedBank)) < 5 {
		t.Fatal("expected many banks")
	}
}
