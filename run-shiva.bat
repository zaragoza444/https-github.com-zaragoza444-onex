@echo off
cd /d "%~dp0"
if not exist "bin\shivad.exe" (
  echo Building Shiva node...
  call "%~dp0build-shiva.bat"
)
if exist "bin\shivad.exe" (
  start "Shiva Node" "bin\shivad.exe" -datadir "%~dp0data" -api :8545 -listen :30303
  timeout /t 2 >nul
  start http://localhost:8545/explorer/
  exit /b 0
)
echo Shiva node not built in this folder yet.
echo.
echo Options:
echo   1. Install Go and build: go build -o bin\shivad.exe .\cmd\shivad
echo   2. Run from WSL: /home/ubuntu/shiva-blockchain
echo.
start notepad "%~dp0README.txt"
pause
