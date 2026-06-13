package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/onex-blockchain/onex/internal/bridge/chains"
	"github.com/onex-blockchain/onex/internal/rpc"
)

type liveChain struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	NetworkID uint64 `json:"networkId"`
	RPC       string `json:"rpc"`
	Explorer  string `json:"explorer"`
	Type      string `json:"type"`
}

type liveDeployment struct {
	ChainID           string `json:"chainId"`
	ChainName         string `json:"chainName"`
	Symbol            string `json:"symbol"`
	ContractAddress   string `json:"contractAddress"`
	PredictedAddress  string `json:"predictedAddress,omitempty"`
	TxHash            string `json:"txHash,omitempty"`
	Explorer          string `json:"explorer"`
	SupplyHuman       string `json:"supplyHuman"`
	Status            string `json:"status"`
	VerifiedOnChain   bool   `json:"verifiedOnChain"`
	DeployedAt        string `json:"deployedAt,omitempty"`
}

type liveAddressBook struct {
	Name        string           `json:"name"`
	OriginToken string           `json:"originToken"`
	Symbol      string           `json:"symbol"`
	UpdatedAt   string           `json:"updatedAt"`
	Deployments []liveDeployment `json:"deployments"`
}

func runFlashCoinDeployLive(args []string) {
	fs := flag.NewFlagSet("flash-coin-deploy-live", flag.ExitOnError)
	configPath := fs.String("config", "", "flash-coin-mirror.json path")
	outPath := fs.String("out", "", "output live addresses JSON")
	verifyOnly := fs.Bool("verify", false, "only verify on-chain bytecode (no deploy)")
	_ = fs.Parse(args)

	cfgFile := *configPath
	if cfgFile == "" {
		cfgFile = findFlashCoinConfig()
	}
	raw, err := os.ReadFile(cfgFile)
	if err != nil {
		log.Fatalf("read config: %v", err)
	}
	var cfg flashCoinMirrorConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		log.Fatalf("parse config: %v", err)
	}

	chainList, err := loadEVMChains()
	if err != nil {
		log.Fatal(err)
	}
	chainByID := make(map[string]liveChain, len(chainList))
	for _, c := range chainList {
		chainByID[c.ID] = c
	}

	out := *outPath
	if out == "" {
		out = filepath.Join(filepath.Dir(cfgFile), "flash-coin-live-addresses.json")
	}

	book := loadLiveBook(out, cfg)
	ctx := context.Background()

	if *verifyOnly {
		for i := range book.Deployments {
			ch, ok := chainByID[book.Deployments[i].ChainID]
			if !ok || ch.RPC == "" {
				continue
			}
			addr := book.Deployments[i].ContractAddress
			if addr == "" {
				continue
			}
			okChain, err := chains.ContractDeployed(ctx, ch.RPC, addr)
			if err != nil {
				log.Printf("%s verify error: %v", ch.ID, err)
				continue
			}
			book.Deployments[i].VerifiedOnChain = okChain
			if okChain {
				book.Deployments[i].Status = "live"
			}
		}
		book.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		writeLiveBook(out, book)
		printJSON(book, 200)
		return
	}

	loadFlashDeployEnv()
	key := strings.TrimPrefix(strings.TrimSpace(os.Getenv("FLASH_DEPLOYER_PRIVATE_KEY")), "0x")
	if key == "" {
		key = strings.TrimPrefix(strings.TrimSpace(os.Getenv("BSC_DEPLOYER_PRIVATE_KEY")), "0x")
	}
	if key == "" {
		log.Fatal("set FLASH_DEPLOYER_PRIVATE_KEY or BSC_DEPLOYER_PRIVATE_KEY in bsc-launcher/.env to deploy on mainnet")
	}

	wrapSupply, err := rpc.ParseAmount(cfg.WrapAmountPerChain)
	if err != nil {
		log.Fatalf("wrap amount: %v", err)
	}

	predicted := loadPredictedAddresses(filepath.Join(filepath.Dir(cfgFile), "flash-coin-mirror-result.json"))
	var failed int

	for _, chainID := range cfg.MirrorChains {
		ch, ok := chainByID[chainID]
		if !ok || ch.Type != "evm" || ch.RPC == "" {
			log.Printf("skip %s: not an EVM mainnet chain", chainID)
			continue
		}
		if existing, ok := findDeployment(book, chainID); ok && existing.VerifiedOnChain && existing.ContractAddress != "" {
			fmt.Printf("skip %s: already live at %s\n", ch.Name, existing.ContractAddress)
			continue
		}

		symbol := "w" + strings.ToUpper(cfg.Symbol)
		name := cfg.Name + " (Wrapped)"
		fmt.Printf("Deploying %s on %s...\n", symbol, ch.Name)

		res, err := chains.DeployFlashCoinCreate2Live(ctx, ch.RPC, ch.NetworkID, name, symbol, cfg.Decimals, wrapSupply, cfg.CanonicalOwner, "FLASH", key)
		if err != nil {
			log.Printf("deploy %s failed: %v", chainID, err)
			failed++
			continue
		}
		live, _ := chains.ContractDeployed(ctx, ch.RPC, res.ContractAddress)
		dep := liveDeployment{
			ChainID:          chainID,
			ChainName:        ch.Name,
			Symbol:           symbol,
			ContractAddress:  res.ContractAddress,
			PredictedAddress: predicted[chainID],
			TxHash:           res.TxHash,
			Explorer:         ch.Explorer,
			SupplyHuman:      cfg.WrapAmountPerChain,
			Status:           "live",
			VerifiedOnChain:  live,
			DeployedAt:       time.Now().UTC().Format(time.RFC3339),
		}
		book = upsertDeployment(book, dep)
		writeLiveBook(out, book)
		fmt.Printf("  %s %s tx %s\n", ch.Name, res.ContractAddress, res.TxHash)
	}

	if failed > 0 {
		log.Printf("warning: %d chain(s) failed - re-run to resume remaining chains", failed)
	}

	book.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	writeLiveBook(out, book)
	fmt.Printf("Saved live addresses: %s\n", out)
	printJSON(book, 200)
}

func loadEVMChains() ([]liveChain, error) {
	paths := []string{"configs/chains.json", filepath.Join("..", "configs", "chains.json")}
	var raw []byte
	var err error
	for _, p := range paths {
		raw, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("chains.json: %w", err)
	}
	var list []liveChain
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func loadLiveBook(path string, cfg flashCoinMirrorConfig) liveAddressBook {
	book := liveAddressBook{
		Name:        cfg.Name,
		OriginToken: strings.ToUpper(cfg.Symbol),
		Symbol:      "w" + strings.ToUpper(cfg.Symbol),
		Deployments: []liveDeployment{},
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return book
	}
	_ = json.Unmarshal(raw, &book)
	return book
}

func upsertDeployment(book liveAddressBook, dep liveDeployment) liveAddressBook {
	for i := range book.Deployments {
		if book.Deployments[i].ChainID == dep.ChainID {
			book.Deployments[i] = dep
			return book
		}
	}
	book.Deployments = append(book.Deployments, dep)
	return book
}

func findDeployment(book liveAddressBook, chainID string) (liveDeployment, bool) {
	for _, d := range book.Deployments {
		if d.ChainID == chainID {
			return d, true
		}
	}
	return liveDeployment{}, false
}

func loadPredictedAddresses(path string) map[string]string {
	out := map[string]string{}
	raw, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	var result struct {
		Steps []struct {
			Chain  string `json:"chain"`
			Result struct {
				Wrapped *struct {
					ChainID         string `json:"chainId"`
					ContractAddress string `json:"contractAddress"`
				} `json:"wrapped"`
			} `json:"result"`
		} `json:"steps"`
	}
	if json.Unmarshal(raw, &result) != nil {
		return out
	}
	for _, step := range result.Steps {
		if step.Result.Wrapped == nil {
			continue
		}
		id := step.Result.Wrapped.ChainID
		if id == "" {
			id = step.Chain
		}
		out[id] = step.Result.Wrapped.ContractAddress
	}
	return out
}

func writeLiveBook(path string, book liveAddressBook) {
	data, _ := json.MarshalIndent(book, "", "  ")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		log.Fatalf("write %s: %v", path, err)
	}
}

func loadFlashDeployEnv() {
	for _, p := range []string{
		filepath.Join("bsc-launcher", ".env"),
		filepath.Join("..", "bsc-launcher", ".env"),
	} {
		loadDotEnvFile(p)
	}
}

func loadDotEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		_ = os.Setenv(key, val)
	}
}