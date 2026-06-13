package chains

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

//go:embed flashcoin.json
var flashCoinArtifactJSON []byte

type flashCoinArtifact struct {
	Contract string          `json:"contract"`
	Bytecode string          `json:"bytecode"`
	ABI      json.RawMessage `json:"abi"`
}

var parsedFlashCoin *flashCoinArtifact

func loadFlashCoinArtifact() (*flashCoinArtifact, error) {
	if parsedFlashCoin != nil {
		return parsedFlashCoin, nil
	}
	var a flashCoinArtifact
	if err := json.Unmarshal(flashCoinArtifactJSON, &a); err != nil {
		return nil, err
	}
	parsedFlashCoin = &a
	return parsedFlashCoin, nil
}

// evmOwnerFromCreator maps a OneX wallet address to a deterministic EVM owner.
func evmOwnerFromCreator(creator string) string {
	raw := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(creator)), "0x")
	h := sha256.Sum256([]byte("evm-owner:" + raw))
	return "0x" + hex.EncodeToString(h[:20])
}

// EvmOwnerFromCreator maps a OneX wallet address to a deterministic EVM owner.
func EvmOwnerFromCreator(creator string) string {
	return evmOwnerFromCreator(creator)
}

// EncodeFlashCoinDeployData returns creation bytecode + ABI-encoded constructor args.
func EncodeFlashCoinDeployData(name, symbol string, decimals int, supply uint64, ownerHex string) (string, map[string]interface{}, error) {
	art, err := loadFlashCoinArtifact()
	if err != nil {
		return "", nil, err
	}
	contractABI, err := abi.JSON(strings.NewReader(string(art.ABI)))
	if err != nil {
		return "", nil, err
	}
	owner := common.HexToAddress(ownerHex)
	args, err := contractABI.Pack("", name, symbol, uint8(decimals), new(big.Int).SetUint64(supply), owner)
	if err != nil {
		return "", nil, fmt.Errorf("pack constructor: %w", err)
	}
	bytecode, err := hex.DecodeString(strings.TrimPrefix(art.Bytecode, "0x"))
	if err != nil {
		return "", nil, err
	}
	data := append(bytecode, args...)
	payload := map[string]interface{}{
		"contract":   art.Contract,
		"bytecode":   art.Bytecode,
		"abi":        json.RawMessage(art.ABI),
		"data":       "0x" + hex.EncodeToString(data),
		"method":     "eth_sendTransaction",
		"deployType": "contract_create",
	}
	return "0x" + hex.EncodeToString(data), payload, nil
}

// FlashCoinDeployExtras adds real ERC-20 contract fields to an EVM deploy payload.
func FlashCoinDeployExtras(name, symbol string, decimals int, supply uint64, creator string) (map[string]interface{}, error) {
	owner := evmOwnerFromCreator(creator)
	_, payload, err := EncodeFlashCoinDeployData(name, symbol, decimals, supply, owner)
	if err != nil {
		return nil, err
	}
	payload["owner"] = owner
	payload["standard"] = "ERC-20"
	payload["source"] = "contracts/FlashCoin.sol"
	return payload, nil
}
