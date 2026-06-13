package chains

import "fmt"

type EVMAdapter struct{}

func (a *EVMAdapter) Type() string { return "evm" }

func (a *EVMAdapter) WrapSymbol(originSymbol string) string {
	return "w" + originSymbol
}

func (a *EVMAdapter) Deploy(in DeployInput) (*DeployResult, error) {
	addr := evmAddress(in.Chain.ID, in.Creator, in.Symbol, in.TokenID)
	owner := evmOwnerFromCreator(in.Creator)

	extras, err := FlashCoinDeployExtras(in.Name, in.Symbol, in.Decimals, in.Supply, in.Creator)
	if err != nil {
		return nil, fmt.Errorf("flashcoin deploy: %w", err)
	}

	payload := map[string]interface{}{
		"standard":    "ERC-20",
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

	return &DeployResult{
		ContractAddress: addr,
		DeployStatus:    "ready",
		DeployTxHash:    "0x" + deterministicAddress("evm-tx", in.Chain.ID, in.Creator, in.Symbol, in.TokenID),
		DeployPayload:   payload,
		Note:            "FlashCoin.sol ERC-20 — broadcast deployPayload.data via MetaMask or eth_sendTransaction on live network.",
	}, nil
}
