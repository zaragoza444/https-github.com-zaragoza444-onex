# Push main to GitHub and Gitea automatically (no manual dual push).
param(
    [string]$Branch = "main",
    [switch]$SkipGitea,
    [switch]$SkipGitHub
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $root

. (Join-Path $root "scripts/load-remotes.ps1")

$GitHubUser = if ($env:GITHUB_USER) { $env:GITHUB_USER } else { "zaragoza444" }
$RepoName = if ($env:REPO_NAME) { $env:REPO_NAME } else { "onex" }
$GiteaHost = if ($env:GITEA_HOST) { $env:GITEA_HOST.TrimEnd("/") } else { "https://git.anakatech.llc" }
$GiteaUser = if ($env:GITEA_USER) { $env:GITEA_USER } else { "zaragoza" }
$GiteaUrl = if ($env:GITEA_URL) { $env:GITEA_URL } else { "$GiteaHost/$GiteaUser/$RepoName.git" }
$GitHubUrl = "https://github.com/${GitHubUser}/${RepoName}.git"

function Ensure-Remote($name, $url) {
    try { & git remote remove $name 2>&1 | Out-Null } catch {}
    git remote add $name $url
}

function Approve-GiteaCredential {
    param([string]$Token)
    if (-not $Token) { return $false }
    $input = @(
        "protocol=https",
        "host=$([uri]$GiteaHost).Host",
        "username=$GiteaUser",
        "password=$Token",
        ""
    ) -join "`n"
    $input | git credential approve 2>$null
    return $true
}

function Ensure-GiteaRepo {
    param([string]$Token)
    if (-not $Token) { return }
    $headers = @{
        Authorization = "token $Token"
        "Content-Type" = "application/json"
    }
    $body = @{ name = $RepoName; private = $false; auto_init = $false } | ConvertTo-Json
    try {
        Invoke-RestMethod -Method Post -Uri "$GiteaHost/api/v1/user/repos" -Headers $headers -Body $body | Out-Null
        Write-Host "Gitea repo ready: $GiteaUrl" -ForegroundColor Green
    } catch {
        $code = $_.Exception.Response.StatusCode.value__
        if ($code -in 409, 422) {
            Write-Host "Gitea repo exists: $GiteaUrl" -ForegroundColor DarkGreen
        } else {
            Write-Host "Gitea repo create skipped ($code) - push may still work if repo exists." -ForegroundColor Yellow
        }
    }
}

function Get-GhExe {
    $gh = "$env:ProgramFiles\GitHub CLI\gh.exe"
    if (Test-Path $gh) { return $gh }
    return "gh"
}

if (-not $SkipGitHub) {
    $gh = Get-GhExe
    $ghOk = $false
    try { & $gh auth status 2>&1 | Out-Null; $ghOk = ($LASTEXITCODE -eq 0) } catch { $ghOk = $false }

    if (-not $ghOk -and $env:GITHUB_TOKEN) {
        $env:GH_TOKEN = $env:GITHUB_TOKEN
        try { & $gh auth status 2>&1 | Out-Null; $ghOk = ($LASTEXITCODE -eq 0) } catch { $ghOk = $false }
    }

    if ($ghOk) {
        & $gh auth setup-git 2>$null
        $repo = "${GitHubUser}/${RepoName}"
        $exists = $false
        try { & $gh repo view $repo 2>&1 | Out-Null; $exists = ($LASTEXITCODE -eq 0) } catch { $exists = $false }
        if (-not $exists) {
            Write-Host "Creating GitHub repo $repo ..."
            & $gh repo create $repo --public --description "OneX blockchain production stack" --source . --remote github
        }
    }

    Ensure-Remote "github" $GitHubUrl
    Write-Host "Pushing $Branch -> GitHub ..."
    git push -u github $Branch
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    Write-Host "GitHub: https://github.com/${GitHubUser}/${RepoName}" -ForegroundColor Green
}

if (-not $SkipGitea) {
    $giteaToken = $env:GITEA_TOKEN
    if (-not $giteaToken) {
        Write-Host "GITEA_TOKEN not set - add to remotes.env (see remotes.env.example). Skipping Gitea." -ForegroundColor Yellow
        Write-Host "GitHub Actions will mirror to Gitea once GITEA_TOKEN is set in repo Secrets." -ForegroundColor Yellow
        exit 0
    }

    Approve-GiteaCredential -Token $giteaToken | Out-Null
    Ensure-GiteaRepo -Token $giteaToken
    Ensure-Remote "gitea" $GiteaUrl
    Write-Host "Pushing $Branch -> Gitea ..."
    git push -u gitea $Branch
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    Write-Host "Gitea: $GiteaUrl" -ForegroundColor Green
}

Write-Host "Done - pushed to all configured remotes." -ForegroundColor Cyan
