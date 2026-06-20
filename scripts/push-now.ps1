# Push to GitHub and Gitea — use scripts/push-all.ps1 (wrapper for compatibility).
param(
    [string]$GitHubUser = "zaragoza444",
    [string]$RepoName = "onex",
    [string]$GiteaUrl = "https://git.anakatech.llc/zaragoza/onex.git"
)

$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
& (Join-Path $root "scripts/push-all.ps1") -Branch main
