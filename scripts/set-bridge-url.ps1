param(
    [Parameter(Mandatory = $true)]
    [string]$BridgeUrl
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$cfg = Join-Path $root "docs\wallet\config.js"
$url = $BridgeUrl.Trim().TrimEnd('/')
$content = @"
// Auto-generated — bridge API for GitHub Pages wallet UI
window.SHIVA_BRIDGE_URL = '$url';
"@
Set-Content -Path $cfg -Value $content -Encoding utf8
Write-Host "Updated $cfg"
Write-Host ""
Write-Host "Next:"
Write-Host "  1. GitHub repo → Settings → Secrets and variables → Actions → Variables"
Write-Host "     Name: SHIVA_BRIDGE_PUBLIC_URL  Value: $url"
Write-Host "  2. git add docs/wallet/config.js && git commit -m 'Set bridge URL' && git push"
Write-Host "  3. Re-run Actions → GitHub Pages workflow"
