package bridge

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onex-blockchain/onex/internal/ledger"
)

func (b *Bridge) HybxMiddlewareStatus() map[string]interface{} {
	st := ledger.HybxMiddlewareStatus()
	st["virtualCards"] = b.VirtualCardsStatus()
	if b.isProduction() {
		st["production"] = true
	}
	return st
}

func (b *Bridge) HybxExchangeRoutes() []ledger.HybxExchangeRoute {
	return ledger.ListHybxExchangeRoutes()
}

func (b *Bridge) QuoteHybxExchange(req ledger.HybxExchangeRequest) (map[string]interface{}, error) {
	req.Preview = true
	return b.HybxExchange(req)
}

func (b *Bridge) HybxExchange(req ledger.HybxExchangeRequest) (map[string]interface{}, error) {
	if err := b.ensureOnlineBank(); err != nil {
		return nil, err
	}
	route := resolveBridgeHybxRoute(req)
	if req.Preview {
		return b.previewHybxExchange(req, route)
	}
	switch {
	case route.ID == "nsb-hybx" || route.ID == "hybx-nsb" || route.ID == "hybx-bank":
		return ledger.HybxExchangeBank(b.onlineBank(), req)
	case route.ID == "nsb-fineract":
		return b.hybxExchangeNSBToFineract(req)
	case route.ID == "fineract-hybx":
		return b.hybxExchangeFineractToHybx(req)
	case strings.HasPrefix(route.ID, "hybx-") && strings.HasPrefix(route.To, "chain:"):
		return b.hybxExchangeToChain(req, route)
	case route.ID == "hybx-platform":
		return b.hybxExchangeToPlatform(req)
	case route.ID == "ledger-hybx":
		return map[string]interface{}{
			"status": "quoted", "route": route.ID,
			"detail": "Use POST /bridge/ledger/settle with externalBank hybx or viaHybx",
		}, nil
	default:
		return ledger.HybxExchangeBank(b.onlineBank(), req)
	}
}

func resolveBridgeHybxRoute(req ledger.HybxExchangeRequest) ledger.HybxExchangeRoute {
	if id := strings.TrimSpace(req.Route); id != "" {
		for _, r := range ledger.ListHybxExchangeRoutes() {
			if r.ID == id {
				return r
			}
		}
	}
	from := strings.ToLower(strings.TrimSpace(req.From))
	to := strings.ToLower(strings.TrimSpace(req.To))
	if cid := strings.TrimSpace(req.ChainID); cid != "" {
		return ledger.HybxExchangeRoute{ID: "hybx-" + cid, From: "hybx", To: "chain:" + cid}
	}
	for _, r := range ledger.ListHybxExchangeRoutes() {
		if r.From == from && (r.To == to || strings.HasPrefix(to, r.To)) {
			return r
		}
	}
	return ledger.HybxExchangeRoute{ID: from + "-" + to, From: from, To: to}
}

func (b *Bridge) previewHybxExchange(req ledger.HybxExchangeRequest, route ledger.HybxExchangeRoute) (map[string]interface{}, error) {
	req.Preview = true
	out := map[string]interface{}{
		"status": "quoted", "preview": true, "route": route.ID,
		"from": route.From, "to": route.To, "amount": req.Amount,
		"symbol": req.Symbol, "nsbAccount": req.NSBAccount,
		"chainId": req.ChainID, "address": req.Address,
		"platformToken": req.PlatformToken,
	}
	if route.ID == "nsb-hybx" || route.ID == "hybx-nsb" || route.ID == "hybx-bank" {
		return ledger.HybxExchangeBank(b.onlineBank(), req)
	}
	return out, nil
}

func (b *Bridge) hybxExchangeNSBToFineract(req ledger.HybxExchangeRequest) (map[string]interface{}, error) {
	accts, err := b.fineract().ListSavingsAccounts()
	if err != nil {
		return nil, err
	}
	if len(accts) == 0 {
		return nil, fmt.Errorf("no fineract accounts — sync fineract first")
	}
	fid := accts[0].ID
	facct := ledger.FineractOnlineBankID(fid)
	if req.Preview {
		return map[string]interface{}{
			"status": "quoted", "preview": true, "route": "nsb-fineract",
			"from": req.NSBAccount, "to": facct, "amount": req.Amount,
		}, nil
	}
	res, err := b.onlineBank().Send(ledger.OnlineBankTransferRequest{
		FromAccount: req.NSBAccount, ToAccount: facct,
		Amount: req.Amount, Reference: "HYBX middleware NSB→Fineract",
	})
	if err != nil {
		return nil, err
	}
	_ = b.applyFineractTransfer(ledger.OnlineBankTransferRequest{
		FromAccount: req.NSBAccount, ToAccount: facct, Amount: req.Amount,
	}, res)
	return map[string]interface{}{
		"status": "completed", "route": "nsb-fineract", "transfer": res,
	}, nil
}

func (b *Bridge) hybxExchangeFineractToHybx(req ledger.HybxExchangeRequest) (map[string]interface{}, error) {
	if _, err := ledger.SyncFineractToOnlineBank(b.onlineBank(), b.fineract()); err != nil {
		return nil, err
	}
	mirrors, err := ledger.SyncMirrorsFromOnlineBank(b.onlineBank())
	if err != nil {
		return nil, err
	}
	if req.Preview {
		return map[string]interface{}{
			"status": "quoted", "preview": true, "route": "fineract-hybx",
			"mirrors": len(mirrors), "amount": req.Amount,
		}, nil
	}
	if req.NSBAccount == "" && len(mirrors) > 0 {
		req.NSBAccount = mirrors[0].NSBAccountID
	}
	res, err := ledger.HybrixConvert(b.onlineBank(), ledger.HybrixConvertRequest{
		Direction: "nsb-to-hybx", NSBAccount: req.NSBAccount, Amount: req.Amount,
	})
	if err != nil {
		return nil, err
	}
	res["route"] = "fineract-hybx"
	return res, nil
}

func (b *Bridge) hybxExchangeToChain(req ledger.HybxExchangeRequest, route ledger.HybxExchangeRoute) (map[string]interface{}, error) {
	chainID := strings.TrimPrefix(route.To, "chain:")
	if chainID == route.To {
		chainID = strings.TrimSpace(req.ChainID)
	}
	addr := strings.TrimSpace(req.Address)
	if chainID == "" || addr == "" {
		return nil, fmt.Errorf("chainId and address required")
	}
	if req.Preview {
		return map[string]interface{}{
			"status": "quoted", "preview": true, "route": route.ID,
			"chainId": chainID, "address": addr, "amount": req.Amount, "symbol": req.Symbol,
		}, nil
	}
	if req.NSBAccount != "" && req.Amount != "" {
		if _, err := ledger.HybrixConvert(b.onlineBank(), ledger.HybrixConvertRequest{
			Direction: "hybx-to-nsb", NSBAccount: req.NSBAccount, Amount: req.Amount,
		}); err != nil {
			return nil, err
		}
	}
	ref := fmt.Sprintf("HYBX-CHAIN-%d", time.Now().Unix())
	settle, err := ledger.HybxFederateChain(chainID, addr, req.Amount, req.Symbol, ref)
	if err != nil {
		return nil, err
	}
	evmRef := settle
	if strings.HasPrefix(settle, "hybx-chain:") {
		rec := ledger.TransferRecord{
			ID: ref, Asset: strings.ToUpper(req.Symbol), Amount: req.Amount, ToAmount: req.Amount,
		}
		dest := &ledger.ExternalDestination{Kind: ledger.ExternalEVM, ChainID: chainID, Address: addr}
		if live, err := b.settleLedgerEVM(rec, dest); err == nil && !strings.HasPrefix(live, "chain-pending:") {
			evmRef = live
		}
	}
	return map[string]interface{}{
		"status": "submitted", "route": route.ID, "hybxSettlement": settle,
		"chainSettlement": evmRef, "chainId": chainID, "address": addr,
	}, nil
}

func (b *Bridge) hybxExchangeToPlatform(req ledger.HybxExchangeRequest) (map[string]interface{}, error) {
	platform, _ := b.PlatformStatus()
	if req.Preview {
		return map[string]interface{}{
			"status": "quoted", "preview": true, "route": "hybx-platform",
			"platformTokens": platform.TotalTokens, "token": req.PlatformToken,
			"amount": req.Amount,
		}, nil
	}
	if req.NSBAccount != "" && req.Amount != "" {
		if _, err := ledger.HybrixConvert(b.onlineBank(), ledger.HybrixConvertRequest{
			Direction: "nsb-to-hybx", NSBAccount: req.NSBAccount, Amount: req.Amount,
		}); err != nil {
			return nil, err
		}
	}
	return map[string]interface{}{
		"status": "completed", "route": "hybx-platform",
		"platform": platform, "token": req.PlatformToken,
		"detail": "HYBX mirror funded — deploy via /bridge/platform/deploy",
	}, nil
}

func (b *Bridge) HybxSettle(req ledger.HybxExchangeRequest) (map[string]interface{}, error) {
	req.From = "hybx"
	if req.ChainID != "" {
		req.To = "chain:" + req.ChainID
		req.Route = "hybx-" + req.ChainID
	} else if req.BankAccount != "" {
		req.To = "bank"
		req.Route = "hybx-bank"
	}
	return b.HybxExchange(req)
}

func (b *Bridge) applyHybrixTransfer(req ledger.OnlineBankTransferRequest, res *ledger.OnlineBankTransferResult) error {
	if req.Preview || res == nil || res.Transaction == nil {
		return nil
	}
	cfg := ledger.LoadHybrixConfig()
	if !cfg.Enabled {
		return nil
	}
	destBank := strings.ToLower(strings.TrimSpace(req.ToBank))
	if destBank != "hybx" && destBank != "hybrix" {
		return nil
	}
	ref := res.Transaction.Reference
	if ref == "" {
		ref = res.Transaction.ID
	}
	acct := strings.TrimSpace(req.ToIBAN)
	if acct == "" {
		acct = strings.TrimSpace(req.ToAccount)
	}
	_, err := ledger.HybxFederateOutbound(ledger.BankTransferRequest{
		Rail: ledger.BankRail(req.Rail), BankName: "hybx", Account: acct,
		Amount: req.Amount, Asset: "usd", Reference: ref,
	}, ref)
	return err
}

func (b *Bridge) settleViaHybx(rec ledger.TransferRecord, dest *ledger.ExternalDestination) (string, bool) {
	if !ledger.LoadHybrixConfig().Enabled || dest == nil {
		return "", false
	}
	ref := rec.ID
	switch dest.Kind {
	case ledger.ExternalBank:
		if dest.BankName != "hybx" && dest.BankName != "hybrix" {
			return "", false
		}
		settle, err := ledger.HybxFederateOutbound(ledger.BankTransferRequest{
			Rail: dest.BankRail, BankName: dest.BankName, Account: dest.Address,
			Amount: rec.ToAmount, Asset: rec.Asset, Reference: ref,
		}, ref)
		return settle, err == nil
	case ledger.ExternalEVM, ledger.ExternalSolana, ledger.ExternalBitcoin, ledger.ExternalTron:
		amt := rec.ToAmount
		if amt == "" {
			amt = rec.Amount
		}
		settle, err := ledger.HybxFederateChain(dest.ChainID, dest.Address, amt, rec.Asset, ref)
		return settle, err == nil
	default:
		return "", false
	}
}

func (b *Bridge) SyncHybxMiddleware(ctx context.Context, evmHolder string) error {
	if err := b.ensureOnlineBank(); err != nil {
		return err
	}
	if ledger.LoadHybrixConfig().Enabled {
		_, _ = ledger.SyncMirrorsFromOnlineBank(b.onlineBank())
		_ = b.ensureVirtualCards()
	}
	if ledger.LoadFineractConfig().Configured() {
		_, _ = ledger.SyncFineractToOnlineBank(b.onlineBank(), b.fineract())
	}
	return b.SyncLedgerBook(ctx, evmHolder)
}
