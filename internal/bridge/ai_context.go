package bridge

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/onex-blockchain/onex/internal/ledger"
)

func (s *Server) buildWalletAIContext(ctx context.Context, evmHolder string) string {
	var b strings.Builder

	st, _ := s.b.Status()
	if st != nil {
		data, _ := json.Marshal(st)
		b.WriteString("bridge_status: ")
		b.Write(data)
		b.WriteByte('\n')
	}

	ledgerSt := s.b.LedgerStatus()
	if ledgerSt != nil {
		data, _ := json.Marshal(ledgerSt)
		b.WriteString("ledger_status: ")
		b.Write(data)
		b.WriteByte('\n')
	}

	if snap, err := s.b.ReadRealLedger(ctx, "all", evmHolder, s.b.LoadLatestImport()); err == nil {
		entries := make([]map[string]interface{}, 0, min(len(snap.Entries), 16))
		for i, e := range snap.Entries {
			if i >= 16 {
				break
			}
			entries = append(entries, map[string]interface{}{
				"asset": e.Asset, "amount": e.Human, "source": e.Source,
				"fundClass": e.FundClass, "account": e.Account, "usd": e.FiatUSD,
			})
		}
		data, _ := json.Marshal(map[string]interface{}{
			"totalUsd":    snap.TotalUSD,
			"mode":        snap.Mode,
			"bySourceUsd": snap.BySource,
			"byFundUsd":   snap.ByFundClass,
			"entries":     entries,
			"entryCount":  len(snap.Entries),
		})
		b.WriteString("real_ledger: ")
		b.Write(data)
		b.WriteByte('\n')
	}

	if accts, err := s.b.ListLedgerAccounts(ctx, evmHolder); err == nil && len(accts) > 0 {
		rows := make([]map[string]interface{}, 0, min(len(accts), 12))
		for i, a := range accts {
			if i >= 12 {
				break
			}
			rows = append(rows, map[string]interface{}{
				"id": a.ID, "asset": a.Asset, "balance": a.Balance,
				"source": a.Source, "account": a.Account,
			})
		}
		data, _ := json.Marshal(map[string]interface{}{
			"accounts": rows, "count": len(accts),
		})
		b.WriteString("ledger_accounts: ")
		b.Write(data)
		b.WriteByte('\n')
	}

	if assets, err := s.b.ListExternalAssets(""); err == nil && len(assets) > 0 {
		wallets := make([]map[string]string, 0)
		banks := make([]map[string]string, 0)
		for _, a := range assets {
			switch a.Kind {
			case AssetKindBank:
				banks = append(banks, map[string]string{
					"label": a.Label, "bankId": a.BankID, "rail": a.Rail,
					"iban": a.IBAN, "currency": a.Currency,
				})
			default:
				wallets = append(wallets, map[string]string{
					"label": a.Label, "chainId": a.ChainID, "address": a.Address,
				})
			}
		}
		data, _ := json.Marshal(map[string]interface{}{
			"wallets": wallets, "banks": banks,
			"walletCount": len(wallets), "bankCount": len(banks),
		})
		b.WriteString("saved_destinations: ")
		b.Write(data)
		b.WriteByte('\n')
	}

	p, err := s.b.GetPortfolio()
	if err == nil && p != nil {
		data, _ := json.Marshal(map[string]interface{}{
			"address":  p.Address,
			"balances": p.Balances,
			"stakes":   len(p.Stakes),
			"loans":    len(p.Loans),
			"nfts":     len(p.NFTs),
			"tasks":    len(p.Tasks),
		})
		b.WriteString("portfolio: ")
		b.Write(data)
		b.WriteByte('\n')
	}

	if caps := s.b.SettlementCapabilities(); caps != nil {
		data, _ := json.Marshal(caps)
		b.WriteString("settlement: ")
		b.Write(data)
		b.WriteByte('\n')
	}

	if ob := ledger.DefaultOnlineBankStore().Status(); ob != nil {
		data, _ := json.Marshal(ob)
		b.WriteString("online_bank: ")
		b.Write(data)
		b.WriteByte('\n')
	}

	_ = s.b.ensureVirtualCards()
	if vc := s.b.VirtualCardsStatus(); vc != nil {
		data, _ := json.Marshal(vc)
		b.WriteString("virtual_cards: ")
		b.Write(data)
		b.WriteByte('\n')
	}

	return b.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
