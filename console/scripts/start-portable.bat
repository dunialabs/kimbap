@echo off
title Kimbap Console
echo ========================================
echo         Kimbap Console Starting
echo ========================================
echo.

REM 设置环境变量
set "SCRIPT_DIR=%~dp0"
set "ROOT_DIR=%SCRIPT_DIR%.."
set "PATH=%ROOT_DIR%\node;%ROOT_DIR%\postgresql\bin;%PATH%"
set "NODE_ENV=production"

REM 创建必要目录
if not exist "%ROOT_DIR%\logs" mkdir "%ROOT_DIR%\logs"
if not exist "%ROOT_DIR%\postgresql\data" mkdir "%ROOT_DIR%\postgresql\data"

echo 🔧 Initializing Kimbap Console...

REM 检查端口是否被占用
netstat -an | find "3000" > nul
if %errorlevel% == 0 (
    echo ⚠️  Port 3000 is already in use!
    echo Please close the application using port 3000 and try again.
    pause
    exit /b 1
)

REM 智能数据库配置
echo 🔍 Auto-detecting database configuration...
cd /d "%ROOT_DIR%\app"
..\node\node.exe scripts\database-config.js validate

if %errorlevel% neq 0 (
    echo.
    echo ❌ Database setup failed!
    echo.
    echo 📖 Available options:
    echo 1. 🐳 Docker: docker run --name kimbap-postgres -e POSTGRES_USER=kimbap -e POSTGRES_PASSWORD=kimbap123 -e POSTGRES_DB=kimbap_db -p 5432:5432 -d postgres:16
    echo 2. 🏠 Local: Install PostgreSQL 16 and create database 'kimbap_db'
    echo 3. ☁️  Cloud: Set CLOUD_DB_* environment variables in .env.local
    echo.
    pause
    exit /b 1
)

echo ✅ Database ready

REM 启动应用
echo.
echo 🚀 Starting Kimbap Console (Frontend + Backend)...
echo 📱 Open http://localhost:3000 in your browser
echo 🛑 Press Ctrl+C to stop
echo.

REM 启动后端代理服务器
echo 🔧 Starting backend proxy server...
start "Kimbap Backend" /min ..\node\node.exe proxy-server\index.js

REM 等待后端启动
timeout /t 3 > nul

REM 启动前端应用
echo 🎨 Starting frontend application...
..\node\node.exe node_modules\next\dist\bin\next start -p 3000

REM 应用退出后的清理
echo.
echo 🛑 Stopping all services...
taskkill /f /im node.exe /t > nul 2>&1
echo ✅ All processes stopped.
echo.
echo 🛑 Kimbap Console stopped.
pause