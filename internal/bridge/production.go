package bridge

import (
	"context"
	"os"
	"strings"

	"github.com/onex-blockchain/onex/internal/ledger"
)

// ProductionPlatformStatus is the unified production bridge (ledger + token platform + node).
func (b *Bridge) ProductionPlatformStatus(ctx context.Context, evmHolder string) map[string]interface{} {
	ledgerStatus := b.LedgerStatus()
	platform, _ := b.PlatformStatus()

	var snap ledger.Snapshot
	if s, err := b.ReadRealLedger(ctx, "all", evmHolder, b.LoadLatestImport()); err == nil {
		snap = s
	}

	nodeOK := b.node != nil

	domain := strings.TrimSpace(os.Getenv("ONEX_PRODUCTION_DOMAIN"))
	if domain == "" {
		domain = "onexproduction.com"
	}
	publicHost := strings.TrimSpace(os.Getenv("ONEX_PUBLIC_HOST"))
	publicWallet := "/wallet/"
	if publicHost != "" {
		publicWallet = "http://" + publicHost + ":9338/wallet/"
	}

	return map[string]interface{}{
		"service":        "onex-production-platform",
		"mode":           "production",
		"production":     b.isProduction(),
		"domain":         domain,
		"nodeReady":      nodeOK,
		"ledger":         ledgerStatus,
		"ledgerTotalUsd": snap.TotalUSD,
		"ledgerEntries":  len(snap.Entries),
		"platform":       platform,
		"urls": map[string]string{
			"wallet":   "/wallet/",
			"ledger":   "/wallet/#ledger",
			"platform": "/wallet/#discover",
			"tokenLab": "/bridge/platform/tokens",
		},
		"public": map[string]string{
			"host":   publicHost,
			"wallet": publicWallet,
			"ledger": strings.TrimSuffix(publicWallet, "/") + "/#ledger",
			"green":  "http://" + publicHost + ":9338/bridge/health/green",
		},
		"api": map[string]string{
			"status":    "/bridge/production/status",
			"green":     "/bridge/health/green",
			"ledger":    "/bridge/ledger/real",
			"platform":  "/bridge/platform/status",
			"portfolio": "/bridge/portfolio",
			"transfer":  "/bridge/ledger/transfer",
		},
	}
}
