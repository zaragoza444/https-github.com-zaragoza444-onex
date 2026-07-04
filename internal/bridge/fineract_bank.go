package bridge

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/onex-blockchain/onex/internal/ledger"
)

func (b *Bridge) fineract() *ledger.FineractClient {
	return ledger.NewFineractClient()
}

func (b *Bridge) FineractStatus() map[string]interface{} {
	return b.fineract().Status()
}

func (b *Bridge) FineractAccounts() ([]ledger.FineractSavingsAccount, error) {
	return b.fineract().ListSavingsAccounts()
}

func (b *Bridge) SyncFineractAccounts() ([]ledger.OnlineBankAccount, error) {
	if err := b.ensureOnlineBank(); err != nil {
		return nil, err
	}
	return ledger.SyncFineractToOnlineBank(b.onlineBank(), b.fineract())
}

func (b *Bridge) FineractDeposit(accountID string, amount float64, reference string, preview bool) (map[string]interface{}, error) {
	fid, ok := ledger.ParseFineractOnlineBankID(accountID)
	if !ok {
		return nil, fmt.Errorf("not a fineract account")
	}
	if preview {
		return map[string]interface{}{
			"status": "quoted", "preview": true,
			"accountId": accountID, "fineractId": fid, "amount": amount, "reference": reference,
		}, nil
	}
	res, err := b.fineract().Deposit(fid, amount, reference)
	if err != nil {
		return nil, err
	}
	_, _ = ledger.SyncFineractToOnlineBank(b.onlineBank(), b.fineract())
	return map[string]interface{}{"status": "deposited", "fineract": res}, nil
}

func (b *Bridge) FineractWithdraw(accountID string, amount float64, reference string, preview bool) (map[string]interface{}, error) {
	fid, ok := ledger.ParseFineractOnlineBankID(accountID)
	if !ok {
		return nil, fmt.Errorf("not a fineract account")
	}
	if preview {
		return map[string]interface{}{
			"status": "quoted", "preview": true,
			"accountId": accountID, "fineractId": fid, "amount": amount, "reference": reference,
		}, nil
	}
	res, err := b.fineract().Withdraw(fid, amount, reference)
	if err != nil {
		return nil, err
	}
	_, _ = ledger.SyncFineractToOnlineBank(b.onlineBank(), b.fineract())
	return map[string]interface{}{"status": "withdrawn", "fineract": res}, nil
}

func (b *Bridge) applyFineractTransfer(req ledger.OnlineBankTransferRequest, res *ledger.OnlineBankTransferResult) error {
	if req.Preview || res == nil || res.Transaction == nil {
		return nil
	}
	if !ledger.LoadFineractConfig().Configured() {
		return nil
	}
	client := b.fineract()
	amt, _ := parseAmount(req.Amount)
	ref := req.Reference
	if ref == "" {
		ref = res.Transaction.Reference
	}
	if fid, ok := ledger.ParseFineractOnlineBankID(req.FromAccount); ok {
		_, err := client.Withdraw(fid, amt, ref)
		return err
	}
	if fid, ok := ledger.ParseFineractOnlineBankID(req.ToAccount); ok {
		_, err := client.Deposit(fid, amt, ref)
		return err
	}
	return nil
}

func (b *Bridge) applyFineractDeposit(req ledger.OnlineBankDepositRequest, res *ledger.OnlineBankDepositResult) error {
	if req.Preview || res == nil {
		return nil
	}
	if !ledger.LoadFineractConfig().Configured() {
		return nil
	}
	client := b.fineract()
	fid, ok := ledger.ParseFineractOnlineBankID(req.ToAccount)
	if !ok {
		return nil
	}
	amt, _ := parseAmount(req.Amount)
	_, err := client.Deposit(fid, amt, req.Reference)
	return err
}

func parseAmount(s string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(s), 64)
}
