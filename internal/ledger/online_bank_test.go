package ledger

import (
	"os"
	"path/filepath"
	"strings"
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

func TestOnlineBankOfficerPINRequiredForOMNLTransfer(t *testing.T) {
	dir := t.TempDir()
	store := &OnlineBankStore{path: filepath.Join(dir, "online-bank.json")}
	st := &OnlineBankState{
		Name: defaultOnlineBankName, Online: true, SWIFT: defaultOnlineBankSWIFT,
		Accounts: []OnlineBankAccount{
			{ID: "operating", Name: "Operating", Currency: "USD", Balance: "1000.00", Status: "active"},
			{
				ID: "omnl", Name: "OMNL Central Bank", Currency: "USD", Balance: "500.00",
				IBAN: "OMNL00US00000000000001", Bank: "omnl", Status: "active",
				OfficerPINRequired: true, OfficerPINHash: hashOfficerPIN("246810"),
			},
		},
	}
	if err := store.save(st); err != nil {
		t.Fatal(err)
	}

	if _, err := store.Transfer(OnlineBankTransferRequest{
		FromAccount: "operating", ToAccount: "omnl", Amount: "25.00",
	}); err == nil || !strings.Contains(err.Error(), "officer PIN required") {
		t.Fatalf("expected missing officer PIN error, got %v", err)
	}
	if _, err := store.Transfer(OnlineBankTransferRequest{
		FromAccount: "operating", ToAccount: "omnl", Amount: "25.00", OfficerPIN: "000000",
	}); err == nil || !strings.Contains(err.Error(), "invalid officer PIN") {
		t.Fatalf("expected invalid officer PIN error, got %v", err)
	}

	accts, err := store.ListAccounts()
	if err != nil {
		t.Fatal(err)
	}
	for _, acct := range accts {
		if acct.ID == "omnl" {
			if acct.OfficerPINHash != "" || !acct.OfficerPINConfigured || !acct.OfficerPINRequired {
				t.Fatalf("officer PIN metadata leaked or missing: %+v", acct)
			}
		}
	}

	res, err := store.Transfer(OnlineBankTransferRequest{
		FromAccount: "operating", ToAccount: "omnl", Amount: "25.00", OfficerPIN: "246810",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "completed" || !res.Transaction.OfficerAuthorized {
		t.Fatalf("expected authorized completed transfer, got %+v", res)
	}
	accts, _ = store.ListAccounts()
	for _, acct := range accts {
		switch acct.ID {
		case "operating":
			if acct.Balance != "975" && acct.Balance != "975.00" && acct.Balance != "975.0" {
				t.Fatalf("operating balance %s", acct.Balance)
			}
		case "omnl":
			if acct.Balance != "525" && acct.Balance != "525.00" && acct.Balance != "525.0" {
				t.Fatalf("omnl balance %s", acct.Balance)
			}
		}
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

func TestOnlineBankWireAndStatement(t *testing.T) {
	dir := t.TempDir()
	store := &OnlineBankStore{path: filepath.Join(dir, "online-bank.json")}
	st := &OnlineBankState{
		Name: defaultOnlineBankName, Online: true, SWIFT: defaultOnlineBankSWIFT,
		Accounts: []OnlineBankAccount{
			{ID: "a1", Name: "Checking", Currency: "USD", Balance: "100.00", IBAN: "US00M1USD00000000000001", Status: "active"},
		},
		Transactions: []OnlineBankTransaction{
			{ID: "t1", Type: "deposit", ToAccount: "a1", Amount: "100.00", Currency: "USD", Status: "completed", CreatedAt: 1700000000},
		},
	}
	if err := store.save(st); err != nil {
		t.Fatal(err)
	}
	wire, err := store.WireInstructions("a1")
	if err != nil || wire.IBAN != "US00M1USD00000000000001" || wire.SWIFT != defaultOnlineBankSWIFT {
		t.Fatalf("wire %+v err %v", wire, err)
	}
	txs, err := store.ListTransactionsFiltered(10, "a1", "deposit")
	if err != nil || len(txs) != 1 {
		t.Fatalf("filtered txs %v err %v", txs, err)
	}
	csv, err := store.ExportTransactionsCSV("a1")
	if err != nil || !strings.Contains(csv, "deposit") {
		t.Fatalf("csv %q err %v", csv, err)
	}
}
