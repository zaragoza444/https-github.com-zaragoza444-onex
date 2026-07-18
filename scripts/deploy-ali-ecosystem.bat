@echo off
setlocal
cd /d "%~dp0.."
title Deploy to ALI Ecosystem Server

if "%SSH_PASS%"=="" (
  echo Set SSH_PASS to the ubuntu password for zblockchainsystem.com
  echo   set SSH_PASS=your-password
  echo   scripts\deploy-ali-ecosystem.bat
  exit /b 1
)

python scripts\deploy-ali-ecosystem.py
exit /b %ERRORLEVEL%
