@echo off
title CarFix Canada Stealth Chrome Host (CARFAXOnline)
color 0A

echo ========================================================
echo    CarFix Platform - Canada Chrome Dealer Profile
echo ========================================================
echo.
echo [*] Looking for Google Chrome installation...

set CHROME_PATH=""
if exist "C:\Program Files\Google\Chrome\Application\chrome.exe" (
    set CHROME_PATH="C:\Program Files\Google\Chrome\Application\chrome.exe"
) else if exist "C:\Program Files (x86)\Google\Chrome\Application\chrome.exe" (
    set CHROME_PATH="C:\Program Files (x86)\Google\Chrome\Application\chrome.exe"
) else if exist "%LOCALAPPDATA%\Google\Chrome\Application\chrome.exe" (
    set CHROME_PATH="%LOCALAPPDATA%\Google\Chrome\Application\chrome.exe"
)

if %CHROME_PATH%=="" (
    echo [ERROR] Google Chrome was not found on this Windows machine!
    echo Please install Google Chrome and try again.
    pause
    exit /b 1
)

echo [*] Launching Real Chrome with Stealth Flags on Port 9222...
echo [*] Profile Directory: C:\CarfaxDealerProfile
echo [*] Target Portal    : https://www.carfaxonline.com
echo.
echo [IMPORTANT INSTRUCTIONS]
echo 1. Login to your official CARFAXOnline Dealer account manually in this browser window.
echo 2. Stay on the main VIN search screen.
echo 3. Keep this Chrome window OPEN. Do not close it.
echo.

%CHROME_PATH% --remote-debugging-port=9222 --remote-allow-origins=* --user-data-dir="C:\CarfaxDealerProfile" --disable-blink-features=AutomationControlled --no-first-run --no-default-browser-check https://www.carfaxonline.com

pause
