package ledger

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFineractParseSavingsAccount(t *testing.T) {
	raw := []byte(`{
		"id": 42,
		"accountNo": "000000042",
		"clientId": 7,
		"clientName": "HYBX Client",
		"productName": "Passbook Savings",
		"currency": {"code": "USD"},
		"status": {"value": "Active"},
		"summary": {"accountBalance": 1500.25, "availableBalance": 1500.25}
	}`)
	acct, err := parseFineractSavings(raw)
	if err != nil {
		t.Fatal(err)
	}
	if acct.ID != 42 || acct.AccountNo != "000000042" || acct.Currency != "USD" {
		t.Fatalf("unexpected account: %+v", acct)
	}
	if acct.AccountBalance != 1500.25 {
		t.Fatalf("balance: %v", acct.AccountBalance)
	}
}

func TestSyncFineractToOnlineBank(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/savingsaccounts" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Fineract-Platform-TenantId") != "default" {
			t.Errorf("tenant header missing")
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"pageItems": []map[string]interface{}{
				{
					"id": 1, "accountNo": "000000001", "clientId": 1, "clientName": "Alice",
					"productName": "Savings", "currency": map[string]string{"code": "EUR"},
					"status": map[string]string{"value": "Active"},
					"summary": map[string]float64{"accountBalance": 500, "availableBalance": 500},
				},
			},
		})
	}))
	defer srv.Close()

	dir := t.TempDir()
	t.Setenv("ONEX_HOME", dir)
	t.Setenv("ONEX_FINERACT_ENABLED", "1")
	t.Setenv("ONEX_FINERACT_URL", srv.URL)
	t.Setenv("ONEX_FINERACT_USERNAME", "user")
	t.Setenv("ONEX_FINERACT_PASSWORD", "pass")
	t.Setenv("ONEX_ONLINE_BANK", "1")

	client := NewFineractClient()
	store := &OnlineBankStore{path: filepath.Join(dir, "online-bank.json")}
	synced, err := SyncFineractToOnlineBank(store, client)
	if err != nil {
		t.Fatal(err)
	}
	if len(synced) != 1 || synced[0].ID != "fineract-1" {
		t.Fatalf("synced: %+v", synced)
	}
	if synced[0].Bank != "fineract" || synced[0].Currency != "EUR" {
		t.Fatalf("account fields: %+v", synced[0])
	}
}

func TestParseFineractOnlineBankID(t *testing.T) {
	id, ok := ParseFineractOnlineBankID("fineract-99")
	if !ok || id != 99 {
		t.Fatalf("parse: %d %v", id, ok)
	}
	if IsFineractOnlineBankAccount("nsb-usd-1") {
		t.Fatal("expected false")
	}
}

func TestFineractStatusWithoutCredentials(t *testing.T) {
	t.Setenv("ONEX_FINERACT_ENABLED", "1")
	t.Setenv("ONEX_FINERACT_USERNAME", "")
	t.Setenv("ONEX_FINERACT_PASSWORD", "")
	st := NewFineractClient().Status()
	if st["configured"] != false {
		t.Fatalf("expected not configured: %+v", st)
	}
	_ = os.Unsetenv("ONEX_FINERACT_USERNAME")
}
