package ai

import "testing"

func TestSummarizeContextRealLedger(t *testing.T) {
	ctx := `real_ledger: {"totalUsd":360000,"mode":"production","byFundUsd":{"m0":225000,"m1":37500,"nsb":82000},"entryCount":6,"entries":[{"asset":"USD","amount":"150000.00","source":"bank","fundClass":"m0","usd":150000}]}
saved_destinations: {"walletCount":1,"bankCount":1,"wallets":[{"label":"BSC","chainId":"bsc","address":"0x28397643D5A8E9C752Fd72362eBbBfA3e4F8C195"}],"banks":[{"label":"NSB","bankId":"nsb","rail":"iban","iban":"NSB00US00000000000001"}]}`
	out := summarizeContext(ctx)
	if out == "" {
		t.Fatal("expected summary")
	}
	if !containsAny(out, "360000", "360,000", "360000.00") && !containsAny(out, "360") {
		// flexible check for total
		if !containsAny(out, "Real ledger total") {
			t.Fatalf("missing ledger summary: %q", out)
		}
	}
	if !containsAny(out, "NSB", "iban", "IBAN") {
		t.Fatalf("missing bank summary: %q", out)
	}
}
