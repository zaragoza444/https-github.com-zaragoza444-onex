package chains

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/onex-blockchain/onex/internal/legacy"
)

func bridgeSenderKeyPath() string {
	return filepath.Join(legacy.HomeDir(), "evm-sender.key")
}

func loadBridgeSenderKeyFromFile() (string, error) {
	data, err := os.ReadFile(bridgeSenderKeyPath())
	if err != nil {
		return "", err
	}
	v := strings.TrimSpace(string(data))
	if v == "" {
		return "", fmt.Errorf("empty evm sender key file")
	}
	if !IsPrivateKeyHex(v) {
		return "", fmt.Errorf("invalid evm sender key file")
	}
	return strings.TrimPrefix(NormalizeHex(v), ""), nil
}

// BridgeSenderAddress returns the checksummed address for the configured sender key.
func BridgeSenderAddress() (string, error) {
	key, err := LoadBridgeSenderKey()
	if err != nil {
		return "", err
	}
	priv, err := crypto.HexToECDSA(key)
	if err != nil {
		return "", err
	}
	return FormatAddress(crypto.PubkeyToAddress(priv.PublicKey).Hex()), nil
}

// EnsureBridgeSenderKey loads an EVM sender key from env or ~/.onex/evm-sender.key,
// generating and persisting one when missing unless ONEX_EVM_NO_AUTO_KEY=1.
func EnsureBridgeSenderKey() (string, error) {
	if key, err := LoadBridgeSenderKey(); err == nil {
		return key, nil
	}
	if strings.TrimSpace(os.Getenv("ONEX_EVM_NO_AUTO_KEY")) == "1" {
		return "", fmt.Errorf("evm sender key not configured")
	}
	if key, err := loadBridgeSenderKeyFromFile(); err == nil {
		_ = os.Setenv("ONEX_EVM_SENDER_KEY", key)
		return key, nil
	}

	priv, err := crypto.GenerateKey()
	if err != nil {
		return "", err
	}
	key := NormalizeHex(fmt.Sprintf("%x", crypto.FromECDSA(priv)))
	path := bridgeSenderKeyPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(key+"\n"), 0o600); err != nil {
		return "", err
	}
	_ = os.Setenv("ONEX_EVM_SENDER_KEY", key)
	return key, nil
}
