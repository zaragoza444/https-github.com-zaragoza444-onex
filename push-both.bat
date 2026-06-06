@echo off
cd /d "%~dp0"
git remote set-url github https://github.com/zaragoza444/shiva-blockchain.git
git remote set-url gitea https://git.anakatech.llc/zaragoza/onex.git
echo Pushing main to GitHub zaragoza444/shiva-blockchain ...
git push -u github main
if errorlevel 1 goto fail
echo Pushing main to Gitea zaragoza/onex ...
git push -u gitea main
if errorlevel 1 goto fail
echo.
echo GitHub: https://github.com/zaragoza444/shiva-blockchain
echo Gitea:  https://git.anakatech.llc/zaragoza/onex
pause
exit /b 0
:fail
echo Push failed - check login / create empty Gitea repo first.
pause
exit /b 1
