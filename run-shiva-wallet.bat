@echo off
setlocal
cd /d "%~dp0"
title Shiva Wallet Bridge

if not exist "bin\shivad.exe" (
  echo Building Shiva...
  call "%~dp0build-shiva.bat"
)

REM Start blockchain node if API is down
powershell -NoProfile -Command "try { (Invoke-WebRequest -Uri 'http://127.0.0.1:8545/health' -UseBasicParsing -TimeoutSec 2).StatusCode } catch { exit 1 }" >nul 2>&1
if errorlevel 1 (
  echo Starting Shiva node...
  start "Shiva Node" /MIN "bin\shivad.exe" -datadir "%~dp0data" -api :8545 -listen :30303
  timeout /t 3 >nul
)

if not exist "bin\shiva-bridge.exe" (
  echo Building bridge...
  go build -o bin\shiva-bridge.exe ./cmd/shiva-bridge
)

REM Start bridge if not running
powershell -NoProfile -Command "try { (Invoke-WebRequest -Uri 'http://127.0.0.1:9338/bridge/status' -UseBasicParsing -TimeoutSec 2).StatusCode } catch { exit 1 }" >nul 2>&1
if errorlevel 1 (
  echo Starting Shiva Wallet bridge...
  start "Shiva Bridge" /MIN "bin\shiva-bridge.exe"
  timeout /t 2 >nul
)

start http://127.0.0.1:9338/wallet/
echo Shiva Wallet opened. Bridge links your wallet to the local node.
exit /b 0
