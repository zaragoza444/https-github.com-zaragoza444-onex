package bridge

// ReceiverWallet is a saved payout address for convert / settlement.
type ReceiverWallet struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	ChainID   string `json:"chainId"`
	Address   string `json:"address"`
	CreatedAt int64  `json:"createdAt"`
}

func assetToReceiver(a ExternalAsset) ReceiverWallet {
	return ReceiverWallet{
		ID: a.ID, Label: a.Label, ChainID: a.ChainID,
		Address: a.Address, CreatedAt: a.CreatedAt,
	}
}

func (b *Bridge) ListReceiverWallets() ([]ReceiverWallet, error) {
	list, err := b.ListExternalAssets(AssetKindWallet)
	if err != nil {
		return nil, err
	}
	out := make([]ReceiverWallet, len(list))
	for i, a := range list {
		out[i] = assetToReceiver(a)
	}
	return out, nil
}

func (b *Bridge) SaveReceiverWallet(label, chainID, address string) (*ReceiverWallet, error) {
	a, err := b.SaveExternalAsset(ExternalAsset{
		Kind: AssetKindWallet, Label: label, ChainID: chainID, Address: address,
	})
	if err != nil {
		return nil, err
	}
	w := assetToReceiver(*a)
	return &w, nil
}

var errInvalidReceiver = errInvalidAsset
