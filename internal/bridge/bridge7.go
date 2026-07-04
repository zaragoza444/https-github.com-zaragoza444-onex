package bridge

import (
	"context"
	"encoding/json"
	"time"

	"github.com/onex-blockchain/onex/internal/ledger"
)

func (b *Bridge) Bridge7Status() map[string]interface{} {
	st := ledger.Bridge7Status()
	if b.isProduction() {
		st["production"] = true
	}
	return st
}

func (b *Bridge) SyncBridge7(ctx context.Context, evmHolder string) (map[string]interface{}, error) {
	cfg := ledger.LoadBridge7Config()
	if !cfg.Enabled {
		return nil, nil
	}
	entries, err := ledger.LoadBridge7Entries()
	if err != nil {
		return nil, err
	}
	prices := b.ledgerPrices()
	fiat := b.resolvedLedgerConfig().FiatCurrency
	valued := ledger.ValueImportEntries(entries, prices, fiat)

	payload := map[string]interface{}{
		"source": "bridge7", "mode": "production",
		"entries": entries, "importedAt": time.Now().Unix(),
	}
	raw, _ := json.MarshalIndent(payload, "", "  ")
	path := ""
	if len(raw) > 0 {
		path, _ = b.SaveLedgerImport(raw)
	}

	synced, err := b.ledgerBook().SyncImportEntries(valued)
	if err != nil {
		return nil, err
	}
	_ = b.SyncLedgerBook(ctx, evmHolder)

	importUSD, byAsset := ledger.SummarizeImport(valued)
	return map[string]interface{}{
		"status": "synced", "entries": len(valued), "accountsSynced": synced,
		"path": path, "importUsd": importUSD, "byAssetUsd": byAsset,
		"sources": ledger.SummarizeBridge7Files(cfg),
	}, nil
}
