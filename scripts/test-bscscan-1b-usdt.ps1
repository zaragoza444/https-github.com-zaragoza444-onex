# BSCScan $1B market cap test — 1 billion tokens + 1 billion USDT @ $1 on BSC
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$cfgPath = Join-Path $root "configs\bscscan-1b-usdt-test.json"
$cfg = Get-Content $cfgPath -Raw | ConvertFrom-Json

$base = "http://127.0.0.1:9340"
$q = "tokenAmount=$($cfg.tokenAmount)&targetUsd=$($cfg.targetUsdPerToken)&quote=$($cfg.quote)&chain=$($cfg.chain)"

Write-Host "=== BSCScan `$1B USDT Test (BSC) ===" -ForegroundColor Cyan
Write-Host "Token:  $($cfg.tokenAmount.ToString('N0')) tokens"
Write-Host "Pair:   $($cfg.quote.ToUpper()) on $($cfg.chain)"
Write-Host "Price:  `$$($cfg.targetUsdPerToken) per token"
Write-Host "MktCap: `$$($cfg.marketCapUsd.ToString('N0'))"
Write-Host "Flash:  $($cfg.flashCoinAddress)"
Write-Host ""

try {
  $quote = Invoke-RestMethod -Uri "$base/api/liquidity/quote?$q" -TimeoutSec 15
  Write-Host "API quote OK:" -ForegroundColor Green
  Write-Host "  USDT needed: $($quote.quoteAmount)"
  Write-Host "  Market cap:  `$$($quote.marketCapUsd.ToString('N0'))"
  Write-Host "  $($quote.bscscanNote)"
} catch {
  Write-Host "API offline — using config values only" -ForegroundColor Yellow
  Write-Host "  USDT needed: $($cfg.quoteAmount.ToString('N0'))"
}

Write-Host ""
Write-Host "MetaMask steps (BSC mainnet):" -ForegroundColor Cyan
Write-Host "  1. Open http://127.0.0.1:9340/ -> Liquidity"
Write-Host "  2. Chain: BSC | DEX: PancakeSwap V2 | Quote: USDT"
Write-Host "  3. Click 'BSCScan `$1B USDT test'"
Write-Host "  4. Token: $($cfg.flashCoinAddress) (or your deployed token)"
Write-Host "  5. Add liquidity — BSCScan shows ~`$1B mcap in ~5-15 min"
Write-Host ""
Write-Host "Explorer: $($cfg.explorer)/token/$($cfg.flashCoinAddress)"
