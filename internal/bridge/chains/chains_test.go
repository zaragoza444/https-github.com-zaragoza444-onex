package chains_test

import (
	"strings"
	"testing"

	"github.com/onex-blockchain/onex/internal/bridge/chains"
)

func TestFlashCoinSameAddressAllChains(t *testing.T) {
	owner := "0x207df5161022381bbc6c19100ec8e99caa8f9acb"
	a1, err := chains.PredictFlashCoinCreate2Address("Flash Coin (Wrapped)", "wFLASH", 8, 10_000_000_000, owner, "WFLASH-ETH")
	if err != nil {
		t.Fatal(err)
	}
	a2, err := chains.PredictFlashCoinCreate2Address("Flash Coin (Wrapped)", "wFLASH", 8, 10_000_000_000, owner, "WFLASH-ETH")
	if err != nil {
		t.Fatal(err)
	}
	if a1 != a2 {
		t.Fatalf("same inputs must match: %s vs %s", a1, a2)
	}
	aBsc, err := chains.PredictFlashCoinCreate2Address("Flash Coin (Wrapped)", "wFLASH", 8, 10_000_000_000, owner, "WFLASH-ETH")
	if err != nil || aBsc != a1 {
		t.Fatalf("create2 address must be chain-independent: %s vs %s", a1, aBsc)
	}
}

func TestEVMDeploySameAddressMirror(t *testing.T) {
	a := &chains.EVMAdapter{}
	res, err := a.Deploy(chains.DeployInput{
		Chain: chains.DeployChain{ID: "ethereum", Name: "Ethereum", NetworkID: 1, Type: "evm"},
		Name: "Flash Coin (Wrapped)", Symbol: "wFLASH", Decimals: 8, Supply: 10_000_000_000,
		Creator: "abc123", TokenID: "WFLASH1", SameAddressMirror: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	resBsc, err := a.Deploy(chains.DeployInput{
		Chain: chains.DeployChain{ID: "bsc", Name: "BNB Chain", NetworkID: 56, Type: "evm"},
		Name: "Flash Coin (Wrapped)", Symbol: "wFLASH", Decimals: 8, Supply: 10_000_000_000,
		Creator: "abc123", TokenID: "WFLASH1", SameAddressMirror: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ContractAddress != resBsc.ContractAddress {
		t.Fatalf("expected same address on ethereum and bsc, got %s vs %s", res.ContractAddress, resBsc.ContractAddress)
	}
	if res.DeployPayload["mirrorMode"] != "create2-same-address" {
		t.Fatalf("mirrorMode %v", res.DeployPayload["mirrorMode"])
	}
}

func TestEVMDeploy(t *testing.T) {
	a := &chains.EVMAdapter{}
	res, err := a.Deploy(chains.DeployInput{
		Chain: chains.DeployChain{ID: "ethereum", Name: "Ethereum", NetworkID: 1, Type: "evm"},
		Name: "Test", Symbol: "TST", Decimals: 8, Supply: 1_000_000_000,
		Creator: "abc123", TokenID: "TST",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(res.ContractAddress, "0x") {
		t.Fatalf("expected 0x address, got %s", res.ContractAddress)
	}
	if res.DeployStatus != "ready" {
		t.Fatalf("status %s", res.DeployStatus)
	}
	if res.DeployPayload["data"] == nil {
		t.Fatal("expected deploy data for real contract")
	}
}

func TestWrapSymbol(t *testing.T) {
	a := &chains.EVMAdapter{}
	if got := a.WrapSymbol("MYC"); got != "wMYC" {
		t.Fatalf("wrap symbol %s", got)
	}
}

func TestFactory(t *testing.T) {
	for _, typ := range []string{"onex", "evm", "solana", "btc", "tron"} {
		if _, err := chains.For(typ); err != nil {
			t.Fatalf("type %s: %v", typ, err)
		}
	}
}
