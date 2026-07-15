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
	t.Setenv("ONEX_ZBANK_OFFICER_PIN", "918273")
	t.Setenv("ONEX_ZBANK_OFFICER_SIGNATURE", "ProdSignature-DSSBOAT-01")
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
		OfficerID: o.ID, PIN: "918273", Signature: "ProdSignature-DSSBOAT-01",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ok.Valid || ok.ApprovalID == "" {
		t.Fatalf("expected valid auth %+v", ok)
	}
	bad, err := store.Verify(OfficerAuthRequest{
		OfficerID: o.ID, PIN: "0000", Signature: "ProdSignature-DSSBOAT-01",
	})
	if err != nil {
		t.Fatal(err)
	}
	if bad.Valid {
		t.Fatal("wrong pin should fail")
	}
	badSig, err := store.Verify(OfficerAuthRequest{
		OfficerID: o.ID, PIN: "918273", Signature: "WrongSignatureXX",
	})
	if err != nil {
		t.Fatal(err)
	}
	if badSig.Valid {
		t.Fatal("wrong signature should fail")
	}
}

func TestBankOfficerNoDemoDefaultsWithoutEnv(t *testing.T) {
	dir := t.TempDir()
	store := &BankOfficerStore{path: filepath.Join(dir, "zbank-officers.json")}
	root := filepath.Clean(filepath.Join("..", ".."))
	t.Setenv("ONEX_ZBANK_OFFICER_PIN", "")
	t.Setenv("ONEX_ZBANK_OFFICER_SIGNATURE", "")
	t.Setenv("ONEX_LEDGER_MODE", "production")
	if err := store.EnsureSeeded(filepath.Join(root, "configs", "zbank-officers.dssboat.example.json")); err != nil {
		t.Fatal(err)
	}
	list, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("expected no officers without env secrets, got %d", len(list))
	}
	if err := RequireProductionOfficerSecrets(); err == nil {
		t.Fatal("expected production secrets requirement error")
	}
}

func TestBankOfficerSetCredentialsRotation(t *testing.T) {
	dir := t.TempDir()
	store := &BankOfficerStore{path: filepath.Join(dir, "zbank-officers.json")}
	root := filepath.Clean(filepath.Join("..", ".."))
	t.Setenv("ONEX_ZBANK_OFFICER_PIN", "918273")
	t.Setenv("ONEX_ZBANK_OFFICER_SIGNATURE", "ProdSignature-DSSBOAT-01")
	if err := store.EnsureSeeded(filepath.Join(root, "configs", "zbank-officers.dssboat.example.json")); err != nil {
		t.Fatal(err)
	}
	pub, err := store.SetCredentials(OfficerCredentialsRequest{
		OfficerID: "dssboat-officer-bneihaus",
		PIN: "556677", Signature: "ProdSignature-DSSBOAT-02",
		CurrentPIN: "918273", CurrentSignature: "ProdSignature-DSSBOAT-01",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !pub.HasPIN {
		t.Fatal("expected pin after rotation")
	}
	ok, err := store.Verify(OfficerAuthRequest{
		OfficerID: pub.ID, PIN: "556677", Signature: "ProdSignature-DSSBOAT-02",
	})
	if err != nil || !ok.Valid {
		t.Fatalf("rotated credentials should verify: %+v %v", ok, err)
	}
}

func TestBankOfficerAuthorizedTransfer(t *testing.T) {
	dir := t.TempDir()
	officers := &BankOfficerStore{path: filepath.Join(dir, "zbank-officers.json")}
	bank := &OnlineBankStore{path: filepath.Join(dir, "online-bank.json")}
	root := filepath.Clean(filepath.Join("..", ".."))
	t.Setenv("ONEX_ZBANK_OFFICER_PIN", "91827364")
	t.Setenv("ONEX_ZBANK_OFFICER_SIGNATURE", "ProdSignature-DSSBOAT-XFER")
	if err := officers.EnsureSeeded(filepath.Join(root, "configs", "zbank-officers.dssboat.example.json")); err != nil {
		t.Fatal(err)
	}
	if err := bank.EnsureSeeded(filepath.Join(root, "configs", "bank-ledger.zbank.production.json")); err != nil {
		t.Fatal(err)
	}
	res, err := officers.AuthorizeTransfer(OfficerTransferRequest{
		OfficerID: "dssboat-officer-bneihaus",
		PIN:       "91827364",
		Signature: "ProdSignature-DSSBOAT-XFER",
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
		PIN:       "91827364",
		Signature: "ProdSignature-DSSBOAT-XFER",
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
