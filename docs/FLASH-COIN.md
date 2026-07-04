# Flash Coin — cross-chain mirror

Flash Coin (`FLASH`) is a OneX hub token mirrored as **one real contract address** on every EVM mainnet — the same model as canonical tokens like USDT and BNB (real on-chain contracts, not fake per-chain placeholders).

## How it differs from per-chain ERC-20

| Old model | New model (CREATE2) |
|-----------|---------------------|
| Different predicted address per chain | **Same `0x…` address on all EVM chains** |
| Separate ERC-20 deploy each chain | **One canonical contract** via CREATE2 factory |
| Placeholder hashes | **Real bytecode** verifiable on explorers |

Contract source: `contracts/FlashCoin.sol`  
Deploy factory: `0x4e59b44847b379578588920cA78FbF26c0B4956C` (standard CREATE2 deployer on Ethereum, BSC, Polygon, etc.)

## Architecture

```
OneX FLASH  ──wrap──►  wFLASH @ 0xSAME… on Ethereum
                    └►  wFLASH @ 0xSAME… on BSC
                    └►  wFLASH @ 0xSAME… on Polygon
                    └►  … (all 7 EVM mirrors share one address)
```

## Quick start

### 1. Build

```bat
go build -o bin/onex.exe ./cmd/onex
powershell -File scripts\compile-flashcoin.ps1
```

### 2. Generate bridge mirror (predicted same address)

Requires `onex-bridge` at http://127.0.0.1:9338:

```bat
run-onex-wallet.bat
powershell -File scripts\generate-flash-coin-mirror.ps1
```

All mirror chains in `flash-coin-mirror-result.json` will show the **same** `contractAddress`.

### 3. Live mainnet deploy (real contracts)

1. Set `FLASH_DEPLOYER_PRIVATE_KEY` in `bsc-launcher/.env`
2. Optionally set `canonicalOwner` in `configs/flash-coin-mirror.json` (defaults to deployer wallet)
3. Fund deployer with gas on all mirror chains
4. Run:

```bat
scripts\deploy-flash-coin-live.ps1
```

Deploy uses CREATE2 — first chain deploys; others reuse the same address if already live. Re-run to resume.

### 4. Dashboard

```bat
bsc-launcher\run-onex-token-lab.bat
```

http://127.0.0.1:9340/ → **Flash Coin mirrors** — **LIVE** when bytecode exists at the canonical address.

## Configuration

`configs/flash-coin-mirror.json`:

| Field | Example | Notes |
|-------|---------|-------|
| `supply` | `"1000"` | Human decimals on OneX |
| `wrapAmountPerChain` | `"100"` | Minted per live deploy |
| `mirrorMode` | `"create2-same-address"` | Same real contract all chains |
| `canonicalOwner` | `"0x…"` | Token owner (empty = deployer) |

## CLI

```bat
onex flash-coin-mirror
onex flash-coin-deploy-live
onex flash-coin-deploy-live -verify
```
