ONEX_DEFAULT_BRIDGE_CHAIN=dbis-138
DBIS138_RPC_URL=https://rpc-core.d-bis.org
ONEX_EVM_SENDER_KEY=<64-hex-key-with-ETH-on-138>
ONEX_EVM_HOLDER=0xYourAddress
setlocal
cd /d "%~dp0.."
title Deploy OneX to DBIS Chain 138 server

set DBIS138_PUBLIC_HOST=%DBIS138_PUBLIC_HOST%
if "%DBIS138_PUBLIC_HOST%"=="" (
  echo Set DBIS138_PUBLIC_HOST to your chain 138 server IP, e.g.:
  echo   set DBIS138_PUBLIC_HOST=your.server.ip
  exit /b 1
)

if "%SSH_PASS%"=="" (
  echo Run on the server directly:
  echo   bash scripts/deploy-dbis-138.sh
  echo Or set SSH_PASS and re-run this script for remote deploy.
  exit /b 1
)

echo Deploying to ubuntu@%DBIS138_PUBLIC_HOST% ...
where plink >nul 2>&1
if errorlevel 1 (
  echo Install PuTTY plink or run deploy-dbis-138.sh on the server via SSH.
  exit /b 1
)

plink -batch -pw %SSH_PASS% ubuntu@%DBIS138_PUBLIC_HOST% "cd ~/onex && git pull && bash scripts/deploy-dbis-138.sh"
echo Wallet: http://%DBIS138_PUBLIC_HOST%:9338/wallet/#ledger
$env:ONEX_EVM_SENDER_KEY='<64-hex-private-key>'