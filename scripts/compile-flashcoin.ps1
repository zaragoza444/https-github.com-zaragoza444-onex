# Compile FlashCoin.sol and refresh embedded artifact for EVM deploy payloads.
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

Write-Host "Compiling contracts/FlashCoin.sol..."
npx --yes solc@0.8.26 --bin --abi --optimize -o contracts/out contracts/FlashCoin.sol

python -c @"
import json
bin=open('contracts/out/contracts_FlashCoin_sol_FlashCoin.bin').read().strip()
abi=json.load(open('contracts/out/contracts_FlashCoin_sol_FlashCoin.abi'))
json.dump({'contract':'FlashCoin','bytecode':bin,'abi':abi},
          open('internal/bridge/chains/flashcoin.json','w'), separators=(',',':'))
print('Wrote internal/bridge/chains/flashcoin.json')
"@

Write-Host "Done. Run: go test ./internal/bridge/chains/..."
