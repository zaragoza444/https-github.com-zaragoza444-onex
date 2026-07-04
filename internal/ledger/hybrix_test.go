package ledger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHybrixMirrorSyncAndConvert(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ONEX_HOME_DIR", dir)
	t.Setenv("ONEX_ONLINE_BANK", "1")
	t.Setenv("ONEX_HYBRIX_ENABLED", "1")

	bank := &OnlineBankStore{path: filepath.Join(dir, "online-bank.json")}
	st := &OnlineBankState{
		Name: defaultOnlineBankName, Online: true,
		Accounts: []OnlineBankAccount{
			{ID: "a1", Name: "USD", Currency: "USD", Balance: "1000.00", Status: "active"},
		},
	}
	if err := bank.save(st); err != nil {
		t.Fatal(err)
	}
	mirrors, err := SyncMirrorsFromOnlineBank(bank)
	if err != nil {
		t.Fatal(err)
	}
	if len(mirrors) != 1 || mirrors[0].Symbol != "usd" {
		t.Fatalf("mirrors %+v", mirrors)
	}
	res, err := HybrixConvert(bank, HybrixConvertRequest{
		Direction: "nsb-to-hybx", NSBAccount: "a1", Amount: "100",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res["status"] != "completed" {
		t.Fatalf("convert %+v", res)
	}
}

func TestHybrixSettlementRef(t *testing.T) {
	t.Setenv("ONEX_ONLINE_BANK", "1")
	t.Setenv("ONEX_HYBRIX_ENABLED", "1")
	ref, err := InitiateHybrixTransfer(BankTransferRequest{
		Rail: RailSEPA, BankName: "hybx", Account: "DE89370400440532013000",
		Amount: "50.00", Asset: "EUR", Reference: "TEST",
	}, "TEST")
	if err != nil || ref == "" {
		t.Fatalf("ref %s err %v", ref, err)
	}
}

func TestHybrixConfig(t *testing.T) {
	t.Setenv("ONEX_ONLINE_BANK", "1")
	t.Setenv("ONEX_HYBRIX_ENABLED", "0")
	cfg := LoadHybrixConfig()
	if cfg.Enabled {
		t.Fatal("expected disabled")
	}
	os.Unsetenv("ONEX_HYBRIX_ENABLED")
	cfg2 := LoadHybrixConfig()
	if !cfg2.Enabled {
		t.Fatal("expected enabled with online bank")
	}
}
