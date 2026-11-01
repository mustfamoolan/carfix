@echo off
title CarFix One-Click Production Deploy (vinreport.tech)
color 0A

echo =================================================================
echo    CarFix Platform - Automated Production Deployment
echo    Target: https://vinreport.tech (187.126.4.143)
echo =================================================================
echo.

echo [1/3] Building Frontend Assets (Vite)...
cd frontend
call npm run build
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Frontend build failed!
    pause
    exit /b 1
)
cd ..

echo.
echo [2/3] Pushing Updates to GitHub (origin/main)...
git add .
git commit -m "chore: production update"
git push origin main

echo.
echo [3/3] Updating & Restarting Containers on Hostinger VPS...
"C:\Program Files\PuTTY\plink.exe" -batch -ssh root@187.126.4.143 -pw "Moolan0770##" "cd /var/www/carfix && git pull origin main && docker compose -f docker-compose.prod.yml up -d --build"

echo.
echo =================================================================
echo    Deployment Complete! Live at: https://vinreport.tech
echo =================================================================
pause

