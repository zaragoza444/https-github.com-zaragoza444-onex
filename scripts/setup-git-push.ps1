# One-time setup: remotes, credentials, git alias, and optional auto-push hook.
param(
    [switch]$EnableAutoPush
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $root

. (Join-Path $root "scripts/load-remotes.ps1")

$example = Join-Path $root "remotes.env.example"
$envFile = Join-Path $root "remotes.env"
if (-not (Test-Path $envFile) -and (Test-Path $example)) {
    Copy-Item $example $envFile
    Write-Host "Created remotes.env from example - add GITEA_TOKEN there." -ForegroundColor Yellow
    . (Join-Path $root "scripts/load-remotes.ps1")
}

$GitHubUser = if ($env:GITHUB_USER) { $env:GITHUB_USER } else { "zaragoza444" }
$RepoName = if ($env:REPO_NAME) { $env:REPO_NAME } else { "onex" }
$GiteaUrl = if ($env:GITEA_URL) { $env:GITEA_URL } else { "https://git.anakatech.llc/zardashtways44/onex.git" }
$GitHubUrl = "https://github.com/${GitHubUser}/${RepoName}.git"

function Remove-GitRemote([string]$Name) {
    try { & git remote remove $Name 2>&1 | Out-Null } catch {}
}

Remove-GitRemote "github"
git remote add github $GitHubUrl
Remove-GitRemote "gitea"
git remote add gitea $GiteaUrl

Remove-GitRemote "origin"
git remote add origin $GitHubUrl
if ($env:GITEA_TOKEN) {
    git remote set-url --add --push origin $GitHubUrl
    git remote set-url --add --push origin $GiteaUrl
    Write-Host "origin pushes to GitHub and Gitea." -ForegroundColor Green
} else {
    git remote set-url --add --push origin $GitHubUrl
    Write-Host "origin pushes to GitHub only (set GITEA_TOKEN for dual push)." -ForegroundColor Yellow
}

git config alias.pushall "!powershell -NoProfile -ExecutionPolicy Bypass -File scripts/push-all.ps1"
git config alias.push-both "!powershell -NoProfile -ExecutionPolicy Bypass -File scripts/push-all.ps1"

$gh = "$env:ProgramFiles\GitHub CLI\gh.exe"
if (-not (Test-Path $gh)) { $gh = "gh" }
try {
    & $gh auth status 2>&1 | Out-Null
    if ($LASTEXITCODE -eq 0) {
        & $gh auth setup-git
        Write-Host "GitHub CLI git credentials configured." -ForegroundColor Green
    }
} catch {}

if ($env:GITEA_TOKEN) {
    $hostName = ([uri]$GiteaUrl).Host
    $giteaUser = if ($env:GITEA_USER) { $env:GITEA_USER } else { "zardashtways44" }
    @(
        "protocol=https",
        "host=$hostName",
        "username=$giteaUser",
        "password=$env:GITEA_TOKEN",
        ""
    ) -join "`n" | git credential approve
    Write-Host "Gitea credential stored for $giteaUser@$hostName" -ForegroundColor Green
} else {
    Write-Host "Set GITEA_TOKEN in remotes.env then re-run setup, or add GITEA_TOKEN to GitHub repo Secrets for CI mirror." -ForegroundColor Yellow
}

if ($EnableAutoPush) {
    git config core.hooksPath .githooks
    Write-Host "Auto-push hook enabled (.githooks/post-commit)." -ForegroundColor Green
}

Write-Host ""
Write-Host "Configured remotes:" -ForegroundColor Cyan
git remote -v
Write-Host ""
Write-Host "Push both remotes:  git pushall" -ForegroundColor Cyan
Write-Host "Or:                 .\scripts\push-all.ps1" -ForegroundColor Cyan
