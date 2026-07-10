package ledger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPaymentGatewayFeeCalculation(t *testing.T) {
	cfg := PaymentGatewayConfig{
		ProcessingFee: ProcessingFeeConfig{Enabled: true, Percent: "2", Fixed: "0.50", Currency: "USD"},
	}
	fee, enabled := cfg.ResolveFee("100.00", "USD", nil, nil)
	if !enabled || fee != "2.50" {
		t.Fatalf("fee=%s enabled=%v", fee, enabled)
	}
	disabled := false
	fee2, en2 := cfg.ResolveFee("100", "USD", nil, &disabled)
	if en2 || fee2 != "0" {
		t.Fatalf("fee=%s enabled=%v", fee2, en2)
	}
}

func TestPaymentGatewaySessionAndSettlement(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "payment-gateway.config.json")
	cfg := PaymentGatewayConfig{
		Framework:                    FrameworkNova,
		Provider:                     "mock",
		DefaultSettlementDestination: "dest1",
		AcceptedCards:                []string{CardVisa, CardMastercard, CardAmex},
		SettlementDestinations: []SettlementDestination{
			{ID: "dest1", Label: "Checking", Type: SettlementTypeInternal, AccountID: "a1", Currency: "USD", Active: true},
		},
		Pages: []PaymentPage{
			{
				Slug: "test-donate", Flow: PaymentFlowDonation, Title: "Test",
				SettlementDestination: "dest1", Currency: "USD",
				AllowCustomAmount: true, Active: true,
			},
		},
	}
	data, _ := json.Marshal(cfg)
	if err := os.WriteFile(cfgPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ONEX_PAYMENT_GATEWAY_FILE", cfgPath)
	t.Setenv("ONEX_PAYMENT_GATEWAY", "1")

	bankStore := &OnlineBankStore{path: filepath.Join(dir, "online-bank.json")}
	st := &OnlineBankState{
		Name: defaultOnlineBankName, Online: true, SWIFT: defaultOnlineBankSWIFT,
		Accounts: []OnlineBankAccount{
			{ID: "a1", Name: "Checking", Currency: "USD", Balance: "1000.00", Status: "active"},
		},
	}
	if err := bankStore.save(st); err != nil {
		t.Fatal(err)
	}

	gwStore := &PaymentGatewayStore{path: filepath.Join(dir, "payment-gateway.json")}
	_ = cfg

	sess, err := gwStore.CreateSession(CreatePaymentSessionRequest{
		PageSlug: "test-donate", Amount: "50.00", CardBrand: "visa",
		PayerEmail: "test@example.com",
	}, bankStore)
	if err != nil {
		t.Fatal(err)
	}
	if sess.Status != PaymentStatusProcessing {
		t.Fatalf("status %s", sess.Status)
	}

	confirmed, err := gwStore.ConfirmSession(ConfirmPaymentRequest{
		SessionID: sess.ID, ProviderRef: sess.ProviderRef, CardBrand: "visa",
	}, bankStore)
	if err != nil {
		t.Fatal(err)
	}
	if confirmed.Status != PaymentStatusSucceeded {
		t.Fatalf("confirmed status %s", confirmed.Status)
	}
	if confirmed.SettlementRef == "" {
		t.Fatal("missing settlement ref")
	}

	accts, _ := bankStore.ListAccounts()
	bal, _ := parseMoney(accts[0].Balance)
	if bal != 1050 {
		t.Fatalf("balance after deposit: %s", accts[0].Balance)
	}
}

func TestCardBrandAccepted(t *testing.T) {
	cfg := PaymentGatewayConfig{
		AcceptedCards: []string{CardVisa, CardMastercard, CardAmex},
	}
	if !cfg.CardAccepted("visa") || !cfg.CardAccepted("AMEX") {
		t.Fatal("expected visa and amex accepted")
	}
	if cfg.CardAccepted("discover") {
		t.Fatal("discover should not be accepted")
	}
}

func TestPaymentGatewayZBankFramework(t *testing.T) {
	cfg := PaymentGatewayConfig{Framework: FrameworkZBank}
	cfg.applyFrameworkDefaults()
	if cfg.DisplayName != "Z Bank Payment Gateway" {
		t.Fatalf("display name %s", cfg.DisplayName)
	}
}

func TestPaymentGatewayDashboardStats(t *testing.T) {
	dir := t.TempDir()
	gwStore := &PaymentGatewayStore{path: filepath.Join(dir, "payment-gateway.json")}
	st := &PaymentGatewayState{
		Sessions: []PaymentSession{
			{ID: "pay_1", Flow: PaymentFlowDonation, Amount: "50.00", Currency: "USD", ProcessingFee: "1.00", TotalCharged: "51.00", Status: PaymentStatusSucceeded, SettlementDestination: "dest1", SettlementLabel: "Checking", CreatedAt: 1000},
			{ID: "pay_2", Flow: PaymentFlowPayment, Amount: "100.00", Currency: "USD", Status: PaymentStatusFailed, CreatedAt: 2000},
			{ID: "pay_3", Flow: PaymentFlowCollection, Amount: "25.00", Currency: "EUR", Status: PaymentStatusProcessing, CreatedAt: 3000},
		},
	}
	if err := gwStore.save(st); err != nil {
		t.Fatal(err)
	}
	stats, err := gwStore.DashboardStats(10)
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalSessions != 3 {
		t.Fatalf("total=%d", stats.TotalSessions)
	}
	if stats.ByStatus[PaymentStatusSucceeded] != 1 || stats.ByStatus[PaymentStatusFailed] != 1 {
		t.Fatalf("byStatus=%v", stats.ByStatus)
	}
	if stats.VolumeByCurrency["USD"] != "50.00" {
		t.Fatalf("volume=%v", stats.VolumeByCurrency)
	}
	if stats.FeesByCurrency["USD"] != "1.00" {
		t.Fatalf("fees=%v", stats.FeesByCurrency)
	}
	if len(stats.RecentSessions) != 3 || stats.RecentSessions[0].ClientSecret != "" {
		t.Fatalf("recent=%d", len(stats.RecentSessions))
	}
	if stats.BySettlement["dest1"].Count != 1 || stats.BySettlement["dest1"].Volume != "50.00" {
		t.Fatalf("settlement=%v", stats.BySettlement)
	}
}
