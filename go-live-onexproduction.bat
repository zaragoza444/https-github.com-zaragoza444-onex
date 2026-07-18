@echo off
REM Quick go-live helper — runs DNS/HTTPS preflight for onexproduction.com
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0deploy-onexproduction.ps1" -VpsIp zblockchainsystem.com %*
pause
