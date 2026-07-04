package ledger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCashCodeIssueRedeem(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("ONEX_HOME_DIR", dir)
	os.Setenv("ONEX_CASHCODE_ENABLED", "1")
	t.Cleanup(func() {
		os.Unsetenv("ONEX_HOME_DIR")
		os.Unsetenv("ONEX_CASHCODE_ENABLED")
	})

	store := &CashCodeStore{path: filepath.Join(dir, "cash-codes.json")}
	issue, err := store.Issue(CashCodeIssueRequest{
		FromAccount: "nsb-operating", Amount: "50.00", Currency: "USD", PIN: "1234",
	}, "NSB Operating")
	if err != nil {
		t.Fatal(err)
	}
	if issue.Code == "" || issue.Preview {
		t.Fatal("expected issued code")
	}

	verify, err := store.Verify(issue.Code, "1234")
	if err != nil || !verify.Valid {
		t.Fatalf("verify: %v %+v", err, verify)
	}
	verifyBad, _ := store.Verify(issue.Code, "0000")
	if verifyBad.Valid {
		t.Fatal("expected invalid pin")
	}

	redeem, err := store.Redeem(CashCodeRedeemRequest{
		Code: issue.Code, PIN: "1234", ToAccount: "nsb-reserve",
	})
	if err != nil || redeem.CashCode.Status != cashCodeStatusRedeemed {
		t.Fatalf("redeem: %v %+v", err, redeem)
	}

	verify2, _ := store.Verify(issue.Code, "1234")
	if verify2.Valid {
		t.Fatal("code should not be valid after redeem")
	}
}

func TestGenerateCashCodeFormat(t *testing.T) {
	code := generateCashCode()
	if len(code) != 14 || code[4] != '-' || code[9] != '-' {
		t.Fatalf("bad format: %q", code)
	}
	if normalizeCashCode(code) != code {
		t.Fatal("normalize changed code")
	}
}
