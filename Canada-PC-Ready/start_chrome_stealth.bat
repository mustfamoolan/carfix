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

rem Only seed C:\CarfaxDealerProfile from Main Chrome on first-time setup when Local State does not exist yet
if not exist "C:\CarfaxDealerProfile\Local State" (
    if exist "%LOCALAPPDATA%\Google\Chrome\User Data" (
        robocopy "%LOCALAPPDATA%\Google\Chrome\User Data" "C:\CarfaxDealerProfile" /E /XO /R:1 /W:1 /XD "Cache" "Code Cache" "GPUCache" "ShaderCache" "GrShaderCache" "DawnCache" "Crashpad" "SwReporter" "Safe Browsing" "OptimizationGuidePredictionModels" "BrowserMetrics" "ComponentCRXCache" /XF "*.tmp" "lockfile" "Singleton*" >nul 2>&1
    )
)

start "" "%CHROME_PATH%" --remote-debugging-port=9222 --remote-allow-origins=* --user-data-dir="C:\CarfaxDealerProfile" --disable-blink-features=AutomationControlled --no-first-run --no-default-browser-check https://www.carfaxonline.com
exit /b 0
