@echo off
title CAWForge AI - Install or Repair
echo ==================================================
echo   CAWForge AI Components - Install / Repair
echo ==================================================
echo.
echo Checking bundled setup script...
cd /d "%~dp0"
if not exist "%~dp0Setup-StableDiffusion.ps1" (
 echo ERROR: Setup-StableDiffusion.ps1 is missing.
 echo Extract the complete CAWForge Windows ZIP.
 pause
 exit /b 1
)
echo Starting PowerShell setup. This may take several minutes.
echo The first installation needs an internet connection and free disk space.
echo.
powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%~dp0Setup-StableDiffusion.ps1"
if errorlevel 1 (
 echo.
 echo ERROR: AI setup failed. Copy the error shown above for troubleshooting.
) else (
 echo.
 echo AI setup completed. Restart CAWForge before using the AI tools.
)
echo.
pause
