package chains

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const erc20ABI = `[
  {"inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"name":"approve","outputs":[{"type":"bool"}],"stateMutability":"nonpayable","type":"function"},
  {"inputs":[{"name":"owner","type":"address"},{"name":"spender","type":"address"}],"name":"allowance","outputs":[{"type":"uint256"}],"stateMutability":"view","type":"function"},
  {"inputs":[{"name":"account","type":"address"}],"name":"balanceOf","outputs":[{"type":"uint256"}],"stateMutability":"view","type":"function"}
]`

const routerV2ABI = `[
  {"inputs":[
    {"name":"tokenA","type":"address"},{"name":"tokenB","type":"address"},
    {"name":"amountADesired","type":"uint256"},{"name":"amountBDesired","type":"uint256"},
    {"name":"amountAMin","type":"uint256"},{"name":"amountBMin","type":"uint256"},
    {"name":"to","type":"address"},{"name":"deadline","type":"uint256"}
  ],"name":"addLiquidity","outputs":[
    {"name":"amountA","type":"uint256"},{"name":"amountB","type":"uint256"},{"name":"liquidity","type":"uint256"}
  ],"stateMutability":"nonpayable","type":"function"}
]`

type PoolLiveInput struct {
	RPCURL         string
	NetworkID      uint64
	Router         string
	TokenAddress   string
	QuoteAddress   string
	TokenDecimals  int
	QuoteDecimals  int
	TokenAmount    string
	QuoteAmount    string
	PrivateKeyHex  string
	DeployIfNeeded bool
	TokenName      string
	TokenSymbol    string
	TokenSupply    uint64
	TokenID        string
	OwnerHex       string
}

type PoolLiveResult struct {
	TokenAddress string `json:"tokenAddress"`
	PairAddress  string `json:"pairAddress,omitempty"`
	TxHash       string `json:"txHash"`
	DeployTxHash string `json:"deployTxHash,omitempty"`
	Creator      string `json:"creator"`
}

func HumanToBaseUnits(human string, decimals int) (*big.Int, error) {
	human = strings.ReplaceAll(strings.TrimSpace(human), ",", "")
	if human == "" {
		return nil, fmt.Errorf("empty amount")
	}
	parts := strings.SplitN(human, ".", 2)
	whole := parts[0]
	if whole == "" {
		whole = "0"
	}
	frac := ""
	if len(parts) == 2 {
		frac = parts[1]
	}
	if len(frac) > decimals {
		return nil, fmt.Errorf("too many decimal places")
	}
	for len(frac) < decimals {
		frac += "0"
	}
	raw := whole + frac
	raw = strings.TrimLeft(raw, "0")
	if raw == "" {
		raw = "0"
	}
	out := new(big.Int)
	if _, ok := out.SetString(raw, 10); !ok {
		return nil, fmt.Errorf("invalid amount %q", human)
	}
	return out, nil
}

func AddLiquidityV2Live(ctx context.Context, in PoolLiveInput) (*PoolLiveResult, error) {
	if in.RPCURL == "" || in.Router == "" || in.QuoteAddress == "" {
		return nil, fmt.Errorf("rpc, router, and quote address required")
	}
	keyHex := strings.TrimPrefix(strings.TrimSpace(in.PrivateKeyHex), "0x")
	key, err := crypto.HexToECDSA(keyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key")
	}

	client, err := ethclient.DialContext(ctx, in.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("rpc connect: %w", err)
	}
	defer client.Close()

	pub := key.Public().(*ecdsa.PublicKey)
	from := crypto.PubkeyToAddress(*pub)
	tokenAddr := strings.TrimSpace(in.TokenAddress)
	var deployTx string

	if tokenAddr != "" {
		live, err := ContractDeployed(ctx, in.RPCURL, tokenAddr)
		if err != nil {
			return nil, err
		}
		if !live && in.DeployIfNeeded {
			tokenAddr = ""
		} else if !live {
			return nil, fmt.Errorf("token %s not deployed on chain", tokenAddr)
		}
	}

	if tokenAddr == "" && in.DeployIfNeeded {
		if in.TokenName == "" || in.TokenSymbol == "" || in.TokenSupply == 0 {
			return nil, fmt.Errorf("token not deployed — provide deploy params or live address")
		}
		dep, err := DeployFlashCoinCreate2Live(ctx, in.RPCURL, in.NetworkID, in.TokenName, in.TokenSymbol, in.TokenDecimals, in.TokenSupply, in.OwnerHex, in.TokenID, in.PrivateKeyHex)
		if err != nil {
			return nil, fmt.Errorf("deploy token: %w", err)
		}
		tokenAddr = dep.ContractAddress
		deployTx = dep.TxHash
	}

	if tokenAddr == "" {
		return nil, fmt.Errorf("token address required")
	}

	tokenAmt, err := HumanToBaseUnits(in.TokenAmount, in.TokenDecimals)
	if err != nil {
		return nil, fmt.Errorf("token amount: %w", err)
	}
	quoteAmt, err := HumanToBaseUnits(in.QuoteAmount, in.QuoteDecimals)
	if err != nil {
		return nil, fmt.Errorf("quote amount: %w", err)
	}

	erc20, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return nil, err
	}
	routerParsed, err := abi.JSON(strings.NewReader(routerV2ABI))
	if err != nil {
		return nil, err
	}

	token := common.HexToAddress(tokenAddr)
	quote := common.HexToAddress(in.QuoteAddress)
	router := common.HexToAddress(in.Router)

	if err := ensureApprove(ctx, client, key, from, token, router, tokenAmt, erc20, in.NetworkID); err != nil {
		return nil, fmt.Errorf("approve token: %w", err)
	}
	if err := ensureApprove(ctx, client, key, from, quote, router, quoteAmt, erc20, in.NetworkID); err != nil {
		return nil, fmt.Errorf("approve quote: %w", err)
	}

	minA := new(big.Int).Mul(tokenAmt, big.NewInt(95))
	minA.Div(minA, big.NewInt(100))
	minB := new(big.Int).Mul(quoteAmt, big.NewInt(95))
	minB.Div(minB, big.NewInt(100))
	deadline := big.NewInt(time.Now().Add(20 * time.Minute).Unix())

	data, err := routerParsed.Pack("addLiquidity", token, quote, tokenAmt, quoteAmt, minA, minB, from, deadline)
	if err != nil {
		return nil, err
	}

	txHash, err := sendPoolTx(ctx, client, key, from, &router, big.NewInt(0), data, in.NetworkID, 800000)
	if err != nil {
		return nil, fmt.Errorf("addLiquidity: %w", err)
	}

	return &PoolLiveResult{
		TokenAddress: tokenAddr,
		TxHash:       txHash,
		DeployTxHash: deployTx,
		Creator:      from.Hex(),
	}, nil
}

func ensureApprove(ctx context.Context, client *ethclient.Client, key *ecdsa.PrivateKey, from, token, spender common.Address, amount *big.Int, erc20 abi.ABI, chainID uint64) error {
	allowData, _ := erc20.Pack("allowance", from, spender)
	out, err := client.CallContract(ctx, ethereum.CallMsg{To: &token, Data: allowData}, nil)
	if err == nil {
		vals, _ := erc20.Unpack("allowance", out)
		if len(vals) > 0 {
			if cur, ok := vals[0].(*big.Int); ok && cur.Cmp(amount) >= 0 {
				return nil
			}
		}
	}
	approveData, err := erc20.Pack("approve", spender, amount)
	if err != nil {
		return err
	}
	_, err = sendPoolTx(ctx, client, key, from, &token, big.NewInt(0), approveData, chainID, 120000)
	return err
}

func sendPoolTx(ctx context.Context, client *ethclient.Client, key *ecdsa.PrivateKey, from common.Address, to *common.Address, value *big.Int, data []byte, chainID uint64, gasLimit uint64) (string, error) {
	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return "", err
	}
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return "", err
	}
	tx := types.NewTransaction(nonce, *to, value, gasLimit, gasPrice, data)
	signed, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(int64(chainID))), key)
	if err != nil {
		return "", err
	}
	if err := client.SendTransaction(ctx, signed); err != nil {
		return "", err
	}
	receipt, err := waitPoolReceipt(ctx, client, signed.Hash(), 5*time.Minute)
	if err != nil {
		return signed.Hash().Hex(), err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return signed.Hash().Hex(), fmt.Errorf("tx %s failed", signed.Hash().Hex())
	}
	return signed.Hash().Hex(), nil
}

func waitPoolReceipt(ctx context.Context, client *ethclient.Client, hash common.Hash, timeout time.Duration) (*types.Receipt, error) {
	deadline := time.Now().Add(timeout)
	for {
		receipt, err := client.TransactionReceipt(ctx, hash)
		if err == nil {
			return receipt, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for %s", hash.Hex())
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}
}

func LoadPoolJSON(path string) (map[string]interface{}, error) {
	if path == "" {
		path = "configs/bscscan-1b-usdt-test.json"
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func ResolveABIPath(name string) (string, error) {
	for _, base := range []string{"bsc-launcher/abi", "abi"} {
		p := filepath.Join(base, name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("abi %s not found", name)
}
