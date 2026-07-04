package ledger

import (
	"os"
	"testing"
)

func TestListHybxExchangeRoutes(t *testing.T) {
	routes := ListHybxExchangeRoutes()
	if len(routes) < 10 {
		t.Fatalf("expected at least 10 routes, got %d", len(routes))
	}
	found := false
	for _, r := range routes {
		if r.ID == "nsb-hybx" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("missing nsb-hybx route")
	}
}

func TestHybxFederateOutbound(t *testing.T) {
	os.Setenv("ONEX_HYBX_ENABLED", "1")
	os.Setenv("ONEX_ONLINE_BANK", "1")
	t.Cleanup(func() {
		os.Unsetenv("ONEX_HYBX_ENABLED")
		os.Unsetenv("ONEX_ONLINE_BANK")
	})
	ref, err := HybxFederateOutbound(BankTransferRequest{
		Rail: "swift", Account: "GB00TEST", Amount: "100.00", Asset: "USD",
	}, "TEST-REF")
	if err != nil {
		t.Fatal(err)
	}
	if ref == "" || !containsStr(ref, "hybx-fed:") {
		t.Fatalf("unexpected ref: %s", ref)
	}
	recs, err := ListHybxFederationRecords(5)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) == 0 {
		t.Fatal("expected federation record")
	}
}

func TestHybxMiddlewareStatus(t *testing.T) {
	st := HybxMiddlewareStatus()
	if st["service"] != "onex-hybx-middleware" {
		t.Fatalf("unexpected service: %v", st["service"])
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexStr(s, sub) >= 0)
}

func indexStr(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
