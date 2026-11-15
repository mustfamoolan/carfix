@echo off
setlocal enabledelayedexpansion
cd /d "%~dp0"

rem 1. Launch Chrome only once via start_chrome_stealth.bat (if port 9222 is not already listening)
call "%~dp0start_chrome_stealth.bat"

rem 2. Restart only canada_worker.exe silently in background via watchdog (never closes Chrome and leaves zero black CMD windows)
taskkill /F /IM wscript.exe >nul 2>&1
taskkill /F /IM canada_worker.exe >nul 2>&1
start "" wscript.exe //B "%~dp0silent_runner.vbs"

rem 3. Close this batch window immediately - Zero CMD windows remain on screen!
exit /b 0
