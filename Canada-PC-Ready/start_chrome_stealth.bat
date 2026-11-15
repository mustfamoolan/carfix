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

rem Check if Worker Chrome is ALREADY running on port 9222 - if so, NEVER kill it or touch its session!
netstat -a -n -o | findstr "127.0.0.1:9222" | findstr "LISTENING" >nul 2>&1
if %errorlevel% equ 0 (
    exit /b 0
)

rem Dedicated, isolated Chrome profile (never overwritten by Main Chrome so login cookies & 2FA trust are preserved forever)
if not exist "C:\CarfaxDealerProfile" (
    mkdir "C:\CarfaxDealerProfile" >nul 2>&1
)

start "" "%CHROME_PATH%" --remote-debugging-port=9222 --remote-allow-origins=* --user-data-dir="C:\CarfaxDealerProfile" --profile-directory="Default" --disable-popup-blocking --disable-session-crashed-bubble --disable-blink-features=AutomationControlled --no-first-run --no-default-browser-check https://www.carfaxonline.com
exit /b 0
