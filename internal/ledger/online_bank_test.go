package ledger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOnlineBankInternalTransfer(t *testing.T) {
	dir := t.TempDir()
	store := &OnlineBankStore{path: filepath.Join(dir, "online-bank.json")}
	st := &OnlineBankState{
		Name: defaultOnlineBankName, Online: true, SWIFT: defaultOnlineBankSWIFT,
		Accounts: []OnlineBankAccount{
			{ID: "a1", Name: "Checking", Currency: "USD", Balance: "1000.00", IBAN: "US00M1USD00000000000001", Status: "active"},
			{ID: "a2", Name: "Savings", Currency: "USD", Balance: "500.00", IBAN: "US00M1USD00000000000002", Status: "active"},
		},
	}
	if err := store.save(st); err != nil {
		t.Fatal(err)
	}
	res, err := store.Transfer(OnlineBankTransferRequest{
		FromAccount: "a1", ToAccount: "a2", Amount: "250.50",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Transaction.Status != "completed" {
		t.Fatalf("status %s", res.Transaction.Status)
	}
	accts, _ := store.ListAccounts()
	var a1, a2 OnlineBankAccount
	for _, a := range accts {
		if a.ID == "a1" {
			a1 = a
		}
		if a.ID == "a2" {
			a2 = a
		}
	}
	if a1.Balance != "749.50" && a1.Balance != "749.5" {
		t.Fatalf("a1 balance %s", a1.Balance)
	}
	if a2.Balance != "750.50" && a2.Balance != "750.5" {
		t.Fatalf("a2 balance %s", a2.Balance)
	}
}

func TestOnlineBankDeposit(t *testing.T) {
	dir := t.TempDir()
	store := &OnlineBankStore{path: filepath.Join(dir, "online-bank.json")}
	st := &OnlineBankState{
		Name: defaultOnlineBankName, Online: true,
		Accounts: []OnlineBankAccount{
			{ID: "a1", Name: "Checking", Currency: "USD", Balance: "100.00", Status: "active"},
		},
	}
	if err := store.save(st); err != nil {
		t.Fatal(err)
	}
	res, err := store.Deposit(OnlineBankDepositRequest{
		ToAccount: "a1", Amount: "500", Source: "wire", Reference: "TEST-DEP",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "completed" || (res.ToBalance != "600" && res.ToBalance != "600.00" && res.ToBalance != "600.0") {
		t.Fatalf("deposit result %+v", res)
	}
	snap, err := store.BankLedger(nil, 10)
	if err != nil || len(snap.Entries) != 1 {
		t.Fatalf("ledger snap %v err %v", snap, err)
	}
	if len(snap.Transactions) != 1 || snap.Transactions[0].Type != "deposit" {
		t.Fatalf("txs %+v", snap.Transactions)
	}
}

func TestOnlineBankSeedFromFile(t *testing.T) {
	dir := t.TempDir()
	bankFile := filepath.Join(dir, "bank.json")
	data := []byte(`{"accounts":[{"id":"x1","name":"Test","iban":"DE89370400440532013000","currency":"EUR","balance":"100.00"}]}`)
	if err := os.WriteFile(bankFile, data, 0o644); err != nil {
		t.Fatal(err)
	}
	store := &OnlineBankStore{path: filepath.Join(dir, "online-bank.json")}
	if err := store.EnsureSeeded(bankFile); err != nil {
		t.Fatal(err)
	}
	accts, err := store.ListAccounts()
	if err != nil || len(accts) != 1 {
		t.Fatalf("accounts %v err %v", accts, err)
	}
	if accts[0].IBAN != "DE89370400440532013000" {
		t.Fatalf("iban %s", accts[0].IBAN)
	}
}
