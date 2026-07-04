# Production build for OneX Token Lab
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$build = Get-Date -Format "yyyy.MM.dd-HHmm"
$env:BSC_LAUNCHER_BUILD = $build

Write-Host "Building Token Lab (build $build)..."
go build -ldflags="-s -w" -o bin/bsc-launcher.exe ./bsc-launcher/server
go build -o bin/onex.exe ./cmd/onex
go build -o bin/onex-bridge.exe ./cmd/onex-bridge

# Stamp asset version in index.html
$index = Join-Path $root "bsc-launcher\web\index.html"
$html = Get-Content $index -Raw
$html = $html -replace 'styles\.css\?v=\d+', "styles.css?v=$build"
$html = $html -replace 'app\.js\?v=\d+', "app.js?v=$build"
Set-Content $index $html -NoNewline

Write-Host "Done. Run: bsc-launcher\run-onex-token-lab-prod.bat"
Write-Host "Docker: docker compose -f docker-compose.bsc-launcher.yml up -d --build"
