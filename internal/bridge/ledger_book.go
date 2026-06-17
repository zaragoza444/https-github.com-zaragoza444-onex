package bridge

import (
	"context"
	"fmt"
	"strings"

	"github.com/onex-blockchain/onex/internal/ledger"
	"github.com/onex-blockchain/onex/internal/legacy"
	"github.com/onex-blockchain/onex/internal/rpc"
	"github.com/onex-blockchain/onex/internal/types"
)

func (b *Bridge) ledgerBook() *ledger.BookStore {
	if b.book == nil {
		b.book = ledger.NewBookStore(legacy.HomeDir())
	}
	return b.book
}

func (b *Bridge) SyncLedgerBook(ctx context.Context, evmHolder string) error {
	snap, err := b.ReadRealLedger(ctx, "all", evmHolder, b.LoadLatestImport())
	if err != nil {
		return err
	}
	return b.ledgerBook().SyncFromSnapshot(snap)
}

func (b *Bridge) ListLedgerAccounts(ctx context.Context, evmHolder string) ([]ledger.BookAccount, error) {
	if err := b.SyncLedgerBook(ctx, evmHolder); err != nil {
		return nil, err
	}
	return b.ledgerBook().ListAccounts()
}

func (b *Bridge) TransferLedger(ctx context.Context, evmHolder string, req ledger.TransferRequest) (*ledger.TransferResult, error) {
	if err := b.SyncLedgerBook(ctx, evmHolder); err != nil {
		return nil, err
	}
	settle := func(rec ledger.TransferRecord, dest *ledger.ExternalDestination) (string, error) {
		if ledger.ResolveExternalRaw(req) == "" {
			return "", nil
		}
		return b.settleLedgerExternal(rec, dest)
	}
	return b.ledgerBook().Transfer(req, b.ledgerPrices(), settle)
}

func (b *Bridge) settleLedgerExternal(rec ledger.TransferRecord, dest *ledger.ExternalDestination) (string, error) {
	if dest == nil {
		return "pending-settlement", nil
	}
	amtStr := rec.ToAmount
	if amtStr == "" {
		amtStr = rec.Amount
	}
	switch dest.Kind {
	case ledger.ExternalOneX:
		atomic, err := rpc.ParseAmount(amtStr)
		if err != nil {
			return "", err
		}
		fee, _ := rpc.ParseAmount("0.001")
		res, err := b.Send(types.Address(strings.ToLower(dest.Address)), atomic, fee)
		if err != nil {
			return "", err
		}
		return res["status"], nil
	case ledger.ExternalEVM, ledger.ExternalSolana, ledger.ExternalBitcoin, ledger.ExternalTron:
		return fmt.Sprintf("chain-pending:%s:%s", dest.ChainID, dest.Address), nil
	case ledger.ExternalBank:
		return ledger.InitiateBankTransfer(ledger.BankTransferRequest{
			Rail: dest.BankRail, BankName: dest.BankName, Account: dest.Address,
			Amount: amtStr, Asset: rec.Asset, Reference: rec.ID,
		})
	}
	return "pending-settlement", nil
}

func (b *Bridge) ListExternalDestinations() map[string]interface{} {
	return ledger.SupportedExternals()
}

func (b *Bridge) ImportAnyLedger(data []byte) ([]ledger.Entry, string, error) {
	entries, err := ledger.ParseAnyLedger(data)
	if err != nil {
		return nil, "", err
	}
	path, err := b.SaveLedgerImport(data)
	if err != nil {
		return nil, "", err
	}
	return entries, path, nil
}
