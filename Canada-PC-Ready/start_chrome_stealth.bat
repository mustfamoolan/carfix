@echo off
cd /d "%~dp0"
chcp 65001 >nul
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

echo [*] Closing any open Chrome windows temporarily to unlock Cookies and Session...
taskkill /F /IM chrome.exe /T >nul 2>&1
timeout /t 2 /nobreak >nul

echo [*] Syncing latest Cookies, Login Session, and Profile from your main Chrome...
robocopy "%LOCALAPPDATA%\Google\Chrome\User Data" "C:\CarfaxDealerProfile" /E /IS /IT /R:1 /W:1 /XD "Cache" "Code Cache" "GPUCache" "ShaderCache" "GrShaderCache" "DawnCache" "Crashpad" "SwReporter" "Safe Browsing" "OptimizationGuidePredictionModels" "BrowserMetrics" "ComponentCRXCache" /XF "*.tmp" "lockfile" "Singleton*" >nul 2>&1
echo [OK] Cookies and Active Login Session copied successfully!
echo.

echo [*] Launching Real Chrome with Off-Screen Stealth Flags on Port 9222...
echo [*] Profile Directory: C:\CarfaxDealerProfile
echo [*] Target Portal    : https://www.carfaxonline.com
echo [*] Window Mode      : Off-Screen Stealth (-10000,-10000) [Full GPU & Render Enabled]
echo.
echo [IMPORTANT INSTRUCTIONS]
echo 1. Chrome is now active with your synced login session in Off-Screen mode.
echo 2. Chrome runs with full GPU and display pipeline without cluttering your physical screen.
echo 3. Keep this host running.
echo.

%CHROME_PATH% --remote-debugging-port=9222 --remote-allow-origins=* --user-data-dir="C:\CarfaxDealerProfile" --window-position=-10000,-10000 --window-size=1920,1080 --disable-blink-features=AutomationControlled --no-first-run --no-default-browser-check https://www.carfaxonline.com

pause
