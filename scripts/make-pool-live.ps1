# Make BSC pool live — deploy + PancakeSwap V2 USDT liquidity
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$envFile = Join-Path $root "bsc-launcher\.env"
if (Test-Path $envFile) {
    Get-Content $envFile | ForEach-Object {
        $line = $_.Trim()
        if ($line -eq "" -or $line.StartsWith("#")) { return }
        if ($line -notmatch "=") {
            if ($line -match "^0x[a-fA-F0-9]{40}$" -and -not $env:FLASH_DEPLOYER_ADDRESS) {
                [Environment]::SetEnvironmentVariable("FLASH_DEPLOYER_ADDRESS", $line, "Process")
            }
            return
        }
        $idx = $line.IndexOf("=")
        $key = $line.Substring(0, $idx).Trim()
        $val = $line.Substring($idx + 1).Trim().Trim('"').Trim("'")
        if ($key) { [Environment]::SetEnvironmentVariable($key, $val, "Process") }
    }
}

$addr = $env:FLASH_DEPLOYER_ADDRESS
if (-not $addr) { $addr = "0x05868c29D58d1EC275Cf078356c03F79B1975600" }

$key = $env:FLASH_DEPLOYER_PRIVATE_KEY
if (-not $key) { $key = $env:BSC_DEPLOYER_PRIVATE_KEY }

$hasKey = $key -and $key -notmatch '\.\.\.' -and $key -notmatch 'YOUR' -and $key.Length -ge 66

if (-not $hasKey) {
    Write-Host ""
    Write-Host "=== Pool live via MetaMask ===" -ForegroundColor Cyan
    Write-Host "No private key in .env (correct for security)."
    Write-Host "Wallet address: $addr"
    Write-Host ""
    Write-Host "Steps:"
    Write-Host "  1. Fund wallet on BSC: BNB (gas) + USDT (liquidity)"
    Write-Host "  2. Open http://127.0.0.1:9340/"
    Write-Host "  3. Liquidity -> BSCScan `$1B USDT test"
    Write-Host "  4. Connect MetaMask with $addr"
    Write-Host "  5. Add liquidity (MetaMask V2)"
    Write-Host ""
    Write-Host "BSCScan: https://bscscan.com/address/$addr"
    Start-Process "http://127.0.0.1:9340/?view=liquidity&preset=bscscan1b"
    exit 0
}

if (-not (Test-Path "bin\onex.exe")) {
    go build -o bin/onex.exe ./cmd/onex
}

Write-Host "=== Make BSC pool live (CLI) ===" -ForegroundColor Cyan
& bin\onex.exe make-pool-live -config configs\bscscan-1b-usdt-test.json -chain bsc -dex pancake-v2 -deploy

Write-Host ""
Write-Host "Reloading mirror market data..." -ForegroundColor Cyan
try {
    $r = Invoke-RestMethod -Uri "http://127.0.0.1:9340/api/flash-mirror?reload=1" -TimeoutSec 120
    $bsc = $r.deployments | Where-Object { $_.chainId -eq 'bsc' } | Select-Object -First 1
    if ($bsc) {
        Write-Host "  Price:     `$$($bsc.priceUsd)"
        Write-Host "  Mkt cap:   `$$($bsc.marketCapUsd)"
        Write-Host "  Liquidity: `$$($bsc.liquidityUsd)"
        Write-Host "  Holders:   $($bsc.holders)"
    }
} catch {
    Write-Host "  Open http://127.0.0.1:9340/ -> Mirrors -> Reload full details"
}
