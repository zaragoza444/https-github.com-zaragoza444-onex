package chains

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// LiveDeployResult is a confirmed on-chain FlashCoin deployment.
type LiveDeployResult struct {
	ContractAddress string `json:"contractAddress"`
	TxHash          string `json:"txHash"`
	Creator         string `json:"creator"`
}

// DeployFlashCoinLive broadcasts FlashCoin.sol and waits for the creation receipt.
func DeployFlashCoinLive(ctx context.Context, rpcURL string, networkID uint64, name, symbol string, decimals int, supply uint64, ownerHex, privateKeyHex string) (*LiveDeployResult, error) {
	if networkID == 0 {
		return nil, fmt.Errorf("network id required")
	}
	keyHex := strings.TrimPrefix(strings.TrimSpace(privateKeyHex), "0x")
	key, err := crypto.HexToECDSA(keyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid deployer private key")
	}

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("rpc connect: %w", err)
	}
	defer client.Close()

	pub := key.Public().(*ecdsa.PublicKey)
	from := crypto.PubkeyToAddress(*pub)
	if strings.TrimSpace(ownerHex) == "" {
		ownerHex = from.Hex()
	}

	_, payload, err := EncodeFlashCoinDeployData(name, symbol, decimals, supply, ownerHex)
	if err != nil {
		return nil, err
	}
	dataHex, _ := payload["data"].(string)
	data := common.FromHex(strings.TrimPrefix(dataHex, "0x"))

	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("gas price: %w", err)
	}

	chainID := big.NewInt(int64(networkID))
	tx := types.NewContractCreation(nonce, big.NewInt(0), 3_000_000, gasPrice, data)
	signed, err := types.SignTx(tx, types.NewEIP155Signer(chainID), key)
	if err != nil {
		return nil, fmt.Errorf("sign tx: %w", err)
	}
	if err := client.SendTransaction(ctx, signed); err != nil {
		return nil, fmt.Errorf("send tx: %w", err)
	}

	receipt, err := waitDeployReceipt(ctx, client, signed.Hash(), 4*time.Minute)
	if err != nil {
		return nil, err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("deploy tx failed")
	}
	if receipt.ContractAddress == (common.Address{}) {
		return nil, fmt.Errorf("no contract address in receipt")
	}

	return &LiveDeployResult{
		ContractAddress: receipt.ContractAddress.Hex(),
		TxHash:          signed.Hash().Hex(),
		Creator:         from.Hex(),
	}, nil
}

// ContractDeployed reports whether bytecode exists at an EVM address.
func ContractDeployed(ctx context.Context, rpcURL, addressHex string) (bool, error) {
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return false, err
	}
	defer client.Close()
	addr := common.HexToAddress(addressHex)
	code, err := client.CodeAt(ctx, addr, nil)
	if err != nil {
		return false, err
	}
	return len(code) > 0, nil
}

func waitDeployReceipt(ctx context.Context, client *ethclient.Client, hash common.Hash, timeout time.Duration) (*types.Receipt, error) {
	deadline := time.Now().Add(timeout)
	for {
		receipt, err := client.TransactionReceipt(ctx, hash)
		if err == nil {
			return receipt, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for tx %s", hash.Hex())
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}
}
