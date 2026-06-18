package bridge

import (
	"context"

	"github.com/onex-blockchain/onex/internal/bridge/chains"
	"github.com/onex-blockchain/onex/internal/ledger"
)

func (b *Bridge) SettlementCapabilities() map[string]interface{} {
	cfg := b.resolvedLedgerConfig()
	bank := ledger.LoadBankProviderConfig()
	if bank.FilePath == "" && cfg.BankFile != "" {
		bank.FilePath = cfg.BankFile
	}
	evmSender := chains.LoadBridgeSenderKeySilent()
	onexWallet := b.WalletAddress() != ""
	caps := ledger.SettlementCapabilities(cfg, bank, evmSender, onexWallet)
	if evmSender {
		if addr, err := chains.BridgeSenderAddress(); err == nil {
			caps["evmSenderAddress"] = addr
		}
	}
	return caps
}

func (b *Bridge) SettleLedger(ctx context.Context, evmHolder string, req ledger.SettlementRequest) (*ledger.SettlementResult, error) {
	if err := b.SyncLedgerBook(ctx, evmHolder); err != nil {
		return nil, err
	}
	settle := func(rec ledger.TransferRecord, dest *ledger.ExternalDestination) (string, error) {
		if ledger.ResolveExternalRaw(transferReqFromSettlement(req)) == "" {
			return "", nil
		}
		return b.settleLedgerExternal(rec, dest)
	}
	return b.ledgerBook().Settle(req, b.ledgerPrices(), b.tokenMetaMap(), settle)
}

func (b *Bridge) ListLedgerSettlements(ctx context.Context, evmHolder string, limit int) ([]ledger.SettlementRecord, error) {
	if err := b.SyncLedgerBook(ctx, evmHolder); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 25
	}
	return b.ledgerBook().ListSettlements(limit), nil
}

func transferReqFromSettlement(req ledger.SettlementRequest) ledger.TransferRequest {
	return ledger.TransferRequest{
		FromAccount:     req.FromAccount,
		ToAccount:       req.ToAccount,
		Amount:          req.Amount,
		ConvertTo:       req.PayoutAsset,
		ExternalTo:      req.ExternalTo,
		ExternalChain:   req.ExternalChain,
		ExternalBank:    req.ExternalBank,
		BankRail:        req.BankRail,
		ExternalAddress: req.ExternalAddress,
	}
}
