package ledger

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// PaymentProviderIntent is returned when creating a payment with an acquirer.
type PaymentProviderIntent struct {
	ProviderRef  string `json:"providerRef"`
	ClientSecret string `json:"clientSecret,omitempty"`
	CheckoutURL  string `json:"checkoutUrl,omitempty"`
	Status       string `json:"status"`
}

// PaymentProvider abstracts card acquiring (Stripe, mock, future Adyen/Worldpay).
type PaymentProvider interface {
	Name() string
	CreateIntent(sess PaymentSession, cfg PaymentGatewayConfig) (*PaymentProviderIntent, error)
	ConfirmPayment(providerRef string, req ConfirmPaymentRequest) error
	VerifyWebhook(payload []byte, signature string, cfg PaymentGatewayConfig) (providerRef string, status string, err error)
}

func ResolvePaymentProvider(cfg PaymentGatewayConfig) PaymentProvider {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "stripe":
		return &stripeProvider{}
	default:
		return &mockProvider{}
	}
}

type mockProvider struct{}

func (m *mockProvider) Name() string { return "mock" }

func (m *mockProvider) CreateIntent(sess PaymentSession, _ PaymentGatewayConfig) (*PaymentProviderIntent, error) {
	ref := "mock_pi_" + sess.ID
	return &PaymentProviderIntent{
		ProviderRef:  ref,
		ClientSecret: ref + "_secret",
		Status:       PaymentStatusProcessing,
	}, nil
}

func (m *mockProvider) ConfirmPayment(providerRef string, _ ConfirmPaymentRequest) error {
	if strings.TrimSpace(providerRef) == "" {
		return fmt.Errorf("missing provider reference")
	}
	return nil
}

func (m *mockProvider) VerifyWebhook(_ []byte, _ string, _ PaymentGatewayConfig) (string, string, error) {
	return "", "", fmt.Errorf("mock provider does not use webhooks")
}

type stripeProvider struct{}

func (s *stripeProvider) Name() string { return "stripe" }

func (s *stripeProvider) CreateIntent(sess PaymentSession, cfg PaymentGatewayConfig) (*PaymentProviderIntent, error) {
	if cfg.StripeSecretKey == "" {
		return nil, fmt.Errorf("stripe secret key not configured")
	}
	amountCents, err := moneyToCents(sess.TotalCharged, sess.Currency)
	if err != nil {
		return nil, err
	}
	body := fmt.Sprintf("amount=%d&currency=%s&automatic_payment_methods[enabled]=true",
		amountCents, strings.ToLower(sess.Currency))
	if sess.PayerEmail != "" {
		body += "&receipt_email=" + urlEncode(sess.PayerEmail)
	}
	body += "&metadata[session_id]=" + urlEncode(sess.ID)
	body += "&metadata[flow]=" + urlEncode(sess.Flow)
	body += "&metadata[framework]=" + urlEncode(sess.Framework)
	body += "&metadata[destination]=" + urlEncode(sess.SettlementDestination)
	body += "&metadata[net_amount]=" + urlEncode(sess.Amount)
	body += "&metadata[processing_fee]=" + urlEncode(sess.ProcessingFee)

	req, err := http.NewRequest(http.MethodPost, "https://api.stripe.com/v1/payment_intents", strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(cfg.StripeSecretKey, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Stripe-Version", "2023-10-16")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("stripe error: %s", string(raw))
	}
	var out struct {
		ID           string `json:"id"`
		ClientSecret string `json:"client_secret"`
		Status       string `json:"status"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &PaymentProviderIntent{
		ProviderRef:  out.ID,
		ClientSecret: out.ClientSecret,
		Status:       out.Status,
	}, nil
}

func (s *stripeProvider) ConfirmPayment(providerRef string, _ ConfirmPaymentRequest) error {
	// Stripe confirms via client-side Elements or webhook; server-side confirm is a no-op when webhook fires.
	if strings.TrimSpace(providerRef) == "" {
		return fmt.Errorf("missing stripe payment intent id")
	}
	return nil
}

func (s *stripeProvider) VerifyWebhook(payload []byte, signature string, cfg PaymentGatewayConfig) (string, string, error) {
	if cfg.StripeWebhookSecret == "" {
		return "", "", fmt.Errorf("stripe webhook secret not configured")
	}
	if err := verifyStripeSignature(payload, signature, cfg.StripeWebhookSecret); err != nil {
		return "", "", err
	}
	var evt struct {
		Type string `json:"type"`
		Data struct {
			Object struct {
				ID       string `json:"id"`
				Status   string `json:"status"`
				Metadata map[string]string `json:"metadata"`
			} `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &evt); err != nil {
		return "", "", err
	}
	status := PaymentStatusProcessing
	switch evt.Type {
	case "payment_intent.succeeded":
		status = PaymentStatusSucceeded
	case "payment_intent.payment_failed":
		status = PaymentStatusFailed
	default:
		return "", "", fmt.Errorf("ignored event: %s", evt.Type)
	}
	return evt.Data.Object.ID, status, nil
}

func verifyStripeSignature(payload []byte, header, secret string) error {
	var ts int64
	var sigs []string
	for _, part := range strings.Split(header, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			ts, _ = strconv.ParseInt(kv[1], 10, 64)
		case "v1":
			sigs = append(sigs, kv[1])
		}
	}
	if ts == 0 || len(sigs) == 0 {
		return fmt.Errorf("invalid stripe signature header")
	}
	if time.Since(time.Unix(ts, 0)) > 5*time.Minute {
		return fmt.Errorf("stripe webhook timestamp too old")
	}
	signed := fmt.Sprintf("%d.%s", ts, string(payload))
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signed))
	expected := hex.EncodeToString(mac.Sum(nil))
	for _, sig := range sigs {
		if hmac.Equal([]byte(expected), []byte(sig)) {
			return nil
		}
	}
	return fmt.Errorf("stripe signature mismatch")
}

func moneyToCents(amount, currency string) (int64, error) {
	v, err := parseMoney(amount)
	if err != nil {
		return 0, err
	}
	// Zero-decimal currencies (JPY, etc.) — simplified: treat as cents for major fiat.
	_ = currency
	cents := int64(v * 100)
	if cents < 50 {
		return 0, fmt.Errorf("amount below minimum charge")
	}
	return cents, nil
}

func urlEncode(s string) string {
	return url.QueryEscape(s)
}
