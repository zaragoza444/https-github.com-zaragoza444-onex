# Copy embedded wallet static files to docs/wallet/ for GitHub Pages
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$src = Join-Path $root "internal\bridge\static\wallet"
$dst = Join-Path $root "docs\wallet"
New-Item -ItemType Directory -Force -Path $dst | Out-Null
$cfg = Join-Path $dst "config.js"
$bak = $null
if (Test-Path $cfg) { $bak = Get-Content $cfg -Raw }
Copy-Item -Path (Join-Path $src "*") -Destination $dst -Recurse -Force
if ($bak) { Set-Content -Path $cfg -Value $bak -NoNewline }
Write-Host "Synced wallet UI to docs/wallet/ (preserved docs/wallet/config.js)"
