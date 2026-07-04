package bridge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/onex-blockchain/onex/internal/bridge/chains"
	"github.com/onex-blockchain/onex/internal/legacy"
	"github.com/onex-blockchain/onex/internal/ledger"
)

const (
	AssetKindWallet = "wallet"
	AssetKindBank   = "bank"
)

// ExternalAsset is a saved wallet address or bank IBAN account.
type ExternalAsset struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"` // wallet | bank
	Label     string `json:"label"`
	ChainID   string `json:"chainId,omitempty"`
	Address   string `json:"address,omitempty"`
	BankID    string `json:"bankId,omitempty"`
	Rail      string `json:"rail,omitempty"`
	IBAN      string `json:"iban,omitempty"`
	Currency  string `json:"currency,omitempty"`
	CreatedAt int64  `json:"createdAt"`
}

func (a ExternalAsset) ExternalTo() string {
	switch a.Kind {
	case AssetKindBank:
		bank := strings.TrimSpace(a.BankID)
		if bank == "" {
			bank = "generic"
		}
		rail := strings.TrimSpace(a.Rail)
		if rail == "" {
			rail = string(ledger.RailIBAN)
		}
		return fmt.Sprintf("bank:%s:%s:%s", bank, rail, strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(a.IBAN), " ", "")))
	default:
		chain := strings.TrimSpace(a.ChainID)
		if chain == "" {
			chain = "ethereum"
		}
		return chain + ":" + strings.TrimSpace(a.Address)
	}
}

type externalAssetStore struct {
	mu      sync.Mutex
	path    string
	seeded  bool
	seedDir string
}

func (b *Bridge) assets() *externalAssetStore {
	if b.assetStore == nil {
		root := b.cfg.ProjectRoot
		if root == "" {
			root = "."
		}
		b.assetStore = &externalAssetStore{
			path:    filepath.Join(legacy.HomeDir(), "external-assets.json"),
			seedDir: root,
		}
	}
	return b.assetStore
}

type assetFile struct {
	Assets []ExternalAsset `json:"assets"`
}

func (as *externalAssetStore) load() ([]ExternalAsset, error) {
	as.mu.Lock()
	defer as.mu.Unlock()
	list, err := as.readFile()
	if err != nil {
		return nil, err
	}
	if len(list) == 0 && !as.seeded {
		as.seeded = true
		if seeded := as.readSeed(); len(seeded) > 0 {
			list = seeded
			_ = as.writeFile(list)
		} else if migrated := as.migrateReceivers(); len(migrated) > 0 {
			list = migrated
			_ = as.writeFile(list)
		}
	}
	return list, nil
}

func (as *externalAssetStore) readFile() ([]ExternalAsset, error) {
	data, err := os.ReadFile(as.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var f assetFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return f.Assets, nil
}

func (as *externalAssetStore) writeFile(list []ExternalAsset) error {
	if err := os.MkdirAll(filepath.Dir(as.path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(assetFile{Assets: list}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(as.path, raw, 0o600)
}

func (as *externalAssetStore) readSeed() []ExternalAsset {
	if as.seedDir == "" {
		return nil
	}
	path := filepath.Join(as.seedDir, "configs", "external-assets.example.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var f assetFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil
	}
	return f.Assets
}

func (as *externalAssetStore) migrateReceivers() []ExternalAsset {
	path := filepath.Join(legacy.HomeDir(), "receiver-wallets.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var recv []ReceiverWallet
	if err := json.Unmarshal(data, &recv); err != nil {
		return nil
	}
	out := make([]ExternalAsset, 0, len(recv))
	for _, w := range recv {
		out = append(out, ExternalAsset{
			ID: w.ID, Kind: AssetKindWallet, Label: w.Label,
			ChainID: w.ChainID, Address: w.Address, CreatedAt: w.CreatedAt,
		})
	}
	return out
}

func (as *externalAssetStore) saveAll(list []ExternalAsset) error {
	as.mu.Lock()
	defer as.mu.Unlock()
	return as.writeFile(list)
}

// ListExternalAssets returns saved wallet and bank assets. kind may be wallet, bank, or empty for all.
func (b *Bridge) ListExternalAssets(kind string) ([]ExternalAsset, error) {
	list, err := b.assets().load()
	if err != nil {
		return nil, err
	}
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind == "" {
		return list, nil
	}
	out := make([]ExternalAsset, 0, len(list))
	for _, a := range list {
		if a.Kind == kind {
			out = append(out, a)
		}
	}
	return out, nil
}

// SaveExternalAsset validates and persists a wallet or bank IBAN asset.
func (b *Bridge) SaveExternalAsset(a ExternalAsset) (*ExternalAsset, error) {
	kind := strings.ToLower(strings.TrimSpace(a.Kind))
	if kind == "" {
		kind = AssetKindWallet
	}
	label := strings.TrimSpace(a.Label)

	switch kind {
	case AssetKindWallet:
		chainID := strings.TrimSpace(a.ChainID)
		address := chains.FormatAddress(strings.TrimSpace(a.Address))
		if chainID == "" || !chains.IsAddressHex(address) {
			return nil, errInvalidAsset
		}
		if _, err := ledger.ParseExternalDestination(chainID + ":" + address); err != nil {
			return nil, err
		}
		if label == "" {
			label = address[:6] + "…" + address[len(address)-4:]
		}
		a = ExternalAsset{Kind: AssetKindWallet, Label: label, ChainID: chainID, Address: address}
	case AssetKindBank:
		bankID := strings.TrimSpace(a.BankID)
		if bankID == "" {
			bankID = "generic"
		}
		rail := strings.ToLower(strings.TrimSpace(a.Rail))
		if rail == "" {
			rail = string(ledger.RailIBAN)
		}
		iban := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(a.IBAN), " ", ""))
		if iban == "" {
			return nil, errInvalidAsset
		}
		extTo := fmt.Sprintf("bank:%s:%s:%s", bankID, rail, iban)
		if _, err := ledger.ParseExternalDestination(extTo); err != nil {
			return nil, err
		}
		if label == "" {
			label = bankID + " · " + iban[:4] + "…" + iban[len(iban)-4:]
		}
		a = ExternalAsset{
			Kind: AssetKindBank, Label: label, BankID: bankID, Rail: rail,
			IBAN: iban, Currency: strings.ToUpper(strings.TrimSpace(a.Currency)),
		}
	default:
		return nil, errInvalidAsset
	}

	list, err := b.assets().load()
	if err != nil {
		return nil, err
	}
	extTo := a.ExternalTo()
	for _, existing := range list {
		if existing.ExternalTo() == extTo {
			return &existing, nil
		}
	}
	a.ID = newID()
	a.CreatedAt = time.Now().Unix()
	list = append(list, a)
	if err := b.assets().saveAll(list); err != nil {
		return nil, err
	}
	return &a, nil
}

var errInvalidAsset = &assetError{"valid wallet (chainId + 0x address) or bank (iban + rail) required"}

type assetError struct{ msg string }

func (e *assetError) Error() string { return e.msg }
