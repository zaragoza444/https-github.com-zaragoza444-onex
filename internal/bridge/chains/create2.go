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

// Create2Factory is deployed on most EVM mainnets (Nick's deterministic deployer).
var Create2Factory = common.HexToAddress("0x4e59b44847b379578588920cA78FbF26c0B4956C")

// FlashCoinSalt derives a stable CREATE2 salt for cross-chain same-address deploys.
func FlashCoinSalt(tokenID string) [32]byte {
	raw := crypto.Keccak256([]byte("onex-flash-coin:" + strings.TrimSpace(tokenID)))
	var salt [32]byte
	copy(salt[:], raw)
	return salt
}

func create2SaltID(in DeployInput) string {
	if in.MirrorOriginID != "" {
		return in.MirrorOriginID
	}
	return in.TokenID
}

// FlashCoinInitCode returns creation bytecode + constructor args for CREATE2.
func FlashCoinInitCode(name, symbol string, decimals int, supply uint64, ownerHex string) ([]byte, error) {
	_, payload, err := EncodeFlashCoinDeployData(name, symbol, decimals, supply, ownerHex)
	if err != nil {
		return nil, err
	}
	dataHex, _ := payload["data"].(string)
	return common.FromHex(strings.TrimPrefix(dataHex, "0x")), nil
}

// PredictFlashCoinCreate2Address returns the same contract address on every EVM chain
// when factory, salt, owner, and supply match (like USDT-style canonical deployments).
func PredictFlashCoinCreate2Address(name, symbol string, decimals int, supply uint64, ownerHex, tokenID string) (string, error) {
	initCode, err := FlashCoinInitCode(name, symbol, decimals, supply, ownerHex)
	if err != nil {
		return "", err
	}
	salt := FlashCoinSalt(tokenID)
	addr := crypto.CreateAddress2(Create2Factory, salt, crypto.Keccak256(initCode))
	return addr.Hex(), nil
}

// DeployFlashCoinCreate2Live deploys via CREATE2 factory so the address matches PredictFlashCoinCreate2Address.
func DeployFlashCoinCreate2Live(ctx context.Context, rpcURL string, networkID uint64, name, symbol string, decimals int, supply uint64, ownerHex, tokenID, privateKeyHex string) (*LiveDeployResult, error) {
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

	initCode, err := FlashCoinInitCode(name, symbol, decimals, supply, ownerHex)
	if err != nil {
		return nil, err
	}
	salt := FlashCoinSalt(tokenID)
	predicted := crypto.CreateAddress2(Create2Factory, salt, crypto.Keccak256(initCode))

	// Already deployed at canonical address on this chain.
	code, err := client.CodeAt(ctx, predicted, nil)
	if err == nil && len(code) > 0 {
		return &LiveDeployResult{
			ContractAddress: predicted.Hex(),
			TxHash:          "",
			Creator:         from.Hex(),
		}, nil
	}

	calldata := append(salt[:], initCode...)
	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("gas price: %w", err)
	}

	chainID := big.NewInt(int64(networkID))
	tx := types.NewTransaction(nonce, Create2Factory, big.NewInt(0), 3_500_000, gasPrice, calldata)
	signed, err := types.SignTx(tx, types.NewEIP155Signer(chainID), key)
	if err != nil {
		return nil, fmt.Errorf("sign tx: %w", err)
	}
	if err := client.SendTransaction(ctx, signed); err != nil {
		return nil, fmt.Errorf("send create2 tx: %w", err)
	}

	receipt, err := waitDeployReceipt(ctx, client, signed.Hash(), 4*time.Minute)
	if err != nil {
		return nil, err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("create2 deploy tx failed")
	}

	liveAddr := predicted.Hex()
	if receipt.ContractAddress != (common.Address{}) {
		liveAddr = receipt.ContractAddress.Hex()
	}
	if ok, _ := ContractDeployed(ctx, rpcURL, liveAddr); !ok {
		return nil, fmt.Errorf("no bytecode at %s after create2 deploy", liveAddr)
	}

	return &LiveDeployResult{
		ContractAddress: liveAddr,
		TxHash:          signed.Hash().Hex(),
		Creator:         from.Hex(),
	}, nil
}
