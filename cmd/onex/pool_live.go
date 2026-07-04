package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/onex-blockchain/onex/internal/bridge/chains"
	"github.com/onex-blockchain/onex/internal/rpc"
)

func runMakePoolLive(args []string) {
	fs := flag.NewFlagSet("make-pool-live", flag.ExitOnError)
	configPath := fs.String("config", "configs/bscscan-1b-usdt-test.json", "pool config JSON")
	mirrorConfig := fs.String("mirror", "configs/flash-coin-mirror.json", "flash coin mirror config")
	chain := fs.String("chain", "bsc", "chain slug")
	dex := fs.String("dex", "pancake-v2", "DEX id")
	deploy := fs.Bool("deploy", true, "deploy token via CREATE2 if missing")
	_ = fs.Parse(args)

	loadFlashDeployEnv()

	key, err := chains.LoadDeployerKey()
	if err != nil {
		log.Fatal(err)
	}

	poolCfg, err := chains.LoadPoolJSON(*configPath)
	if err != nil {
		log.Fatalf("read pool config: %v", err)
	}

	mirrorRaw, err := os.ReadFile(*mirrorConfig)
	if err != nil {
		log.Fatalf("read mirror config: %v", err)
	}
	var mirror struct {
		Name     string `json:"name"`
		Symbol   string `json:"symbol"`
		Decimals int    `json:"decimals"`
		Supply   string `json:"supply"`
	}
	if err := json.Unmarshal(mirrorRaw, &mirror); err != nil {
		log.Fatal(err)
	}
	if mirror.Decimals == 0 {
		mirror.Decimals = 8
	}

	chainList, err := loadEVMChains()
	if err != nil {
		log.Fatal(err)
	}
	var ch liveChain
	for _, c := range chainList {
		if c.ID == *chain {
			ch = c
			break
		}
	}
	if ch.RPC == "" {
		log.Fatalf("unknown chain %s", *chain)
	}

	dexReg, err := loadDexEntry(*chain, *dex)
	if err != nil {
		log.Fatal(err)
	}

	tokenAddr, _ := poolCfg["flashCoinAddress"].(string)
	usdtAddr, _ := poolCfg["usdtAddress"].(string)
	if usdtAddr == "" {
		usdtAddr = dexReg.USDT
	}

	tokenAmount := fmtNum(poolCfg["tokenAmount"])
	quoteAmount := fmtNum(poolCfg["quoteAmount"])
	if tokenAmount == "" {
		tokenAmount = mirror.Supply
	}
	if quoteAmount == "" {
		quoteAmount = tokenAmount
	}

	supplyU64, err := rpc.ParseAmount(mirror.Supply)
	if err != nil {
		log.Fatalf("mirror supply: %v", err)
	}

	fmt.Printf("Making pool live on %s (%s)\n", ch.Name, *dex)
	fmt.Printf("  Token: %s\n", tokenAddr)
	fmt.Printf("  Pair:  %s / USDT\n", tokenAmount)
	fmt.Printf("  Price target: $%s per token\n", fmtNum(poolCfg["targetUsdPerToken"]))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	res, err := chains.AddLiquidityV2Live(ctx, chains.PoolLiveInput{
		RPCURL:         ch.RPC,
		NetworkID:      ch.NetworkID,
		Router:         dexReg.Router,
		TokenAddress:   tokenAddr,
		QuoteAddress:   usdtAddr,
		TokenDecimals:  mirror.Decimals,
		QuoteDecimals:  18,
		TokenAmount:    tokenAmount,
		QuoteAmount:    quoteAmount,
		PrivateKeyHex:  key,
		DeployIfNeeded: *deploy,
		TokenName:      mirror.Name + " (Wrapped)",
		TokenSymbol:    "w" + strings.ToUpper(mirror.Symbol),
		TokenSupply:    supplyU64,
		TokenID:        "FLASH",
		OwnerHex:       chains.LoadDeployerAddress(),
	})
	if err != nil {
		log.Fatalf("pool live failed: %v", err)
	}

	out := map[string]interface{}{
		"chain":          *chain,
		"dex":            *dex,
		"tokenAddress":   res.TokenAddress,
		"txHash":         res.TxHash,
		"deployTxHash":   res.DeployTxHash,
		"creator":        res.Creator,
		"tokenAmount":    tokenAmount,
		"quoteAmount":    quoteAmount,
		"explorer":       ch.Explorer,
		"explorerTxUrl":  ch.Explorer + "/tx/" + res.TxHash,
		"explorerToken":  ch.Explorer + "/token/" + res.TokenAddress,
		"status":         "live",
		"updatedAt":      time.Now().UTC().Format(time.RFC3339),
	}
	printJSON(out, 200)

	liquidityPath := filepath.Join("bsc-launcher", "data", "liquidity.json")
	_ = appendLiquidityRecord(liquidityPath, map[string]interface{}{
		"chainSlug":    *chain,
		"dexId":        *dex,
		"tokenAddress": res.TokenAddress,
		"quoteId":      "usdt",
		"txHash":       res.TxHash,
		"tokenAmount":  tokenAmount,
		"quoteAmount":  quoteAmount,
		"creator":      res.Creator,
		"createdAt":    time.Now().Unix(),
	})

	fmt.Printf("\nPool live — TX %s\n", res.TxHash)
	fmt.Printf("BSCScan token: %s\n", out["explorerToken"])
	fmt.Printf("DexScreener indexes in ~5–15 min.\n")
}

type dexEntry struct {
	Router string
	USDT   string
}

func loadDexEntry(chainSlug, dexID string) (dexEntry, error) {
	raw, err := os.ReadFile("configs/dex-registry.json")
	if err != nil {
		return dexEntry{}, err
	}
	var reg struct {
		Chains map[string]struct {
			Stablecoins []struct {
				ID      string `json:"id"`
				Address string `json:"address"`
			} `json:"stablecoins"`
			Dexes []struct {
				ID     string `json:"id"`
				Router string `json:"router"`
			} `json:"dexes"`
		} `json:"chains"`
	}
	if err := json.Unmarshal(raw, &reg); err != nil {
		return dexEntry{}, err
	}
	ch, ok := reg.Chains[chainSlug]
	if !ok {
		return dexEntry{}, fmt.Errorf("chain %s not in dex registry", chainSlug)
	}
	out := dexEntry{}
	for _, s := range ch.Stablecoins {
		if s.ID == "usdt" {
			out.USDT = s.Address
			break
		}
	}
	for _, d := range ch.Dexes {
		if d.ID == dexID {
			out.Router = d.Router
			break
		}
	}
	if out.Router == "" {
		return dexEntry{}, fmt.Errorf("dex %s not found", dexID)
	}
	return out, nil
}

func fmtNum(v interface{}) string {
	switch x := v.(type) {
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case string:
		return x
	default:
		return ""
	}
}

func appendLiquidityRecord(path string, rec map[string]interface{}) error {
	var list []map[string]interface{}
	if raw, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(raw, &list)
	}
	list = append(list, rec)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
