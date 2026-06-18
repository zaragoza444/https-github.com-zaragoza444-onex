package bridge

import (
	"context"
	"os"
	"strings"

	"github.com/onex-blockchain/onex/internal/bridge/chains"
)

func checkStatus(ok bool) string {
	if ok {
		return "green"
	}
	return "red"
}

func checkStatusSoft(ok bool) string {
	if ok {
		return "green"
	}
	return "amber"
}

// GreenHealth returns a unified all-systems status for UI and ops.
func (b *Bridge) GreenHealth(ctx context.Context, evmHolder string) map[string]interface{} {
	ledgerSt := b.LedgerStatus()
	settle := b.SettlementCapabilities()

	nodeOK := false
	if b.node != nil {
		nodeOK = b.node.Ping() == nil
	}

	bankOK := false
	if v, ok := ledgerSt["bankReady"].(bool); ok {
		bankOK = v
	}
	prodOK := false
	if v, ok := ledgerSt["production"].(bool); ok {
		prodOK = v
	}
	importOK := false
	if v, ok := ledgerSt["importActive"].(bool); ok {
		importOK = v
	}
	dbisOK := false
	if dbisRPC := strings.TrimSpace(os.Getenv("DBIS138_RPC_URL")); dbisRPC != "" {
		dbisOK = true
	} else {
		for _, c := range b.registry().GetChains() {
			if c.ID == "dbis-138" && c.RPC != "" {
				dbisOK = true
				break
			}
		}
	}
	evmSender := chains.LoadBridgeSenderKeySilent()
	evmDetail := "set ONEX_EVM_SENDER_KEY"
	if evmSender {
		if addr, err := chains.BridgeSenderAddress(); err == nil {
			evmDetail = "sender " + addr
		} else {
			evmDetail = "sender key set"
		}
	}

	snap, _ := b.ReadRealLedger(ctx, "all", evmHolder, b.LoadLatestImport())
	ledgerHasValue := snap.TotalUSD > 0 || len(snap.Entries) > 0

	checks := []map[string]interface{}{
		{"id": "bridge", "label": "Bridge API", "status": "green", "ok": true, "detail": "online"},
		{"id": "ledger", "label": "Ledger middleware", "status": checkStatus(prodOK), "ok": prodOK, "detail": "production mode"},
		{"id": "bank", "label": "Bank / fiat ledger", "status": checkStatus(bankOK), "ok": bankOK, "detail": "bank source ready"},
		{"id": "import", "label": "Active import", "status": checkStatus(importOK), "ok": importOK, "detail": "import & value enabled"},
		{"id": "settlement", "label": "Settlement", "status": "green", "ok": true, "detail": "convert → settle pipeline"},
		{"id": "dbis138", "label": "DBIS chain 138", "status": checkStatus(dbisOK), "ok": dbisOK, "detail": "IDBIS bridge configured"},
		{"id": "node", "label": "OneX node", "status": checkStatusSoft(nodeOK), "ok": nodeOK, "detail": map[bool]string{true: "synced", false: "offline (optional)"}[nodeOK]},
		{"id": "evm", "label": "EVM settlement", "status": checkStatusSoft(evmSender), "ok": evmSender, "detail": evmDetail},
		{"id": "value", "label": "Real valuation", "status": checkStatusSoft(ledgerHasValue), "ok": ledgerHasValue, "detail": map[bool]string{true: "ledger valued", false: "import or connect sources"}[ledgerHasValue]},
	}

	allGreen := true
	for _, c := range checks {
		if st, _ := c["status"].(string); st == "red" {
			allGreen = false
			break
		}
	}

	overall := "green"
	if !allGreen {
		overall = "amber"
	}

	return map[string]interface{}{
		"service":    "onex-green-health",
		"status":     overall,
		"allGreen":   allGreen,
		"checks":     checks,
		"ledgerUsd":  snap.TotalUSD,
		"settlement": settle,
		"ledger":     ledgerSt,
	}
}
