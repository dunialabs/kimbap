#!/bin/bash

# KIMBAP Console 便携版启动脚本 - 智能数据库配置
echo "========================================"
echo "       KIMBAP Console Starting"
echo "========================================"
echo

# 获取脚本目录
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ROOT_DIR="$( dirname "$SCRIPT_DIR" )"

# 设置环境变量
export PATH="$ROOT_DIR/node/bin:$PATH"
export NODE_ENV="production"

# 创建必要目录
mkdir -p "$ROOT_DIR/logs"
mkdir -p "$ROOT_DIR/postgresql/data"

# 检查端口是否被占用
check_port() {
    if lsof -Pi :3000 -sTCP:LISTEN -t >/dev/null 2>&1; then
        echo "⚠️  Port 3000 is already in use!"
        echo "Please close the application using port 3000 and try again."
        exit 1
    fi
}

# 智能数据库配置
setup_database() {
    echo "🔍 Auto-detecting database configuration..."
    
    cd "$ROOT_DIR/app"
    
    # 运行智能数据库配置
    if ../node/bin/node scripts/database-config.js validate; then
        echo "✅ Database ready"
        return 0
    else
        echo "❌ Database setup failed"
        echo ""
        echo "📖 Available options:"
        echo "1. 🐳 Docker: docker run --name kimbap-postgres -e POSTGRES_USER=kimbap -e POSTGRES_PASSWORD=kimbap123 -e POSTGRES_DB=kimbap_db -p 5432:5432 -d postgres:16"
        echo "2. 🏠 Local: brew install postgresql@16 && brew services start postgresql@16 && createdb kimbap_db"
        echo "3. ☁️  Cloud: Set CLOUD_DB_* environment variables in .env.local"
        echo ""
        return 1
    fi
}

# 清理函数
cleanup() {
    echo ""
    echo "🛑 Stopping services..."
    
    # 停止前端和后端进程
    if [ ! -z "$BACKEND_PID" ]; then
        kill $BACKEND_PID >/dev/null 2>&1
        echo "✅ Backend stopped"
    fi
    
    if [ ! -z "$FRONTEND_PID" ]; then
        kill $FRONTEND_PID >/dev/null 2>&1
        echo "✅ Frontend stopped"
    fi
    
    # 停止所有相关的node进程
    pkill -f "proxy-server/index.js" >/dev/null 2>&1
    pkill -f "next start" >/dev/null 2>&1
    
    # 停止内置PostgreSQL
    if [ -f "$ROOT_DIR/postgresql/data/postmaster.pid" ]; then
        if [ -f "$ROOT_DIR/postgresql/bin/pg_ctl" ]; then
            "$ROOT_DIR/postgresql/bin/pg_ctl" -D "$ROOT_DIR/postgresql/data" stop -m fast >/dev/null 2>&1
            echo "✅ PostgreSQL stopped"
        fi
    fi
    
    echo "✅ KIMBAP Console stopped."
}

# 设置清理函数
trap cleanup EXIT INT TERM

# 主流程
echo "🔧 Initializing KIMBAP Console..."

# 检查端口
check_port

# 配置数据库
if ! setup_database; then
    echo "❌ Database setup failed. Exiting..."
    exit 1
fi

# 启动应用
echo ""
echo "🚀 Starting KIMBAP Console (Frontend + Backend)..."
echo "📱 Open http://localhost:3000 in your browser"
echo "🛑 Press Ctrl+C to stop"
echo ""

cd "$ROOT_DIR/app"

# 启动后端代理服务器
echo "🔧 Starting backend proxy server..."
../node/bin/node proxy-server/index.js &
BACKEND_PID=$!

# 等待后端启动
sleep 2

# 启动前端应用
echo "🎨 Starting frontend application..."
../node/bin/node node_modules/next/dist/bin/next start -p 3000 &
FRONTEND_PID=$!

# 等待任一进程结束
wait