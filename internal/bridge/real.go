package bridge

import (
	"context"

	"github.com/onex-blockchain/onex/internal/ledger"
)

func (b *Bridge) resolvedLedgerConfig() ledger.Config {
	return ledger.ResolvePaths(b.ledgerConfig(), b.projectRoot())
}

func (b *Bridge) isProduction() bool {
	return b.resolvedLedgerConfig().Production()
}

// syncRealPortfolio keeps only verified on-chain balances in production mode.
func (b *Bridge) syncRealPortfolio(p *Portfolio) {
	if !b.isProduction() {
		return
	}
	onexKey := b.registry().TokenKey("onex-mainnet-1", "ONEX")
	onexBal := p.GetBalance(onexKey)
	p.Balances = map[string]string{}
	if onexBal > 0 {
		p.SetBalance(onexKey, onexBal)
	}

	cfg := b.resolvedLedgerConfig()
	holder := cfg.EVMHolder
	if holder == "" {
		return
	}
	evm, err := ledger.ReadEVMEntries(context.Background(), b.ledgerChains(), holder, b.ledgerTokens())
	if err != nil {
		return
	}
	for _, e := range evm {
		if e.TokenKey == "" || e.Atomic == "" || e.Atomic == "0" {
			continue
		}
		p.Balances[e.TokenKey] = e.Atomic
	}
}
