@echo off
setlocal enabledelayedexpansion
cd /d "%~dp0"

set "CHROME_PATH="
if exist "C:\Program Files\Google\Chrome\Application\chrome.exe" (
    set "CHROME_PATH=C:\Program Files\Google\Chrome\Application\chrome.exe"
) else if exist "C:\Program Files (x86)\Google\Chrome\Application\chrome.exe" (
    set "CHROME_PATH=C:\Program Files (x86)\Google\Chrome\Application\chrome.exe"
) else if exist "%LOCALAPPDATA%\Google\Chrome\Application\chrome.exe" (
    set "CHROME_PATH=%LOCALAPPDATA%\Google\Chrome\Application\chrome.exe"
)

if "%CHROME_PATH%"=="" exit /b 1

rem 1. Only launch Chrome if port 9222 is NOT already open (preserves active Carfax login & 2FA cookies!)
netstat -a -n -o | findstr "127.0.0.1:9222" | findstr "LISTENING" >nul 2>&1
if %errorlevel% neq 0 (
    if not exist "C:\CarfaxDealerProfile" (
        mkdir "C:\CarfaxDealerProfile" >nul 2>&1
    )
    start "" "%CHROME_PATH%" --remote-debugging-port=9222 --remote-allow-origins=* --user-data-dir="C:\CarfaxDealerProfile" --profile-directory="Default" --restore-last-session --disable-blink-features=AutomationControlled --no-first-run --no-default-browser-check https://www.carfaxonline.com
)

rem 2. Restart only canada_worker.exe silently in background via watchdog (never closes Chrome and leaves zero black CMD windows)
taskkill /F /IM wscript.exe >nul 2>&1
taskkill /F /IM canada_worker.exe >nul 2>&1
start "" wscript.exe //B "%~dp0silent_runner.vbs"

rem 3. Close this batch window immediately - Zero CMD windows remain on screen!
exit /b 0
