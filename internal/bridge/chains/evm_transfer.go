package chains

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const erc20TransferABI = `[
  {"inputs":[{"name":"to","type":"address"},{"name":"amount","type":"uint256"}],"name":"transfer","outputs":[{"type":"bool"}],"stateMutability":"nonpayable","type":"function"}
]`

// EVMSendInput sends native or ERC-20 value on an EVM chain (BSC, Ethereum, etc.).
type EVMSendInput struct {
	RPCURL        string
	ChainID       uint64
	PrivateKeyHex string
	ToAddress     string
	Asset         string
	AmountHuman   string
	TokenDecimals int
	Contract      string // empty = native coin
}

// SendEVMTransfer broadcasts a native or ERC-20 transfer and returns the tx hash.
func SendEVMTransfer(ctx context.Context, in EVMSendInput) (string, error) {
	if in.RPCURL == "" {
		return "", fmt.Errorf("rpc url required")
	}
	if in.ChainID == 0 {
		return "", fmt.Errorf("chain id required")
	}
	to := common.HexToAddress(FormatAddress(in.ToAddress))
	if to == (common.Address{}) {
		return "", fmt.Errorf("invalid recipient address")
	}

	keyHex := strings.TrimPrefix(strings.TrimSpace(in.PrivateKeyHex), "0x")
	key, err := crypto.HexToECDSA(keyHex)
	if err != nil {
		return "", fmt.Errorf("invalid sender private key")
	}

	client, err := ethclient.DialContext(ctx, in.RPCURL)
	if err != nil {
		return "", fmt.Errorf("rpc connect: %w", err)
	}
	defer client.Close()

	pub := key.Public().(*ecdsa.PublicKey)
	from := crypto.PubkeyToAddress(*pub)

	contract := strings.TrimSpace(in.Contract)
	if contract == "" {
		amount, err := HumanToBaseUnits(in.AmountHuman, in.TokenDecimals)
		if err != nil {
			return "", err
		}
		return sendEVMTx(ctx, client, key, from, &to, amount, nil, in.ChainID, 21000)
	}

	erc20, err := abi.JSON(strings.NewReader(erc20TransferABI))
	if err != nil {
		return "", err
	}
	amount, err := HumanToBaseUnits(in.AmountHuman, in.TokenDecimals)
	if err != nil {
		return "", err
	}
	data, err := erc20.Pack("transfer", to, amount)
	if err != nil {
		return "", err
	}
	token := common.HexToAddress(FormatAddress(contract))
	gas, err := client.EstimateGas(ctx, ethereum.CallMsg{From: from, To: &token, Data: data})
	if err != nil || gas == 0 {
		gas = 120000
	}
	return sendEVMTx(ctx, client, key, from, &token, big.NewInt(0), data, in.ChainID, gas)
}

func sendEVMTx(ctx context.Context, client *ethclient.Client, key *ecdsa.PrivateKey, from common.Address, to *common.Address, value *big.Int, data []byte, chainID uint64, gasLimit uint64) (string, error) {
	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return "", err
	}
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return "", err
	}
	if value == nil {
		value = big.NewInt(0)
	}
	tx := types.NewTransaction(nonce, *to, value, gasLimit, gasPrice, data)
	signed, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(int64(chainID))), key)
	if err != nil {
		return "", err
	}
	if err := client.SendTransaction(ctx, signed); err != nil {
		return "", err
	}
	receipt, err := waitPoolReceipt(ctx, client, signed.Hash(), 3*time.Minute)
	if err != nil {
		return signed.Hash().Hex(), fmt.Errorf("tx submitted %s: %w", signed.Hash().Hex(), err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return signed.Hash().Hex(), fmt.Errorf("tx %s reverted", signed.Hash().Hex())
	}
	return signed.Hash().Hex(), nil
}
