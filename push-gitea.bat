@echo off
cd /d "%~dp0"
git remote set-url gitea https://git.anakatech.llc/zaragoza/onex.git
echo Pushing main to Gitea zaragoza/onex ...
git push -u gitea main
if errorlevel 1 (
  echo.
  echo If push failed: create empty repo at https://git.anakatech.llc/zaragoza/onex
  echo Then sign in to Gitea in browser and retry.
  pause
  exit /b 1
)
echo Done: https://git.anakatech.llc/zaragoza/onex
pause
