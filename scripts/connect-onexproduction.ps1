# Connect GitHub/Gitea Pages wallet to the OneX Production Platform bridge.
param(
    [Parameter(Mandatory = $true)]
    [string]$ProductionUrl,
    [switch]$GitHubVariable,
    [string]$GitHubUser = "zaragoza444",
    [string]$RepoName = "onex"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$url = $ProductionUrl.Trim().TrimEnd('/')

& "$root\scripts\connect-bridge.ps1" -BridgeUrl $url -GitHubVariable:$GitHubVariable `
    -GitHubUser $GitHubUser -RepoName $RepoName

Write-Host ""
Write-Host "OneX Production Platform:"
Write-Host "  Status:  $url/bridge/production/status"
Write-Host "  Ledger:  $url/wallet/#ledger"
Write-Host "  Tokens:  $url/bridge/platform/tokens"
Write-Host ""
Write-Host "Test Pages wallet:"
Write-Host "  https://${GitHubUser}.github.io/${RepoName}/wallet/?bridge=$url"
