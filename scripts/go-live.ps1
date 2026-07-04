# Go live: push to GitHub, deploy to public VPS, enable Pages wallet.
param(
    [string]$PublicHost = "51.75.64.28",
    [string]$GitHubUser = "zaragoza444",
    [string]$RepoName = "onex",
    [switch]$SkipPush,
    [switch]$SkipDeploy
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location (Join-Path $root "..")
$bridgeUrl = "http://${PublicHost}:9338"

Write-Host "==> OneX go-live"
Write-Host "    Public bridge: $bridgeUrl"
Write-Host "    GitHub Pages:  https://${GitHubUser}.github.io/${RepoName}/wallet/"

if (-not $SkipPush) {
    Write-Host "`n==> Running tests..."
    go test ./internal/...
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

    Write-Host "`n==> Pushing to GitHub..."
    git push github main
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Push failed — commit first or fix auth."
        exit $LASTEXITCODE
    }
}

if (-not $SkipDeploy) {
    if (-not $env:SSH_PASS) {
        Write-Host "`nSSH_PASS not set — skip remote deploy."
        Write-Host "On the VPS run:"
        Write-Host "  ssh ubuntu@${PublicHost}"
        Write-Host "  cd ~/onex && git pull && bash scripts/deploy-ali-ecosystem.sh"
    } else {
        Write-Host "`n==> Deploying to ubuntu@${PublicHost}..."
        $env:SSH_HOST = $PublicHost
        $env:ALI_PUBLIC_HOST = $PublicHost
        python scripts/deploy-ali-ecosystem.py
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    }
}

$gh = "$env:ProgramFiles\GitHub CLI\gh.exe"
if (-not (Test-Path $gh)) { $gh = "gh" }
try {
    & $gh auth status 2>&1 | Out-Null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "`n==> Setting GitHub Pages bridge URL..."
        & $gh variable set ONEX_BRIDGE_PUBLIC_URL --body $bridgeUrl --repo "${GitHubUser}/${RepoName}"
        & $gh workflow run "GitHub Pages" --repo "${GitHubUser}/${RepoName}" 2>&1 | Out-Null
    }
} catch {}

Write-Host ""
Write-Host "=== LIVE URLS ==="
Write-Host "Direct wallet:  ${bridgeUrl}/wallet/"
Write-Host "Ledger:         ${bridgeUrl}/wallet/#ledger"
Write-Host "Green health:   ${bridgeUrl}/bridge/health/green"
Write-Host "GitHub Pages:   https://${GitHubUser}.github.io/${RepoName}/wallet/?bridge=$bridgeUrl"
Write-Host "Node API:       http://${PublicHost}:8545/health"
