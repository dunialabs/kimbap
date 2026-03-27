@echo off
REM PETA Console - Docker Deployment with Auto Port Allocation (Windows)
REM This script automatically finds available ports and starts Docker services

echo 🐳 PETA Console - Auto Port Deployment (Windows)
echo =============================================
echo.

REM Check if Docker is running
docker info >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Docker is not running. Please start Docker Desktop first.
    pause
    exit /b 1
)

echo ✅ Docker is running

REM Check if Node.js is available
node --version >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Node.js is required for port detection. Please install Node.js.
    pause
    exit /b 1
)

echo ✅ Node.js is available

REM Check if .env file exists
if not exist .env (
    if exist .env.example (
        echo ℹ️ Creating .env file from .env.example
        copy .env.example .env >nul
        echo ⚠️ Please review and edit .env file with your configuration
    ) else (
        echo ⚠️ .env file not found, continuing with default configuration
    )
)

REM Find available ports
echo ℹ️ Finding available ports...
node scripts/find-available-ports.js
if %errorlevel% neq 0 (
    echo ❌ Failed to find available ports
    pause
    exit /b 1
)

REM Check if .env.ports was created
if not exist .env.ports (
    echo ❌ Port configuration file not created
    pause
    exit /b 1
)

echo ✅ Port allocation completed

REM Stop existing services if running
echo ℹ️ Stopping existing services (if any)...
docker compose --env-file .env.ports down --remove-orphans >nul 2>&1

REM Pull latest images
echo ℹ️ Pulling latest Docker images...
docker compose --env-file .env.ports pull

REM Start services with auto-allocated ports
echo ℹ️ Starting PETA Console services...
docker compose --env-file .env.ports up -d
if %errorlevel% neq 0 (
    echo ❌ Failed to start Docker services
    pause
    exit /b 1
)

REM Wait for services to initialize
echo ℹ️ Waiting for services to initialize...
timeout /t 10 /nobreak >nul

REM Check service status
echo ℹ️ Checking service status...
docker compose --env-file .env.ports ps

echo.
echo 🎉 PETA Console is now running!
echo ===============================

REM Parse and display port configuration
for /f "tokens=2 delims==" %%i in ('findstr "PETA_FRONTEND_PORT=" .env.ports') do set FRONTEND_PORT=%%i
for /f "tokens=2 delims==" %%i in ('findstr "PETA_BACKEND_PORT=" .env.ports') do set BACKEND_PORT=%%i
for /f "tokens=2 delims==" %%i in ('findstr "PETA_ADMINER_PORT=" .env.ports') do set ADMINER_PORT=%%i

echo.
echo 🌐 Service URLs:
echo    • Frontend:       http://localhost:%FRONTEND_PORT%
echo    • Backend API:    http://localhost:%BACKEND_PORT%
echo    • Database Admin: http://localhost:%ADMINER_PORT%
echo.
echo 📋 Management Commands:
echo    • View logs:      docker compose --env-file .env.ports logs -f
echo    • Stop services:  docker compose --env-file .env.ports down
echo    • Restart:        docker compose --env-file .env.ports restart
echo    • Status:         docker compose --env-file .env.ports ps
echo.
echo 📁 Configuration:
echo    • Port config:    .env.ports
echo    • JSON config:    .port-config.json
echo.

echo ✅ Deployment completed successfully!
echo.

REM Ask if user wants to open browser
set /p "OPEN_BROWSER=📱 Open application in browser? (y/N): "
if /i "%OPEN_BROWSER%"=="y" (
    start http://localhost:%FRONTEND_PORT%
)

pause