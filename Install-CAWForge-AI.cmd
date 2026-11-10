@echo off
cd /d "%~dp0"
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0Setup-StableDiffusion.ps1"
if errorlevel 1 (
 echo.
 echo Setup failed. Copy the error text above when asking for help.
) else (
 echo.
 echo Setup complete. Launch CAWForge to test Stable Diffusion.
)
pause
