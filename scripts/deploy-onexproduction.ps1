# DNS + HTTPS + email preflight for onexproduction.com
param(
    [string]$VpsIp = "51.75.64.28",
    [string]$Domain = "onexproduction.com"
)

$ErrorActionPreference = "Continue"
Write-Host "=== OneX go-live preflight: $Domain ===" -ForegroundColor Cyan
Write-Host ""

Write-Host "[1/4] Website DNS (A records)" -ForegroundColor Yellow
try {
    $a = @(Resolve-DnsName $Domain -Type A -ErrorAction Stop | ForEach-Object { $_.IPAddress } | Sort-Object -Unique)
    Write-Host "  $Domain -> $($a -join ', ')" -ForegroundColor Green
    if ($VpsIp -and $a -notcontains $VpsIp) {
        Write-Host "  WARN: Expected VPS IP $VpsIp" -ForegroundColor Yellow
    } elseif ($VpsIp -and $a -contains $VpsIp) {
        Write-Host "  OK: Domain points to VPS $VpsIp" -ForegroundColor Green
    }
} catch {
    Write-Host "  FAIL: No A record for $Domain" -ForegroundColor Red
    Write-Host "  Add A record pointing to $VpsIp (see deploy/dns-records-onexproduction.md)" -ForegroundColor Cyan
}

try {
    $www = @(Resolve-DnsName "www.$Domain" -Type A -ErrorAction Stop | ForEach-Object { $_.IPAddress } | Sort-Object -Unique)
    Write-Host "  www.$Domain -> $($www -join ', ')" -ForegroundColor Green
} catch {
    Write-Host "  WARN: www.$Domain not set" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "[2/4] Business email (MX)" -ForegroundColor Yellow
try {
    $mx = Resolve-DnsName $Domain -Type MX -ErrorAction Stop | Sort-Object Preference
    foreach ($r in $mx) {
        Write-Host "  MX $($r.Preference) $($r.NameExchange)" -ForegroundColor Green
    }
} catch {
    Write-Host "  WARN: No MX records yet" -ForegroundColor Yellow
    Write-Host "  See docs/BUSINESS-EMAIL.md" -ForegroundColor Cyan
}

Write-Host ""
Write-Host "[3/4] HTTPS services" -ForegroundColor Yellow
$urls = @(
    @{ Url = "https://$Domain/"; Label = "Marketing site" },
    @{ Url = "https://$Domain/contact.html"; Label = "Contact page" },
    @{ Url = "https://$Domain/wallet/"; Label = "Wallet" },
    @{ Url = "https://$Domain/bridge/production/status"; Label = "Production status" },
    @{ Url = "https://$Domain/bridge/bridge7/status"; Label = "Bridge7 status" },
    @{ Url = "https://$Domain/health"; Label = "Health" }
)
foreach ($item in $urls) {
    try {
        $r = Invoke-WebRequest -Uri $item.Url -UseBasicParsing -TimeoutSec 15
        $extra = ""
        if ($item.Label -eq "Marketing site" -and $r.Content -like "*OneX*") {
            $extra = " (OneX site OK)"
        }
        Write-Host "  OK $($r.StatusCode) $($item.Label)$extra" -ForegroundColor Green
    } catch {
        Write-Host "  FAIL $($item.Label) - $($_.Exception.Message)" -ForegroundColor Red
    }
}

Write-Host ""
Write-Host "[4/4] VPS direct (optional)" -ForegroundColor Yellow
if ($VpsIp) {
    foreach ($port in @(9338, 8545)) {
        try {
            $null = Invoke-WebRequest -Uri "http://${VpsIp}:${port}/health" -UseBasicParsing -TimeoutSec 8
            Write-Host "  OK http://${VpsIp}:${port}/health" -ForegroundColor Green
        } catch {
            Write-Host "  -- http://${VpsIp}:${port}/health not reachable" -ForegroundColor DarkGray
        }
    }
}

Write-Host ""
Write-Host "=== Next steps ===" -ForegroundColor Cyan
Write-Host "1. DNS: deploy/dns-records-onexproduction.md"
Write-Host "2. Email: docs/BUSINESS-EMAIL.md"
Write-Host "3. VPS: CERTBOT_EMAIL=hello@onexproduction.com ./scripts/deploy-onexproduction.sh"
Write-Host "4. Re-run this script after DNS propagates"
