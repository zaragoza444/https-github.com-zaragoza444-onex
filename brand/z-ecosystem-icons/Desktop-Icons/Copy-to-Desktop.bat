@echo off
set DEST=%USERPROFILE%\Desktop\Z-Ecosystem-Icons
mkdir "%DEST%" 2>nul
copy /Y "%~dp0*.ico" "%DEST%\" >nul
copy /Y "%~dp0*.png" "%DEST%\" >nul
echo Copied Z ecosystem icons to %DEST%
explorer "%DEST%"
pause
