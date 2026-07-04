package bridge

import (
	"fmt"
	"strings"

	"github.com/onex-blockchain/onex/internal/ledger"
)

func (b *Bridge) cashCodes() *ledger.CashCodeStore {
	return ledger.DefaultCashCodeStore()
}

func (b *Bridge) ensureCashCodeEscrow(currency string) error {
	if err := b.ensureOnlineBank(); err != nil {
		return err
	}
	cur := strings.ToUpper(strings.TrimSpace(currency))
	if cur == "" {
		cur = "USD"
	}
	return b.onlineBank().EnsureSystemAccount(
		ledger.CashCodeEscrowAccountID,
		"NSB Cash Code Escrow",
		cur,
		"m0",
	)
}

func (b *Bridge) CashCodeStatus() map[string]interface{} {
	st := b.cashCodes().Status()
	st["onlineBank"] = ledger.OnlineBankEnabled()
	return st
}

func (b *Bridge) ListCashCodes(accountID string) ([]ledger.CashCode, error) {
	return b.cashCodes().List(accountID)
}

func (b *Bridge) IssueCashCode(req ledger.CashCodeIssueRequest) (*ledger.CashCodeIssueResult, error) {
	if err := b.ensureCashCodeEscrow(req.Currency); err != nil {
		return nil, err
	}
	from, err := b.onlineBank().GetAccount(strings.TrimSpace(req.FromAccount))
	if err != nil {
		return nil, err
	}
	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if currency == "" {
		currency = from.Currency
	}
	if !strings.EqualFold(from.Currency, currency) {
		return nil, fmt.Errorf("currency mismatch")
	}

	res, err := b.cashCodes().Issue(req, from.Name)
	if err != nil {
		return nil, err
	}
	if req.Preview {
		return res, nil
	}

	xfer, err := b.onlineBank().Transfer(ledger.OnlineBankTransferRequest{
		FromAccount: req.FromAccount,
		ToAccount:   ledger.CashCodeEscrowAccountID,
		Amount:      req.Amount,
		Reference:   "CASHCODE-" + res.CashCode.ID,
	})
	if err != nil {
		_, _ = b.cashCodes().Cancel(res.CashCode.ID, req.FromAccount)
		return nil, err
	}
	if xfer.Transaction != nil {
		_ = b.cashCodes().MarkEscrow(res.CashCode.ID, xfer.Transaction.ID)
		res.EscrowTx = xfer.Transaction.ID
	}
	return res, nil
}

func (b *Bridge) VerifyCashCode(code, pin string) (*ledger.CashCodeVerifyResult, error) {
	return b.cashCodes().Verify(code, pin)
}

func (b *Bridge) RedeemCashCode(req ledger.CashCodeRedeemRequest) (*ledger.CashCodeRedeemResult, error) {
	if err := b.ensureCashCodeEscrow(""); err != nil {
		return nil, err
	}
	if _, err := b.onlineBank().GetAccount(strings.TrimSpace(req.ToAccount)); err != nil {
		return nil, err
	}

	verify, err := b.cashCodes().Verify(req.Code, req.PIN)
	if err != nil {
		return nil, err
	}
	if !verify.Valid {
		return nil, fmt.Errorf("invalid cash code")
	}
	if req.Preview {
		c, err := b.cashCodes().GetActive(req.Code, req.PIN)
		if err != nil {
			return nil, err
		}
		return &ledger.CashCodeRedeemResult{Status: "quoted", Preview: true, CashCode: c}, nil
	}

	xfer, err := b.onlineBank().Transfer(ledger.OnlineBankTransferRequest{
		FromAccount: ledger.CashCodeEscrowAccountID,
		ToAccount:   req.ToAccount,
		Amount:      verify.Amount,
		Reference:   "REDEEM-" + verify.CodeLast4,
	})
	if err != nil {
		return nil, err
	}

	res, err := b.cashCodes().Redeem(req)
	if err != nil {
		return nil, err
	}
	if xfer.Transaction != nil {
		_ = b.cashCodes().MarkRedeemed(res.CashCode.ID, xfer.Transaction.ID, req.ToAccount)
		res.RedeemTxID = xfer.Transaction.ID
	}
	res.ToBalance = xfer.ToBalance
	return res, nil
}

func (b *Bridge) CancelCashCode(id, issuerAccount string) (*ledger.CashCode, error) {
	cc, err := b.cashCodes().Cancel(id, issuerAccount)
	if err != nil {
		return nil, err
	}
	if cc.EscrowTxID == "" {
		return cc, nil
	}
	_, err = b.onlineBank().Transfer(ledger.OnlineBankTransferRequest{
		FromAccount: ledger.CashCodeEscrowAccountID,
		ToAccount:   cc.IssuerAccount,
		Amount:      cc.Amount,
		Reference:   "CANCEL-" + cc.ID,
	})
	return cc, err
}
