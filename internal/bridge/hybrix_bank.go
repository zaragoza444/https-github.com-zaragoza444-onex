package bridge

import (
	"github.com/onex-blockchain/onex/internal/ledger"
)

func (b *Bridge) HybrixStatus() map[string]interface{} {
	return ledger.NewHybrixClient().Status()
}

func (b *Bridge) HybrixAssets() ([]string, error) {
	return ledger.NewHybrixClient().ListAssets()
}

func (b *Bridge) HybrixMirrors() ([]ledger.HybrixMirrorAccount, error) {
	return ledger.DefaultHybrixMirrorStore().ListMirrors()
}

func (b *Bridge) SyncHybrixMirrors() ([]ledger.HybrixMirrorAccount, error) {
	return ledger.SyncMirrorsFromOnlineBank(b.onlineBank())
}

func (b *Bridge) HybrixConvert(direction, nsbAccount, amount string, preview bool) (map[string]interface{}, error) {
	return ledger.HybrixConvert(b.onlineBank(), ledger.HybrixConvertRequest{
		Direction: direction, NSBAccount: nsbAccount, Amount: amount, Preview: preview,
	})
}
