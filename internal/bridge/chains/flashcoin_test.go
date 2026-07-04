package chains_test

import (
	"strings"
	"testing"

	"github.com/onex-blockchain/onex/internal/bridge/chains"
)

func TestEncodeFlashCoinDeployData(t *testing.T) {
	_, payload, err := chains.EncodeFlashCoinDeployData(
		"Flash Coin (Wrapped)", "wFLASH", 8, 10_000_000_000,
		"0x207df5161022381bbc6c19100ec8e99caa8f9acb",
	)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := payload["data"].(string)
	if !strings.HasPrefix(data, "0x") || len(data) < 20 {
		t.Fatalf("unexpected deploy data: %s", data)
	}
	if payload["contract"] != "FlashCoin" {
		t.Fatalf("contract %v", payload["contract"])
	}
	if payload["deployType"] != "contract_create" {
		t.Fatalf("deployType %v", payload["deployType"])
	}
}

func TestFlashCoinDeployExtras(t *testing.T) {
	extras, err := chains.FlashCoinDeployExtras("Flash Coin", "FLASH", 8, 1_000_000_000, "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if extras["standard"] != "ERC-20" {
		t.Fatalf("standard %v", extras["standard"])
	}
	owner, _ := extras["owner"].(string)
	if !strings.HasPrefix(owner, "0x") || len(owner) != 42 {
		t.Fatalf("owner %s", owner)
	}
}
