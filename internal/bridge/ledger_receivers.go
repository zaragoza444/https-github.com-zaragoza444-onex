package bridge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/onex-blockchain/onex/internal/bridge/chains"
	"github.com/onex-blockchain/onex/internal/legacy"
)

// ReceiverWallet is a saved payout address for convert / settlement.
type ReceiverWallet struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	ChainID   string `json:"chainId"`
	Address   string `json:"address"`
	CreatedAt int64  `json:"createdAt"`
}

type receiverStore struct {
	mu   sync.Mutex
	path string
}

func (b *Bridge) receivers() *receiverStore {
	if b.recv == nil {
		b.recv = &receiverStore{path: filepath.Join(legacy.HomeDir(), "receiver-wallets.json")}
	}
	return b.recv
}

func (rs *receiverStore) list() ([]ReceiverWallet, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	data, err := os.ReadFile(rs.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []ReceiverWallet
	return out, json.Unmarshal(data, &out)
}

func (rs *receiverStore) saveAll(list []ReceiverWallet) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(rs.path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(rs.path, raw, 0o600)
}

func (b *Bridge) ListReceiverWallets() ([]ReceiverWallet, error) {
	return b.receivers().list()
}

func (b *Bridge) SaveReceiverWallet(label, chainID, address string) (*ReceiverWallet, error) {
	chainID = strings.TrimSpace(chainID)
	address = chains.FormatAddress(strings.TrimSpace(address))
	if chainID == "" || !chains.IsAddressHex(address) {
		return nil, errInvalidReceiver
	}
	label = strings.TrimSpace(label)
	if label == "" {
		label = address[:6] + "…" + address[len(address)-4:]
	}
	list, err := b.receivers().list()
	if err != nil {
		return nil, err
	}
	for _, w := range list {
		if strings.EqualFold(w.Address, address) && w.ChainID == chainID {
			return &w, nil
		}
	}
	w := ReceiverWallet{
		ID:        newID(),
		Label:     label,
		ChainID:   chainID,
		Address:   address,
		CreatedAt: time.Now().Unix(),
	}
	list = append(list, w)
	if err := b.receivers().saveAll(list); err != nil {
		return nil, err
	}
	return &w, nil
}

var errInvalidReceiver = &receiverError{"valid chainId and 0x address required"}

type receiverError struct{ msg string }

func (e *receiverError) Error() string { return e.msg }
