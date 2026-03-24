#!/usr/bin/env node

/**
 * Kimbap Console 便携包构建脚本
 * 将应用打包成可在任意电脑上独立运行的程序包
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');
const https = require('https');
const { createReadStream, createWriteStream } = require('fs');
const { pipeline } = require('stream');
const { promisify } = require('util');

const DependencyDownloader = require('./download-dependencies');
const ExtractUtils = require('./extract-utils');

const pipelineAsync = promisify(pipeline);

class PortableBuilder {
  constructor() {
    this.rootDir = path.resolve(__dirname, '../..');
    this.platform = process.platform;
    this.arch = process.arch;
    this.outputDir = path.join(this.rootDir, 'dist', `kimbap-console-${this.platform}`);
    this.tempDir = path.join(this.rootDir, 'temp-build');
    
    // 版本信息
    this.nodeVersion = '20.11.0';  // LTS version
    this.postgresVersion = '16.1';
    
    console.log(`🚀 Building portable package for ${this.platform}-${this.arch}`);
  }

  async build() {
    try {
      console.log('📦 Starting portable build process...');
      
      // 1. 清理和准备目录
      await this.prepare();
      
      // 2. 构建 Next.js 应用
      await this.buildNextApp();
      
      // 3. 下载 Node.js 便携版
      await this.downloadNode();
      
      // 4. 下载 PostgreSQL 便携版
      await this.downloadPostgreSQL();
      
      // 5. 复制应用文件
      await this.copyAppFiles();
      
      // 6. 创建启动脚本
      await this.createStartupScripts();
      
      // 7. 创建配置文件
      await this.createConfigFiles();
      
      // 8. 创建说明文档
      await this.createDocumentation();
      
      // 9. 清理临时文件
      await this.cleanup();
      
      console.log(`✅ Portable package built successfully!`);
      console.log(`📁 Output directory: ${this.outputDir}`);
      console.log(`📊 Package size: ${this.getDirectorySize(this.outputDir)}MB`);
      
    } catch (error) {
      console.error('❌ Build failed:', error);
      await this.cleanup();
      process.exit(1);
    }
  }

  async prepare() {
    console.log('🔧 Preparing build directories...');
    
    // 清理输出目录
    if (fs.existsSync(this.outputDir)) {
      fs.rmSync(this.outputDir, { recursive: true, force: true });
    }
    
    // 创建必要目录
    fs.mkdirSync(this.outputDir, { recursive: true });
    fs.mkdirSync(this.tempDir, { recursive: true });
    fs.mkdirSync(path.join(this.outputDir, 'app'), { recursive: true });
    fs.mkdirSync(path.join(this.outputDir, 'node'), { recursive: true });
    fs.mkdirSync(path.join(this.outputDir, 'postgresql'), { recursive: true });
    fs.mkdirSync(path.join(this.outputDir, 'scripts'), { recursive: true });
    fs.mkdirSync(path.join(this.outputDir, 'config'), { recursive: true });
  }

  async buildNextApp() {
    console.log('🏗️  Building Next.js application...');
    
    process.chdir(this.rootDir);
    
    try {
      // 安装所有依赖（包括devDependencies用于构建）
      execSync('npm ci', { stdio: 'inherit' });
      
      // 创建便携包专用的Next.js配置（禁用静态导出）
      await this.createPortableNextConfig();
      
      // 使用便携包配置构建应用
      execSync('NODE_ENV=production NEXT_CONFIG=next.config.portable.mjs npm run build', { stdio: 'inherit' });
      
      // 恢复原始配置
      await this.restoreOriginalNextConfig();
      
      console.log('✅ Next.js build completed');
    } catch (error) {
      await this.restoreOriginalNextConfig();
      throw new Error(`Next.js build failed: ${error.message}`);
    }
  }

  async createPortableNextConfig() {
    const portableConfig = `/** @type {import('next').NextConfig} */
const nextConfig = {
  // 便携包构建配置 - 支持API routes和SSR
  output: undefined, // 禁用静态导出以支持API routes

  // 图片配置
  images: {
    unoptimized: true,
    formats: ['image/webp', 'image/avif'],
    minimumCacheTTL: 60,
    dangerouslyAllowSVG: true,
    contentSecurityPolicy: "default-src 'self'; script-src 'none'; sandbox;"
  },

  // 实验性功能和性能优化
  experimental: {
    optimizeCss: false,
    optimizePackageImports: ['@radix-ui/react-*', 'lucide-react']
  },

  // TypeScript 配置
  typescript: {
    ignoreBuildErrors: true
  },

  // ESLint 配置
  eslint: {
    ignoreDuringBuilds: false
  },

  // 启用压缩
  compress: true,

  // 启用严格模式
  reactStrictMode: true,

  // 移除 Next.js 标识
  poweredByHeader: false,

  // Webpack 配置优化
  webpack: (config, { dev, isServer }) => {
    // Node.js 环境下的 fallback 配置
    if (!isServer) {
      config.resolve.fallback = {
        ...config.resolve.fallback,
        fs: false,
        net: false,
        tls: false
      }
    }

    // 生产环境优化
    if (!dev && !isServer) {
      config.optimization.splitChunks = {
        chunks: 'all',
        cacheGroups: {
          default: false,
          vendor: {
            name: 'vendor',
            chunks: 'all',
            test: /node_modules/,
            priority: 10
          },
          common: {
            minChunks: 2,
            priority: -10,
            reuseExistingChunk: true
          }
        }
      }
    }

    return config
  }
}

export default nextConfig`;

    // 备份原始配置
    const originalConfig = path.join(this.rootDir, 'next.config.mjs');
    const backupConfig = path.join(this.rootDir, 'next.config.mjs.backup');
    fs.copyFileSync(originalConfig, backupConfig);

    // 写入便携包配置
    const portableConfigPath = path.join(this.rootDir, 'next.config.portable.mjs');
    fs.writeFileSync(portableConfigPath, portableConfig);
    
    // 临时替换配置文件
    fs.copyFileSync(portableConfigPath, originalConfig);
  }

  async restoreOriginalNextConfig() {
    const originalConfig = path.join(this.rootDir, 'next.config.mjs');
    const backupConfig = path.join(this.rootDir, 'next.config.mjs.backup');
    const portableConfig = path.join(this.rootDir, 'next.config.portable.mjs');
    
    // 恢复原始配置
    if (fs.existsSync(backupConfig)) {
      fs.copyFileSync(backupConfig, originalConfig);
      fs.unlinkSync(backupConfig);
    }
    
    // 清理便携包配置
    if (fs.existsSync(portableConfig)) {
      fs.unlinkSync(portableConfig);
    }
  }

  async downloadNode() {
    console.log('📥 Downloading and extracting Node.js...');
    
    const downloader = new DependencyDownloader();
    const extractor = new ExtractUtils();
    
    try {
      // 下载 Node.js
      await downloader.downloadNode();
      
      // 获取下载的文件路径
      const nodeUrl = downloader.getNodeDownloadUrl();
      const fileName = path.basename(nodeUrl);
      const archivePath = path.join(downloader.tempDir, fileName);
      const extractDir = path.join(this.outputDir, 'node');
      
      // 解压缩
      await extractor.extractArchive(archivePath, extractDir);
      
      // 整理文件结构
      await extractor.organizeExtractedFiles(extractDir);
      
      // 验证关键文件
      const nodeExecutable = this.platform === 'win32' ? 'node.exe' : 'bin/node';
      extractor.validateExtraction(extractDir, [nodeExecutable]);
      
      console.log('✅ Node.js downloaded and extracted');
    } catch (error) {
      throw new Error(`Node.js setup failed: ${error.message}`);
    }
  }

  async downloadPostgreSQL() {
    console.log('📥 Setting up PostgreSQL...');
    
    try {
      // 检查系统是否已安装 PostgreSQL，如果有就复制
      const systemPgPath = this.findSystemPostgreSQL();
      
      if (systemPgPath) {
        console.log(`📋 Found system PostgreSQL at: ${systemPgPath}`);
        await this.copySystemPostgreSQL(systemPgPath);
      } else {
        console.log('📦 Downloading portable PostgreSQL...');
        await this.downloadPortablePostgreSQL();
      }
      
      // 创建数据库初始化脚本
      await this.createDatabaseInitScript();
      
      console.log('✅ PostgreSQL setup completed');
    } catch (error) {
      console.warn('⚠️  PostgreSQL setup failed, falling back to external database requirement');
      console.warn('Error:', error.message);
      
      // 创建说明文档而不是失败
      await this.createPostgreSQLInstructions();
    }
  }

  findSystemPostgreSQL() {
    // 尝试找到系统安装的 PostgreSQL
    const possiblePaths = [
      '/usr/local/bin/postgres',           // Homebrew on Intel Mac
      '/opt/homebrew/bin/postgres',        // Homebrew on Apple Silicon  
      '/usr/bin/postgres',                 // System install on Linux
      '/Applications/Postgres.app/Contents/Versions/16/bin/postgres',  // Postgres.app
      'C:\\Program Files\\PostgreSQL\\16\\bin\\postgres.exe'  // Windows
    ];

    for (const pgPath of possiblePaths) {
      if (fs.existsSync(pgPath)) {
        return path.dirname(pgPath);
      }
    }
    
    // 尝试使用 which/where 命令
    try {
      const command = this.platform === 'win32' ? 'where' : 'which';
      const result = execSync(`${command} postgres`, { encoding: 'utf8', stdio: 'pipe' });
      const pgPath = result.trim().split('\n')[0];
      if (pgPath && fs.existsSync(pgPath)) {
        return path.dirname(pgPath);
      }
    } catch (error) {
      // Command not found or postgres not in PATH
    }
    
    return null;
  }

  async copySystemPostgreSQL(systemPgPath) {
    console.log('📋 Copying system PostgreSQL binaries...');
    
    const pgDir = path.join(this.outputDir, 'postgresql');
    fs.mkdirSync(pgDir, { recursive: true });
    
    // 复制关键的二进制文件
    const requiredBinaries = [
      'postgres',
      'initdb', 
      'pg_ctl',
      'createdb',
      'createuser',
      'psql'
    ];

    const binDir = path.join(pgDir, 'bin');
    fs.mkdirSync(binDir, { recursive: true });

    let copiedFiles = 0;
    for (const binary of requiredBinaries) {
      const srcPath = path.join(systemPgPath, this.platform === 'win32' ? `${binary}.exe` : binary);
      const destPath = path.join(binDir, path.basename(srcPath));
      
      if (fs.existsSync(srcPath)) {
        try {
          fs.copyFileSync(srcPath, destPath);
          if (this.platform !== 'win32') {
            fs.chmodSync(destPath, 0o755);
          }
          copiedFiles++;
        } catch (error) {
          console.warn(`⚠️  Failed to copy ${binary}:`, error.message);
        }
      }
    }

    console.log(`✅ Copied ${copiedFiles}/${requiredBinaries.length} PostgreSQL binaries`);

    // 尝试复制共享库
    await this.copyPostgreSQLLibraries(systemPgPath, pgDir);
  }

  async copyPostgreSQLLibraries(systemPgPath, pgDir) {
    // 查找并复制 PostgreSQL 的共享库
    const possibleLibDirs = [
      path.join(path.dirname(systemPgPath), 'lib'),
      path.join(path.dirname(path.dirname(systemPgPath)), 'lib'),
      '/usr/local/lib',
      '/opt/homebrew/lib'
    ];

    for (const libDir of possibleLibDirs) {
      if (fs.existsSync(libDir)) {
        try {
          const libFiles = fs.readdirSync(libDir).filter(f => 
            f.includes('postgres') || f.includes('pq') || f.startsWith('libpq')
          );
          
          if (libFiles.length > 0) {
            const destLibDir = path.join(pgDir, 'lib');
            fs.mkdirSync(destLibDir, { recursive: true });
            
            for (const libFile of libFiles.slice(0, 10)) { // 限制复制数量
              try {
                const srcPath = path.join(libDir, libFile);
                const destPath = path.join(destLibDir, libFile);
                fs.copyFileSync(srcPath, destPath);
              } catch (error) {
                // 忽略复制失败的库文件
              }
            }
            
            console.log(`✅ Copied ${libFiles.length} PostgreSQL libraries`);
            break;
          }
        } catch (error) {
          // 忽略权限或其他错误
        }
      }
    }
  }

  async downloadPortablePostgreSQL() {
    // 如果系统没有 PostgreSQL，尝试下载便携版
    // 由于下载和配置复杂，这里先使用简化实现
    console.log('⚠️  Portable PostgreSQL download not implemented yet');
    console.log('📝 Creating setup instructions instead...');
    
    await this.createPostgreSQLInstructions();
  }

  async createDatabaseInitScript() {
    console.log('📝 Creating database initialization scripts...');
    
    const scriptsDir = path.join(this.outputDir, 'scripts');
    
    // 创建数据库初始化脚本
    const initScript = this.platform === 'win32' 
      ? await this.createWindowsInitScript()
      : await this.createUnixInitScript();
    
    const scriptName = this.platform === 'win32' ? 'init-db.bat' : 'init-db.sh';
    const scriptPath = path.join(scriptsDir, scriptName);
    
    fs.writeFileSync(scriptPath, initScript);
    
    if (this.platform !== 'win32') {
      fs.chmodSync(scriptPath, 0o755);
    }
    
    console.log(`✅ Created database initialization script: ${scriptName}`);
  }

  async createWindowsInitScript() {
    return `@echo off
echo 🔧 Initializing PostgreSQL database...

set "SCRIPT_DIR=%~dp0"
set "PGROOT=%SCRIPT_DIR%..\\postgresql"
set "PGDATA=%PGROOT%\\data"
set "PGBIN=%PGROOT%\\bin"

if exist "%PGDATA%" (
    echo ✅ Database already initialized
    exit /b 0
)

echo 📁 Creating data directory...
mkdir "%PGDATA%"

echo 🗄️  Initializing database cluster...
"%PGBIN%\\initdb.exe" -D "%PGDATA%" -U kimbap --auth-local=trust --auth-host=md5 -W

echo 🚀 Starting PostgreSQL...
"%PGBIN%\\pg_ctl.exe" -D "%PGDATA%" -l "%PGROOT%\\postgresql.log" start

echo ⏳ Waiting for database to start...
timeout /t 5 > nul

echo 🗃️  Creating application database...
"%PGBIN%\\createdb.exe" -U kimbap kimbap_db

echo ✅ Database initialization completed!
`;
  }

  async createUnixInitScript() {
    return `#!/bin/bash
echo "🔧 Initializing PostgreSQL database..."

SCRIPT_DIR="$( cd "$( dirname "\${BASH_SOURCE[0]}" )" && pwd )"
PGROOT="$SCRIPT_DIR/../postgresql"
PGDATA="$PGROOT/data"
PGBIN="$PGROOT/bin"

if [ -d "$PGDATA" ]; then
    echo "✅ Database already initialized"
    exit 0
fi

echo "📁 Creating data directory..."
mkdir -p "$PGDATA"

echo "🗄️  Initializing database cluster..."
"$PGBIN/initdb" -D "$PGDATA" -U kimbap --auth-local=trust --auth-host=md5

echo "🚀 Starting PostgreSQL..."
"$PGBIN/pg_ctl" -D "$PGDATA" -l "$PGROOT/postgresql.log" start

echo "⏳ Waiting for database to start..."
sleep 5

echo "🗃️  Creating application database..."
"$PGBIN/createdb" -U kimbap kimbap_db

echo "✅ Database initialization completed!"
`;
  }

  async createPostgreSQLInstructions() {
    const instructions = `# PostgreSQL Setup for Kimbap Console

## 自动检测结果
系统中未找到可用的 PostgreSQL 安装。

## 推荐设置方案

### 选项1: 使用 Docker (最简单)
\`\`\`bash
docker run --name kimbap-postgres \\
  -e POSTGRES_USER=kimbap \\
  -e POSTGRES_PASSWORD=kimbap123 \\
  -e POSTGRES_DB=kimbap_db \\
  -p 5432:5432 -d postgres:16
\`\`\`

### 选项2: 安装本地 PostgreSQL

#### macOS (Homebrew):
\`\`\`bash
brew install postgresql@16
brew services start postgresql@16
createdb kimbap_db
\`\`\`

#### Linux (Ubuntu/Debian):
\`\`\`bash
sudo apt update
sudo apt install postgresql-16 postgresql-client-16
sudo systemctl start postgresql
sudo -u postgres createdb kimbap_db
\`\`\`

#### Windows:
1. 下载 PostgreSQL 16 安装包
2. 安装并设置密码
3. 使用 pgAdmin 或命令行创建数据库 kimbap_db

### 选项3: 使用现有数据库
编辑 \`app/.env.local\` 文件中的 DATABASE_URL 连接字符串。

## 验证设置
运行以下命令验证数据库连接:
\`\`\`bash
psql postgresql://kimbap:kimbap123@localhost:5432/kimbap_db -c "SELECT version();"
\`\`\`

---
如需帮助，请参考: https://www.postgresql.org/download/
`;

    const pgDir = path.join(this.outputDir, 'postgresql');
    fs.writeFileSync(path.join(pgDir, 'SETUP.md'), instructions);
  }

  async copyAppFiles() {
    console.log('📋 Copying application files...');
    
    const appDir = path.join(this.outputDir, 'app');
    
    // 复制构建文件
    this.copyDir(path.join(this.rootDir, '.next'), path.join(appDir, '.next'));
    this.copyDir(path.join(this.rootDir, 'public'), path.join(appDir, 'public'));
    this.copyDir(path.join(this.rootDir, 'prisma'), path.join(appDir, 'prisma'));
    this.copyDir(path.join(this.rootDir, 'scripts'), path.join(appDir, 'scripts'));
    
    // 复制配置文件
    fs.copyFileSync(path.join(this.rootDir, 'package.json'), path.join(appDir, 'package.json'));
    fs.copyFileSync(path.join(this.rootDir, 'next.config.mjs'), path.join(appDir, 'next.config.mjs'));
    
    // 安装仅生产依赖到最终包中
    process.chdir(appDir);
    execSync('npm ci --production', { stdio: 'inherit' });
    
    // 创建智能环境配置（启动时自动检测数据库）
    const prodEnv = `NODE_ENV=production
# 数据库URL将在启动时自动检测和配置
DATABASE_URL=postgresql://kimbap:kimbap123@localhost:5432/kimbap_db
# 云端数据库配置（可选）
# CLOUD_DB_HOST=your-cloud-host
# CLOUD_DB_USER=your-username
# CLOUD_DB_PASSWORD=your-password
# CLOUD_DB_NAME=kimbap_db
# CLOUD_DB_PORT=5432`;
    
    fs.writeFileSync(path.join(appDir, '.env.local'), prodEnv);
    
    console.log('✅ Application files copied');
  }

  async createStartupScripts() {
    console.log('📝 Creating startup scripts...');
    
    if (this.platform === 'win32') {
      await this.createWindowsScript();
    } else {
      await this.createUnixScript();
    }
    
    console.log('✅ Startup scripts created');
  }

  async createWindowsScript() {
    const script = `@echo off
title Kimbap Console
echo ========================================
echo         Kimbap Console Starting
echo ========================================
echo.

REM 设置环境变量
set "SCRIPT_DIR=%~dp0"
set "PATH=%SCRIPT_DIR%postgresql\\bin;%SCRIPT_DIR%node;%PATH%"
set "PGDATA=%SCRIPT_DIR%postgresql\\data"
set "DATABASE_URL=postgresql://kimbap:kimbap123@localhost:5432/kimbap_db"
set "NODE_ENV=production"

REM 检查端口是否被占用
netstat -an | find "3000" > nul
if %errorlevel% == 0 (
    echo ⚠️  Port 3000 is already in use!
    echo Please close the application using port 3000 and try again.
    pause
    exit /b 1
)

REM 初始化数据库（首次运行）
if not exist "%PGDATA%" (
    echo 🔧 Initializing database for first time...
    echo This may take a few moments...
    
    REM 创建数据目录
    mkdir "%PGDATA%"
    
    REM 初始化数据库
    initdb -D "%PGDATA%" -U kimbap --auth-local=trust --auth-host=md5
    if %errorlevel% neq 0 (
        echo ❌ Database initialization failed!
        pause
        exit /b 1
    )
    
    REM 启动数据库
    pg_ctl -D "%PGDATA%" -l "%SCRIPT_DIR%logs\\postgresql.log" start
    if %errorlevel% neq 0 (
        echo ❌ Failed to start database!
        pause
        exit /b 1
    )
    
    REM 等待数据库启动
    timeout /t 5 > nul
    
    REM 创建数据库
    createdb -U kimbap kimbap_db
    if %errorlevel% neq 0 (
        echo ❌ Failed to create database!
        pause
        exit /b 1
    )
    
    REM 运行数据库迁移
    cd /d "%SCRIPT_DIR%app"
    ..\\node\\node.exe node_modules\\.bin\\prisma migrate deploy
    if %errorlevel% neq 0 (
        echo ❌ Database migration failed!
        pause
        exit /b 1
    )
    
    echo ✅ Database initialized successfully!
) else (
    echo 🔄 Starting existing database...
    
    REM 启动数据库
    pg_ctl -D "%PGDATA%" -l "%SCRIPT_DIR%logs\\postgresql.log" start
    if %errorlevel% neq 0 (
        echo ❌ Failed to start database!
        pause
        exit /b 1
    )
)

REM 等待数据库完全启动
timeout /t 3 > nul

REM 启动应用
echo 🚀 Starting Kimbap Console...
cd /d "%SCRIPT_DIR%app"
..\\node\\node.exe node_modules\\next\\dist\\bin\\next start -p 3000

REM 如果应用异常退出，显示错误信息
if %errorlevel% neq 0 (
    echo.
    echo ❌ Application failed to start!
    echo Check the logs for more information.
)

echo.
echo 🛑 Kimbap Console stopped.
pause`;

    fs.writeFileSync(path.join(this.outputDir, 'scripts', 'start.bat'), script);
  }

  async createUnixScript() {
    const script = `#!/bin/bash

# Kimbap Console 启动脚本 - 智能 PostgreSQL 检测版本
echo "========================================"
echo "       Kimbap Console Starting"
echo "========================================"
echo

# 获取脚本目录
SCRIPT_DIR="$( cd "$( dirname "\${BASH_SOURCE[0]}" )" && pwd )"
SCRIPT_DIR="$( dirname "$SCRIPT_DIR" )"

# 设置基本环境变量
export PATH="$SCRIPT_DIR/node/bin:$PATH"
export NODE_ENV="production"

# 创建日志和数据目录
mkdir -p "$SCRIPT_DIR/logs"
mkdir -p "$SCRIPT_DIR/postgresql/data"

# 检查端口是否被占用
if lsof -Pi :3000 -sTCP:LISTEN -t >/dev/null ; then
    echo "⚠️  Port 3000 is already in use!"
    echo "Please close the application using port 3000 and try again."
    exit 1
fi

# PostgreSQL 设置函数
setup_postgresql() {
    echo "🔍 Checking PostgreSQL setup..."
    
    # 检查内置 PostgreSQL
    if [ -f "$SCRIPT_DIR/postgresql/bin/postgres" ]; then
        echo "✅ Found embedded PostgreSQL"
        export PATH="$SCRIPT_DIR/postgresql/bin:$PATH"
        export PGDATA="$SCRIPT_DIR/postgresql/data"
        export DATABASE_URL="postgresql://kimbap:kimbap123@localhost:5432/kimbap_db"
        
        # 初始化数据库（首次运行）
        if [ ! -d "$PGDATA/base" ]; then
            echo "🔧 Initializing embedded database..."
            
            # 运行初始化脚本
            if [ -f "$SCRIPT_DIR/scripts/init-db.sh" ]; then
                "$SCRIPT_DIR/scripts/init-db.sh"
            else
                # 手动初始化
                initdb -D "$PGDATA" -U kimbap --auth-local=trust --auth-host=md5
                pg_ctl -D "$PGDATA" -l "$SCRIPT_DIR/logs/postgresql.log" start
                sleep 5
                createdb -U kimbap kimbap_db
            fi
        else
            echo "🔄 Starting embedded database..."
            pg_ctl -D "$PGDATA" -l "$SCRIPT_DIR/logs/postgresql.log" start
        fi
        
        # 等待数据库启动
        sleep 3
        
        # 运行数据库迁移
        cd "$SCRIPT_DIR/app"
        ../node/bin/node node_modules/.bin/prisma migrate deploy
        
        echo "✅ Embedded PostgreSQL ready"
        return 0
        
    else
        echo "📝 No embedded PostgreSQL found"
        echo "🔍 Checking for external PostgreSQL..."
        
        # 检查外部 PostgreSQL 连接
        if command -v psql >/dev/null 2>&1; then
            echo "✅ Found system PostgreSQL"
            export DATABASE_URL="postgresql://kimbap:kimbap123@localhost:5432/kimbap_db"
            
            # 测试连接
            if psql "$DATABASE_URL" -c "SELECT 1;" >/dev/null 2>&1; then
                echo "✅ Database connection successful"
                
                # 运行数据库迁移
                cd "$SCRIPT_DIR/app"
                ../node/bin/node node_modules/.bin/prisma migrate deploy
                
                return 0
            else
                echo "❌ Cannot connect to database"
                echo "🔧 Please ensure PostgreSQL is running with the correct credentials"
                echo "📋 Database URL: $DATABASE_URL"
                return 1
            fi
        else
            echo "❌ No PostgreSQL found!"
            echo ""
            echo "📖 Please set up PostgreSQL using one of these options:"
            echo ""
            echo "1. Docker (Recommended):"
            echo "   docker run --name kimbap-postgres -e POSTGRES_USER=kimbap -e POSTGRES_PASSWORD=kimbap123 -e POSTGRES_DB=kimbap_db -p 5432:5432 -d postgres:16"
            echo ""
            echo "2. Homebrew:"
            echo "   brew install postgresql@16"
            echo "   brew services start postgresql@16"
            echo "   createdb kimbap_db"
            echo ""
            echo "3. See postgresql/SETUP.md for detailed instructions"
            echo ""
            return 1
        fi
    fi
}

# 清理函数
cleanup() {
    echo ""
    echo "🛑 Stopping services..."
    
    # 如果使用了内置 PostgreSQL，停止它
    if [ -f "$SCRIPT_DIR/postgresql/bin/postgres" ] && [ -d "$PGDATA" ]; then
        pg_ctl -D "$PGDATA" stop -m fast >/dev/null 2>&1
        echo "✅ PostgreSQL stopped"
    fi
    
    echo "✅ Kimbap Console stopped."
}

# 设置退出时的清理
trap cleanup EXIT INT TERM

# 设置 PostgreSQL
if ! setup_postgresql; then
    echo ""
    echo "❌ PostgreSQL setup failed. Please resolve database issues and try again."
    exit 1
fi

# 启动应用
echo "🚀 Starting Kimbap Console..."
cd "$SCRIPT_DIR/app"
../node/bin/node node_modules/next/dist/bin/next start -p 3000`;

    const scriptPath = path.join(this.outputDir, 'scripts', 'start.sh');
    fs.writeFileSync(scriptPath, script);
    
    // 添加执行权限
    fs.chmodSync(scriptPath, 0o755);
  }

  async createConfigFiles() {
    console.log('📄 Creating configuration files...');
    
    const config = {
      app: {
        name: 'Kimbap Console',
        version: '1.0.0',
        port: 3000,
        host: 'localhost'
      },
      database: {
        host: 'localhost',
        port: 5432,
        database: 'kimbap_db',
        username: 'kimbap',
        password: 'kimbap123'
      },
      paths: {
        data: './postgresql/data',
        logs: './logs',
        backups: './backups'
      },
      features: {
        autoStart: true,
        autoUpdate: false,
        telemetry: false
      }
    };
    
    fs.writeFileSync(
      path.join(this.outputDir, 'config', 'config.json'),
      JSON.stringify(config, null, 2)
    );
    
    console.log('✅ Configuration files created');
  }

  async createDocumentation() {
    console.log('📚 Creating documentation...');
    
    const readme = `# Kimbap Console 便携版

## 快速开始

### Windows
1. 双击 \`scripts/start.bat\` 启动应用
2. 等待初始化完成（首次运行需要几分钟）
3. 浏览器会自动打开 http://localhost:3000

### Mac/Linux
1. 打开终端，进入应用目录
2. 运行 \`./scripts/start.sh\`
3. 等待初始化完成（首次运行需要几分钟）
4. 在浏览器中打开 http://localhost:3000

## 系统要求

- **内存**: 最少 2GB RAM
- **存储**: 最少 500MB 可用空间
- **操作系统**: 
  - Windows 10 或更新版本
  - macOS 10.14 或更新版本
  - Ubuntu 18.04 或更新版本

## 目录结构

\`\`\`
kimbap-console/
├── app/                # 应用文件
├── postgresql/         # PostgreSQL 数据库
├── node/              # Node.js 运行时
├── scripts/           # 启动脚本
├── config/            # 配置文件
├── logs/              # 日志文件
└── README.txt         # 本说明文件
\`\`\`

## 常见问题

### 1. 端口 3000 被占用
- 关闭占用端口 3000 的其他应用
- 或修改 config/config.json 中的端口设置

### 2. 数据库启动失败
- 确保有足够的磁盘空间
- 检查 logs/postgresql.log 中的错误信息

### 3. 应用无法访问
- 确认防火墙设置允许本地连接
- 检查 logs/ 目录下的日志文件

## 数据备份

应用数据存储在 \`postgresql/data\` 目录中，建议定期备份此目录。

## 卸载

直接删除整个应用目录即可完全卸载。

## 技术支持

如遇问题，请查看：
1. logs/ 目录下的日志文件
2. 访问项目 GitHub 页面提交 Issue
3. 联系技术支持团队

---

Kimbap Console v1.0.0
构建日期: ${new Date().toISOString().split('T')[0]}
平台: ${this.platform}-${this.arch}`;

    fs.writeFileSync(path.join(this.outputDir, 'README.txt'), readme);
    
    console.log('✅ Documentation created');
  }

  // 辅助方法
  getNodeDownloadUrl() {
    const baseUrl = 'https://nodejs.org/dist';
    const fileName = this.platform === 'win32' 
      ? `node-v${this.nodeVersion}-win-${this.arch}.zip`
      : `node-v${this.nodeVersion}-${this.platform}-${this.arch}.tar.xz`;
    
    return `${baseUrl}/v${this.nodeVersion}/${fileName}`;
  }

  getPostgreSQLDownloadUrl() {
    // 这里需要根据实际情况配置 PostgreSQL 下载链接
    // 可以使用预编译的二进制包或便携版
    const baseUrl = 'https://get.enterprisedb.com/postgresql';
    const fileName = this.platform === 'win32'
      ? `postgresql-${this.postgresVersion}-windows-${this.arch}-binaries.zip`
      : `postgresql-${this.postgresVersion}-${this.platform}-${this.arch}.tar.gz`;
    
    return `${baseUrl}/${fileName}`;
  }

  getArchiveExtension() {
    return this.platform === 'win32' ? 'zip' : 'tar.xz';
  }

  async downloadFile(url, destination) {
    return new Promise((resolve, reject) => {
      const file = fs.createWriteStream(destination);
      
      https.get(url, (response) => {
        if (response.statusCode === 200) {
          response.pipe(file);
          file.on('finish', () => {
            file.close();
            resolve();
          });
        } else if (response.statusCode === 302 || response.statusCode === 301) {
          // 处理重定向
          file.close();
          fs.unlinkSync(destination);
          this.downloadFile(response.headers.location, destination)
            .then(resolve)
            .catch(reject);
        } else {
          file.close();
          fs.unlinkSync(destination);
          reject(new Error(`HTTP ${response.statusCode}: ${response.statusMessage}`));
        }
      }).on('error', (err) => {
        file.close();
        fs.unlinkSync(destination);
        reject(err);
      });
    });
  }

  async extractArchive(archive, destination) {
    // 这里需要实现解压缩逻辑
    // 可以使用 node 的 zlib 或第三方库如 yauzl, tar 等
    console.log(`Extracting ${archive} to ${destination}`);
    
    // 临时实现：提示手动解压
    console.log('⚠️  Please manually extract the archive for now');
    console.log(`From: ${archive}`);
    console.log(`To: ${destination}`);
  }

  copyDir(src, dest) {
    if (!fs.existsSync(src)) {
      console.warn(`⚠️  Source directory not found: ${src}`);
      return;
    }
    
    fs.mkdirSync(dest, { recursive: true });
    
    const entries = fs.readdirSync(src, { withFileTypes: true });
    
    for (const entry of entries) {
      const srcPath = path.join(src, entry.name);
      const destPath = path.join(dest, entry.name);
      
      if (entry.isDirectory()) {
        this.copyDir(srcPath, destPath);
      } else {
        fs.copyFileSync(srcPath, destPath);
      }
    }
  }

  getDirectorySize(dir) {
    let size = 0;
    
    if (!fs.existsSync(dir)) return 0;
    
    const walk = (currentPath) => {
      const entries = fs.readdirSync(currentPath, { withFileTypes: true });
      
      for (const entry of entries) {
        const fullPath = path.join(currentPath, entry.name);
        
        if (entry.isDirectory()) {
          walk(fullPath);
        } else {
          size += fs.statSync(fullPath).size;
        }
      }
    };
    
    walk(dir);
    return Math.round(size / 1024 / 1024); // MB
  }

  async cleanup() {
    console.log('🧹 Cleaning up temporary files...');
    
    if (fs.existsSync(this.tempDir)) {
      fs.rmSync(this.tempDir, { recursive: true, force: true });
    }
    
    console.log('✅ Cleanup completed');
  }
}

// 执行构建
if (require.main === module) {
  const builder = new PortableBuilder();
  builder.build();
}

module.exports = PortableBuilder;