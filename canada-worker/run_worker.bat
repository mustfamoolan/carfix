@echo off
title CarFix Canada One-Shot Worker Node
color 0B

echo ========================================================
echo    CarFix Platform - Canada Stealth Worker Node
echo ========================================================
echo.
if not exist "canada_worker.exe" (
    echo [ERROR] canada_worker.exe not found in current folder!
    pause
    exit /b 1
)

canada_worker.exe
pause

