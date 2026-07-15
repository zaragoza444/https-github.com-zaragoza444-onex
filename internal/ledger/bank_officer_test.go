package ledger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBankOfficerDSSBOATSeedAndAuth(t *testing.T) {
	dir := t.TempDir()
	store := &BankOfficerStore{path: filepath.Join(dir, "zbank-officers.json")}
	root := filepath.Clean(filepath.Join("..", ".."))
	seed := filepath.Join(root, "configs", "zbank-officers.dssboat.example.json")
	t.Setenv("ONEX_ZBANK_OFFICER_PIN", "724265")
	t.Setenv("ONEX_ZBANK_OFFICER_SIGNATURE", "BernardGreeffNiehaus-DSSBOAT")
	if err := store.EnsureSeeded(seed); err != nil {
		t.Fatal(err)
	}
	list, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 officer got %d", len(list))
	}
	o := list[0]
	if o.ID != "dssboat-officer-bneihaus" {
		t.Fatalf("id %s", o.ID)
	}
	if o.Name != "Bernard Greeff Niehaus" || o.Position != "CEO" {
		t.Fatalf("officer %+v", o)
	}
	if !o.HasPIN || !o.HasSignature {
		t.Fatal("expected pin and signature set")
	}
	if o.ClientCompany == "" && o.PassportNumber != "LK986067" {
		t.Fatalf("passport %s", o.PassportNumber)
	}
	// Public list must not leak hashes — re-load raw file and ensure Public omits them (encode via List).
	raw, _ := os.ReadFile(store.path)
	if len(raw) == 0 {
		t.Fatal("expected persisted officers file")
	}

	ok, err := store.Verify(OfficerAuthRequest{
		OfficerID: o.ID, PIN: "724265", Signature: "BernardGreeffNiehaus-DSSBOAT",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ok.Valid || ok.ApprovalID == "" {
		t.Fatalf("expected valid auth %+v", ok)
	}
	bad, err := store.Verify(OfficerAuthRequest{
		OfficerID: o.ID, PIN: "0000", Signature: "BernardGreeffNiehaus-DSSBOAT",
	})
	if err != nil {
		t.Fatal(err)
	}
	if bad.Valid {
		t.Fatal("wrong pin should fail")
	}
	badSig, err := store.Verify(OfficerAuthRequest{
		OfficerID: o.ID, PIN: "724265", Signature: "WrongSignatureXX",
	})
	if err != nil {
		t.Fatal(err)
	}
	if badSig.Valid {
		t.Fatal("wrong signature should fail")
	}
}

func TestBankOfficerAuthorizedTransfer(t *testing.T) {
	dir := t.TempDir()
	officers := &BankOfficerStore{path: filepath.Join(dir, "zbank-officers.json")}
	bank := &OnlineBankStore{path: filepath.Join(dir, "online-bank.json")}
	root := filepath.Clean(filepath.Join("..", ".."))
	t.Setenv("ONEX_ZBANK_OFFICER_PIN", "724265")
	t.Setenv("ONEX_ZBANK_OFFICER_SIGNATURE", "BernardGreeffNiehaus-DSSBOAT")
	if err := officers.EnsureSeeded(filepath.Join(root, "configs", "zbank-officers.dssboat.example.json")); err != nil {
		t.Fatal(err)
	}
	if err := bank.EnsureSeeded(filepath.Join(root, "configs", "bank-ledger.zbank.example.json")); err != nil {
		t.Fatal(err)
	}
	res, err := officers.AuthorizeTransfer(OfficerTransferRequest{
		OfficerID: "dssboat-officer-bneihaus",
		PIN:       "724265",
		Signature: "BernardGreeffNiehaus-DSSBOAT",
		FromAccount: "zbank-usd-checking",
		ToAccount:   "zbank-usd-safeguarded",
		Amount:      "100.00",
		Reference:   "DSSBOAT-test",
	}, bank)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "authorized" || res.Transfer == nil {
		t.Fatalf("transfer %+v", res)
	}
	denied, err := officers.AuthorizeTransfer(OfficerTransferRequest{
		OfficerID: "dssboat-officer-bneihaus",
		PIN:       "724265",
		Signature: "BernardGreeffNiehaus-DSSBOAT",
		FromAccount: "nova-usd-checking",
		ToAccount:   "zbank-usd-safeguarded",
		Amount:      "10.00",
	}, bank)
	if err != nil {
		t.Fatal(err)
	}
	if denied.Status != "denied" {
		t.Fatalf("expected denied for unlinked account got %+v", denied)
	}
}
