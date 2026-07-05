package bridge

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/onex-blockchain/onex/internal/ledger"
)

func TestCardWireTransferPassesOfficerPINToOMNL(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ONEX_HOME_DIR", dir)
	t.Setenv("ONEX_ONLINE_BANK", "1")
	t.Setenv("ONEX_LEDGER_MODE", "development")

	b := New(Config{ProjectRoot: dir})
	b.cardStore = &virtualCardStore{path: filepath.Join(dir, "virtual-cards.json"), seedDir: dir}
	bankState := ledger.OnlineBankState{
		Name: "Test Bank", Online: true, SWIFT: "TESTBIC",
		Accounts: []ledger.OnlineBankAccount{
			{ID: "card-account", Name: "Card Account", Currency: "USD", Balance: "100.00", Status: "active"},
			{
				ID: "omnl", Name: "OMNL Central Bank", Currency: "USD", Balance: "500.00",
				IBAN: "OMNL00US00000000000001", Bank: "omnl", Status: "active",
				OfficerPINRequired: true, OfficerPINHash: "246810",
			},
		},
	}
	raw, err := json.Marshal(bankState)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "online-bank.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	card := b.buildCardFromAccount(ledger.OnlineBankAccount{
		ID: "card-account", Name: "Card Account", Currency: "USD", Balance: "100.00", Status: "active",
	}, "visa", "Card 1", "nsb")
	if err := b.cards().save(&virtualCardFile{
		Issuer: "NSB Virtual Cards", Production: true,
		Cards: []VirtualCard{card},
	}); err != nil {
		t.Fatal(err)
	}

	req := CardWireRequest{
		CardID: card.ID, Amount: "5", BeneficiaryIBAN: "OMNL00US00000000000001",
		BeneficiaryName: "OMNL Central Bank", Preview: false,
	}
	req.Preview = true
	if _, err := b.WireTransferCard(context.Background(), req); err == nil ||
		!strings.Contains(err.Error(), "officer PIN required") {
		t.Fatalf("expected missing officer PIN preview error, got %v", err)
	}

	req.Preview = false
	req.OfficerPIN = "246810"
	res, err := b.WireTransferCard(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	transfer, _ := res["transfer"].(*ledger.OnlineBankTransferResult)
	if transfer == nil || transfer.Transaction == nil || !transfer.Transaction.OfficerAuthorized {
		t.Fatalf("expected officer-authorized transfer, got %#v", res["transfer"])
	}
}
