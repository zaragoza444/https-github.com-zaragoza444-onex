package chains

import "fmt"

type EVMAdapter struct{}

func (a *EVMAdapter) Type() string { return "evm" }

func (a *EVMAdapter) WrapSymbol(originSymbol string) string {
	return "w" + originSymbol
}

func (a *EVMAdapter) Deploy(in DeployInput) (*DeployResult, error) {
	owner := evmOwnerFromCreator(in.Creator)
	var addr string
	var err error

	if in.SameAddressMirror {
		addr, err = PredictFlashCoinCreate2Address(in.Name, in.Symbol, in.Decimals, in.Supply, owner, create2SaltID(in))
		if err != nil {
			return nil, fmt.Errorf("create2 address: %w", err)
		}
	} else {
		addr = evmAddress(in.Chain.ID, in.Creator, in.Symbol, in.TokenID)
	}

	extras, err := FlashCoinDeployExtras(in.Name, in.Symbol, in.Decimals, in.Supply, in.Creator)
	if err != nil {
		return nil, fmt.Errorf("flashcoin deploy: %w", err)
	}

	standard := chainTokenStandard(in.Chain.ID)
	payload := map[string]interface{}{
		"standard":    standard,
		"chainId":     in.Chain.ID,
		"networkId":   in.Chain.NetworkID,
		"rpc":         in.Chain.RPC,
		"explorer":    in.Chain.Explorer,
		"name":        in.Name,
		"symbol":      in.Symbol,
		"decimals":    in.Decimals,
		"totalSupply": in.Supply,
		"constructor": map[string]interface{}{
			"name":          in.Name,
			"symbol":        in.Symbol,
			"decimals":      in.Decimals,
			"initialSupply": in.Supply,
			"owner":         owner,
		},
	}
	for k, v := range extras {
		payload[k] = v
	}
	if in.SameAddressMirror {
		payload["mirrorMode"] = "create2-same-address"
		payload["create2Factory"] = Create2Factory.Hex()
		payload["create2Salt"] = fmt.Sprintf("0x%x", FlashCoinSalt(create2SaltID(in)))
		payload["standard"] = "real-token"
		delete(payload, "method")
		payload["method"] = "create2_deploy"
		payload["note"] = "Same real contract address on every EVM chain (CREATE2). Not a per-chain ERC-20 deploy."
	}

	note := "FlashCoin.sol — broadcast deployPayload.data via MetaMask or eth_sendTransaction on live network."
	if in.SameAddressMirror {
		note = "CREATE2 same-address mirror — one real contract address on all EVM chains (like USDT/BNB canonical tokens)."
	}

	return &DeployResult{
		ContractAddress: addr,
		DeployStatus:    "ready",
		DeployTxHash:    "0x" + deterministicAddress("evm-tx", in.Chain.ID, in.Creator, in.Symbol, in.TokenID),
		DeployPayload:   payload,
		Note:            note,
	}, nil
}

func chainTokenStandard(chainID string) string {
	switch chainID {
	case "bsc":
		return "BEP-20"
	case "polygon", "avalanche", "arbitrum", "optimism", "base":
		return "token"
	default:
		return "token"
	}
}
