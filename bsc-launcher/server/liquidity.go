package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type DexConfig struct {
	ID            string                   `json:"id"`
	ChainSlug     string                   `json:"chainSlug"`
	ChainName     string                   `json:"chainName"`
	NetworkID     int64                    `json:"networkId"`
	Explorer      string                   `json:"explorer"`
	DexChainID    string                   `json:"dexChainId"`
	Router        string                   `json:"router"`
	Factory       string                   `json:"factory"`
	WrappedNative string                   `json:"wrappedNative"`
	USDT          string                   `json:"usdt"`
	Name          string                   `json:"name"`
	Version       int                      `json:"version"`
	Type          string                   `json:"type"`
	LiquidityMode string                   `json:"liquidityMode"`
	LiquidityURL  string                   `json:"liquidityUrl"`
	InfoURL       string                   `json:"infoUrl"`
	SwapURL       string                   `json:"swapUrl"`
	DexScreenerID string                   `json:"dexscreenerId"`
	Quotes        []QuoteToken             `json:"quotes"`
	RouterABI     []map[string]interface{} `json:"routerAbi,omitempty"`
	FactoryABI    []map[string]interface{} `json:"factoryAbi,omitempty"`
}

type QuoteToken struct {
	ID       string `json:"id"`
	Symbol   string `json:"symbol"`
	Address  string `json:"address"`
	Decimals int    `json:"decimals"`
}

type LiquidityRecord struct {
	ChainSlug    string `json:"chainSlug"`
	DexID        string `json:"dexId"`
	TokenAddress string `json:"tokenAddress"`
	QuoteID      string `json:"quoteId"`
	PairAddress  string `json:"pairAddress"`
	TokenAmount  string `json:"tokenAmount"`
	QuoteAmount  string `json:"quoteAmount"`
	TxHash       string `json:"txHash"`
	Creator      string `json:"creator"`
	CreatedAt    int64  `json:"createdAt"`
}

type LiquidityStore struct {
	mu   sync.Mutex
	path string
}

func NewLiquidityStore(dataDir string) *LiquidityStore {
	return &LiquidityStore{path: filepath.Join(dataDir, "liquidity.json")}
}

func (s *LiquidityStore) Load() ([]LiquidityRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []LiquidityRecord{}, nil
		}
		return nil, err
	}
	var list []LiquidityRecord
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func (s *LiquidityStore) Save(list []LiquidityRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}

func (s *LiquidityStore) Add(rec LiquidityRecord) error {
	list, err := s.Load()
	if err != nil {
		return err
	}
	if rec.CreatedAt == 0 {
		rec.CreatedAt = time.Now().Unix()
	}
	list = append(list, rec)
	return s.Save(list)
}

func loadABIFile(name string) ([]map[string]interface{}, error) {
	path := filepath.Join(projectRoot(), "abi", name)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var abiArr []map[string]interface{}
	if err := json.Unmarshal(data, &abiArr); err != nil {
		return nil, err
	}
	return abiArr, nil
}

func dexConfig() (DexConfig, error) {
	return DexConfig{}, fmt.Errorf("use dexConfigFor(chain, dex)")
}

func (s *Server) chainRPC(chainSlug string) (string, error) {
	chain, err := chainBySlug(chainSlug)
	if err == nil && chain.RPCURL != "" {
		return chain.RPCURL, nil
	}
	if s.dexRegistry != nil {
		cfg, e := s.dexRegistry.chainConfig(chainSlug)
		if e == nil {
			if c, e2 := chainByID(cfg.NetworkID); e2 == nil && c.RPCURL != "" {
				return c.RPCURL, nil
			}
		}
	}
	return "", fmt.Errorf("no RPC for chain %s", chainSlug)
}

func (s *Server) getPairAddress(ctx context.Context, chainSlug, dexID, tokenAddr, quoteID string) (string, error) {
	if s.dexRegistry == nil {
		return "", fmt.Errorf("dex registry unavailable")
	}
	_, dex, err := s.dexRegistry.dex(chainSlug, dexID)
	if err != nil {
		return "", err
	}
	if dex.LiquidityMode != "router-v2" || dex.Factory == "" {
		return "", nil
	}
	quote, err := s.dexRegistry.quoteToken(chainSlug, quoteID)
	if err != nil {
		return "", err
	}
	token := common.HexToAddress(tokenAddr)
	quoteAddr := common.HexToAddress(quote.Address)

	rpcURL, err := s.chainRPC(chainSlug)
	if err != nil {
		return "", err
	}
	client, err := s.rpcClient(ctx, rpcURL)
	if err != nil {
		return "", err
	}
	defer client.Close()

	factoryABIFile := dex.FactoryABI
	if factoryABIFile == "" {
		factoryABIFile = "PancakeFactory.json"
	}
	factoryABI, err := loadABIFile(factoryABIFile)
	if err != nil {
		return "", err
	}
	parsed, err := abi.JSON(strings.NewReader(mustJSON(factoryABI)))
	if err != nil {
		return "", err
	}
	data, err := parsed.Pack("getPair", token, quoteAddr)
	if err != nil {
		return "", err
	}
	factory := common.HexToAddress(dex.Factory)
	out, err := client.CallContract(ctx, ethereum.CallMsg{To: &factory, Data: data}, nil)
	if err != nil {
		return "", err
	}
	vals, err := parsed.Unpack("getPair", out)
	if err != nil || len(vals) == 0 {
		return "", fmt.Errorf("getPair failed")
	}
	pair := vals[0].(common.Address)
	if pair == (common.Address{}) {
		return "", nil
	}
	return pair.Hex(), nil
}

func mustJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
