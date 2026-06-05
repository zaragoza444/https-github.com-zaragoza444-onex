@echo off
setlocal EnableDelayedExpansion
cd /d "%~dp0"
title BSC Token Launcher

echo Building BSC Token Launcher...
go build -o bin\bsc-launcher.exe ./bsc-launcher/server
if errorlevel 1 (
  echo Build failed. Install Go from https://go.dev/dl/ then retry.
  pause
  exit /b 1
)

set "BSC_LAUNCHER_ROOT=%~dp0bsc-launcher"
if exist "bsc-launcher\.env" (
  for /f "usebackq eol=# tokens=1,* delims==" %%a in ("bsc-launcher\.env") do (
    if not "%%a"=="" (
      set "%%a=%%b"
    )
  )
)

taskkill /IM bsc-launcher.exe /F >nul 2>&1
echo Starting BSC Token Launcher on :9340...
start "BSC Launcher" /MIN cmd /c "set BSC_LAUNCHER_ROOT=%BSC_LAUNCHER_ROOT%&& set BSC_DEPLOYER_PRIVATE_KEY=%BSC_DEPLOYER_PRIVATE_KEY%&& set BSCSCAN_API_KEY=%BSCSCAN_API_KEY%&& set ETHERSCAN_API_KEY=%ETHERSCAN_API_KEY%&& set BSC_RPC_URL=%BSC_RPC_URL%&& bin\bsc-launcher.exe"
timeout /t 2 >nul
start http://127.0.0.1:9340/
echo BSC Token Launcher opened at http://127.0.0.1:9340/
exit /b 0
