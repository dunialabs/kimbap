#!/usr/bin/env node

/**
 * 完成便携包构建 - 跳过已完成的部分
 * 只执行剩余的步骤：复制应用文件、下载PostgreSQL、创建启动脚本等
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const DependencyDownloader = require('./download-dependencies');
const ExtractUtils = require('./extract-utils');

class BuildCompleter {
  constructor() {
    this.rootDir = path.resolve(__dirname, '../..');
    this.platform = process.platform;
    this.arch = process.arch;
    this.outputDir = path.join(this.rootDir, 'dist', `kimbap-console-${this.platform}`);
    
    console.log(`🚀 Completing build for ${this.platform}-${this.arch}`);
  }

  async complete() {
    try {
      console.log('📋 Copying application files...');
      await this.copyAppFiles();
      
      console.log('📥 Setting up PostgreSQL...');
      await this.setupPostgreSQL();
      
      console.log('📝 Creating startup scripts...');
      await this.createStartupScripts();
      
      console.log('📄 Creating configuration files...');
      await this.createConfigFiles();
      
      console.log('📚 Creating documentation...');
      await this.createDocumentation();
      
      console.log(`✅ Build completed successfully!`);
      console.log(`📁 Output directory: ${this.outputDir}`);
      
    } catch (error) {
      console.error('❌ Build completion failed:', error);
      process.exit(1);
    }
  }

  async copyAppFiles() {
    const appDir = path.join(this.outputDir, 'app');
    
    // 复制构建文件
    console.log('  📦 Copying .next build output...');
    this.copyDir(path.join(this.rootDir, '.next'), path.join(appDir, '.next'));
    
    console.log('  📁 Copying public assets...');
    this.copyDir(path.join(this.rootDir, 'public'), path.join(appDir, 'public'));
    
    console.log('  🗄️  Copying prisma schema...');
    this.copyDir(path.join(this.rootDir, 'prisma'), path.join(appDir, 'prisma'));
    
    // 复制所有 node_modules (简化方法)
    console.log('  📚 Copying node_modules (this may take a while)...');
    this.copyDir(path.join(this.rootDir, 'node_modules'), path.join(appDir, 'node_modules'));
    
    // 复制配置文件
    console.log('  ⚙️  Copying configuration files...');
    fs.copyFileSync(path.join(this.rootDir, 'package.json'), path.join(appDir, 'package.json'));
    
    // 创建适合便携包的 next.config.mjs
    const portableNextConfig = `/** @type {import('next').NextConfig} */
const nextConfig = {
  // 便携包配置 - 支持API routes
  output: undefined,

  // 图片配置
  images: {
    unoptimized: true,
    formats: ['image/webp', 'image/avif'],
    minimumCacheTTL: 60,
    dangerouslyAllowSVG: true,
    contentSecurityPolicy: "default-src 'self'; script-src 'none'; sandbox;"
  },

  // 实验性功能
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
  poweredByHeader: false
}

export default nextConfig`;

    fs.writeFileSync(path.join(appDir, 'next.config.mjs'), portableNextConfig);
    
    // 创建生产环境配置
    const prodEnv = `NODE_ENV=production
DATABASE_URL=postgresql://kimbap:kimbap123@localhost:5432/kimbap_db`;
    
    fs.writeFileSync(path.join(appDir, '.env.local'), prodEnv);
    
    console.log('✅ Application files copied');
  }

  async setupPostgreSQL() {
    console.log('🔍 Checking for PostgreSQL installations...');
    
    try {
      // 检查系统 PostgreSQL
      const systemPgPath = this.findSystemPostgreSQL();
      
      if (systemPgPath) {
        console.log(`✅ Found system PostgreSQL at: ${systemPgPath}`);
        await this.copySystemPostgreSQL(systemPgPath);
        await this.createDatabaseInitScript();
        console.log('✅ PostgreSQL setup completed with embedded binaries');
      } else {
        console.log('📝 No system PostgreSQL found, creating setup instructions');
        await this.createPostgreSQLInstructions();
        console.log('✅ PostgreSQL setup instructions created');
      }
    } catch (error) {
      console.warn('⚠️  PostgreSQL setup failed:', error.message);
      await this.createPostgreSQLInstructions();
    }
  }

  findSystemPostgreSQL() {
    const possiblePaths = [
      '/opt/homebrew/bin/postgres',        // Homebrew on Apple Silicon
      '/usr/local/bin/postgres',           // Homebrew on Intel Mac
      '/usr/bin/postgres',                 // System install
      '/Applications/Postgres.app/Contents/Versions/16/bin/postgres'  // Postgres.app
    ];

    for (const pgPath of possiblePaths) {
      try {
        if (fs.existsSync(pgPath)) {
          console.log(`🔍 Checking: ${pgPath}`);
          return path.dirname(pgPath);
        }
      } catch (error) {
        // 忽略权限错误，继续检查其他路径
      }
    }
    
    // 尝试使用 which 命令
    try {
      const { execSync } = require('child_process');
      const result = execSync('which postgres', { encoding: 'utf8', stdio: 'pipe' });
      const pgPath = result.trim();
      if (pgPath && fs.existsSync(pgPath)) {
        console.log(`🔍 Found via which: ${pgPath}`);
        return path.dirname(pgPath);
      }
    } catch (error) {
      // which 命令失败或 postgres 不在 PATH 中
    }
    
    return null;
  }

  async copySystemPostgreSQL(systemPgPath) {
    console.log('📋 Copying system PostgreSQL binaries...');
    
    const pgDir = path.join(this.outputDir, 'postgresql');
    fs.mkdirSync(pgDir, { recursive: true });
    
    // 需要的 PostgreSQL 二进制文件
    const requiredBinaries = [
      'postgres',
      'initdb', 
      'pg_ctl',
      'createdb',
      'createuser',
      'psql',
      'pg_dump',
      'pg_restore'
    ];

    const binDir = path.join(pgDir, 'bin');
    fs.mkdirSync(binDir, { recursive: true });

    let copiedFiles = 0;
    for (const binary of requiredBinaries) {
      const srcPath = path.join(systemPgPath, binary);
      const destPath = path.join(binDir, binary);
      
      if (fs.existsSync(srcPath)) {
        try {
          fs.copyFileSync(srcPath, destPath);
          fs.chmodSync(destPath, 0o755);
          copiedFiles++;
          console.log(`  ✅ ${binary}`);
        } catch (error) {
          console.warn(`  ⚠️  Failed to copy ${binary}: ${error.message}`);
        }
      } else {
        console.warn(`  ❌ ${binary} not found`);
      }
    }

    console.log(`✅ Copied ${copiedFiles}/${requiredBinaries.length} PostgreSQL binaries`);

    // 尝试复制共享库
    await this.copyPostgreSQLLibraries(systemPgPath, pgDir);
  }

  async copyPostgreSQLLibraries(systemPgPath, pgDir) {
    console.log('📚 Copying PostgreSQL libraries...');
    
    const possibleLibDirs = [
      path.join(path.dirname(systemPgPath), 'lib'),
      path.join(path.dirname(path.dirname(systemPgPath)), 'lib')
    ];

    for (const libDir of possibleLibDirs) {
      if (fs.existsSync(libDir)) {
        try {
          const libFiles = fs.readdirSync(libDir).filter(f => 
            f.includes('postgres') || f.startsWith('libpq') || f.includes('pq')
          );
          
          if (libFiles.length > 0) {
            const destLibDir = path.join(pgDir, 'lib');
            fs.mkdirSync(destLibDir, { recursive: true });
            
            let copiedLibs = 0;
            for (const libFile of libFiles.slice(0, 15)) { // 限制复制数量
              try {
                const srcPath = path.join(libDir, libFile);
                const destPath = path.join(destLibDir, libFile);
                fs.copyFileSync(srcPath, destPath);
                copiedLibs++;
              } catch (error) {
                // 忽略复制失败的库文件
              }
            }
            
            if (copiedLibs > 0) {
              console.log(`✅ Copied ${copiedLibs} PostgreSQL libraries`);
            }
            break;
          }
        } catch (error) {
          // 忽略目录读取错误
        }
      }
    }
  }

  async createDatabaseInitScript() {
    console.log('📝 Creating database initialization script...');
    
    const scriptsDir = path.join(this.outputDir, 'scripts');
    const scriptName = 'init-db.sh';
    const scriptPath = path.join(scriptsDir, scriptName);
    
    const initScript = `#!/bin/bash
echo "🔧 Initializing PostgreSQL database..."

SCRIPT_DIR="$( cd "$( dirname "\${BASH_SOURCE[0]}" )" && pwd )"
PGROOT="$SCRIPT_DIR/../postgresql"
PGDATA="$PGROOT/data"
PGBIN="$PGROOT/bin"

# 检查 PostgreSQL 二进制文件
if [ ! -f "$PGBIN/postgres" ]; then
    echo "❌ PostgreSQL binaries not found!"
    echo "📝 Please see postgresql/SETUP.md for installation instructions"
    exit 1
fi

if [ -d "$PGDATA/base" ]; then
    echo "✅ Database already initialized"
    exit 0
fi

echo "📁 Creating data directory..."
mkdir -p "$PGDATA"

echo "🗄️  Initializing database cluster..."
"$PGBIN/initdb" -D "$PGDATA" -U kimbap --auth-local=trust --auth-host=md5

if [ $? -ne 0 ]; then
    echo "❌ Database initialization failed!"
    exit 1
fi

echo "🚀 Starting PostgreSQL..."
"$PGBIN/pg_ctl" -D "$PGDATA" -l "$PGROOT/postgresql.log" start

echo "⏳ Waiting for database to start..."
sleep 5

echo "🗃️  Creating application database and user..."
"$PGBIN/createdb" -U kimbap kimbap_db

if [ $? -ne 0 ]; then
    echo "❌ Failed to create database!"
    exit 1
fi

echo "✅ Database initialization completed!"
echo "🎯 Database is ready at: postgresql://kimbap@localhost:5432/kimbap_db"
`;
    
    fs.writeFileSync(scriptPath, initScript);
    fs.chmodSync(scriptPath, 0o755);
    
    console.log(`✅ Created database initialization script: ${scriptName}`);
  }

  async createPostgreSQLInstructions() {
    const instructions = `# PostgreSQL Setup for KIMBAP Console

## 自动检测结果
系统中未找到可用的 PostgreSQL 安装。

## 推荐设置方案

### 选项1: 使用 Docker (最简单) ⭐
\`\`\`bash
docker run --name kimbap-postgres \\
  -e POSTGRES_USER=kimbap \\
  -e POSTGRES_PASSWORD=kimbap123 \\
  -e POSTGRES_DB=kimbap_db \\
  -p 5432:5432 -d postgres:16
\`\`\`

### 选项2: 安装本地 PostgreSQL

#### macOS (推荐 Homebrew):
\`\`\`bash
# 安装 PostgreSQL
brew install postgresql@16
brew services start postgresql@16

# 创建用户和数据库
createuser kimbap
createdb -U kimbap kimbap_db
\`\`\`

#### Linux (Ubuntu/Debian):
\`\`\`bash
sudo apt update
sudo apt install postgresql-16 postgresql-client-16
sudo systemctl start postgresql

# 创建用户和数据库
sudo -u postgres createuser kimbap
sudo -u postgres createdb -O kimbap kimbap_db
\`\`\`

#### Windows:
1. 从 https://www.postgresql.org/download/windows/ 下载 PostgreSQL 16
2. 安装并记住设置的密码
3. 使用 pgAdmin 或命令行创建用户 \`kimbap\` 和数据库 \`kimbap_db\`

### 选项3: 使用现有数据库实例
编辑 \`app/.env.local\` 文件中的 DATABASE_URL：
\`\`\`
DATABASE_URL=postgresql://username:password@host:port/database
\`\`\`

## 验证设置
运行以下命令验证数据库连接:
\`\`\`bash
psql "postgresql://kimbap:kimbap123@localhost:5432/kimbap_db" -c "SELECT version();"
\`\`\`

## 故障排除

### 连接被拒绝
- 确保 PostgreSQL 服务正在运行
- 检查端口 5432 是否被占用
- 验证防火墙设置

### 认证失败
- 检查用户名和密码是否正确
- 确保用户 \`kimbap\` 有访问数据库 \`kimbap_db\` 的权限

### 数据库不存在
- 使用 \`createdb -U kimbap kimbap_db\` 创建数据库
- 或者修改 \`.env.local\` 中的数据库名称

---

📖 详细文档: https://www.postgresql.org/docs/
🐳 Docker 文档: https://hub.docker.com/_/postgres
`;

    const pgDir = path.join(this.outputDir, 'postgresql');
    fs.mkdirSync(pgDir, { recursive: true });
    fs.writeFileSync(path.join(pgDir, 'SETUP.md'), instructions);
  }

  async createStartupScripts() {
    if (this.platform === 'win32') {
      await this.createWindowsScript();
    } else {
      await this.createUnixScript();
    }
    
    console.log('✅ Startup scripts created');
  }

  async createWindowsScript() {
    const script = `@echo off
title KIMBAP Console
echo ========================================
echo         KIMBAP Console Starting
echo ========================================
echo.

REM 设置环境变量
set "SCRIPT_DIR=%~dp0"
set "PATH=%SCRIPT_DIR%node\\bin;%PATH%"
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

REM 检查PostgreSQL连接
echo 🔍 Checking PostgreSQL connection...
echo Make sure PostgreSQL is running on localhost:5432
echo Database: kimbap_db, User: kimbap, Password: kimbap123
echo.

REM 启动应用
echo 🚀 Starting KIMBAP Console...
cd /d "%SCRIPT_DIR%app"
..\\node\\bin\\node.exe node_modules\\next\\dist\\bin\\next start -p 3000

REM 如果应用异常退出，显示错误信息
if %errorlevel% neq 0 (
    echo.
    echo ❌ Application failed to start!
    echo Check the logs for more information.
)

echo.
echo 🛑 KIMBAP Console stopped.
pause`;

    fs.writeFileSync(path.join(this.outputDir, 'scripts', 'start.bat'), script);
  }

  async createUnixScript() {
    const script = `#!/bin/bash

# KIMBAP Console 启动脚本
echo "========================================"
echo "       KIMBAP Console Starting"
echo "========================================"
echo

# 获取脚本目录
SCRIPT_DIR="$( cd "$( dirname "\${BASH_SOURCE[0]}" )" && pwd )"
SCRIPT_DIR="$( dirname "$SCRIPT_DIR" )"

# 设置环境变量
export PATH="$SCRIPT_DIR/node/bin:$PATH"
export DATABASE_URL="postgresql://kimbap:kimbap123@localhost:5432/kimbap_db"
export NODE_ENV="production"

# 检查端口是否被占用
if lsof -Pi :3000 -sTCP:LISTEN -t >/dev/null ; then
    echo "⚠️  Port 3000 is already in use!"
    echo "Please close the application using port 3000 and try again."
    exit 1
fi

# 检查PostgreSQL连接
echo "🔍 Checking PostgreSQL connection..."
echo "Make sure PostgreSQL is running on localhost:5432"
echo "Database: kimbap_db, User: kimbap, Password: kimbap123"
echo

# 启动应用
echo "🚀 Starting KIMBAP Console..."
cd "$SCRIPT_DIR/app"
../node/bin/node node_modules/next/dist/bin/next start -p 3000

echo
echo "✅ KIMBAP Console stopped."`;

    const scriptPath = path.join(this.outputDir, 'scripts', 'start.sh');
    fs.writeFileSync(scriptPath, script);
    
    // 添加执行权限
    fs.chmodSync(scriptPath, 0o755);
  }

  async createConfigFiles() {
    const config = {
      app: {
        name: 'KIMBAP Console',
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
    const readme = `# KIMBAP Console 便携版

## 快速开始

### 前置要求
1. **PostgreSQL**: 需要运行 PostgreSQL 数据库
   - 选项1: 使用 Docker (推荐)
     \`\`\`bash
     docker run --name kimbap-postgres -e POSTGRES_USER=kimbap -e POSTGRES_PASSWORD=kimbap123 -e POSTGRES_DB=kimbap_db -p 5432:5432 -d postgres:16
     \`\`\`
   - 选项2: 安装本地 PostgreSQL 并创建数据库 kimbap_db

### Windows
1. 确保 PostgreSQL 正在运行
2. 双击 \`scripts/start.bat\` 启动应用
3. 浏览器会自动打开 http://localhost:3000

### Mac/Linux
1. 确保 PostgreSQL 正在运行
2. 打开终端，进入应用目录
3. 运行 \`./scripts/start.sh\`
4. 在浏览器中打开 http://localhost:3000

## 系统要求

- **内存**: 最少 2GB RAM
- **存储**: 最少 500MB 可用空间
- **数据库**: PostgreSQL 16.x (通过 Docker 或本地安装)
- **操作系统**: 
  - Windows 10 或更新版本
  - macOS 10.14 或更新版本
  - Ubuntu 18.04 或更新版本

## 目录结构

\`\`\`
kimbap-console/
├── app/                # 应用文件 (.next, node_modules, etc.)
├── node/              # Node.js 运行时
├── postgresql/         # PostgreSQL 说明文档
├── scripts/           # 启动脚本
├── config/            # 配置文件
└── README.txt         # 本说明文件
\`\`\`

## 常见问题

### 1. 端口 3000 被占用
- 关闭占用端口 3000 的其他应用
- 或修改 config/config.json 中的端口设置

### 2. 数据库连接失败
- 确保 PostgreSQL 正在运行在 localhost:5432
- 检查数据库用户名密码是否为 kimbap/kimbap123
- 确保数据库 kimbap_db 存在

### 3. 应用无法访问
- 确认防火墙设置允许本地连接
- 检查终端输出的错误信息

## 数据备份

如果使用 Docker PostgreSQL，可以通过以下命令备份数据：
\`\`\`bash
docker exec kimbap-postgres pg_dump -U kimbap kimbap_db > backup.sql
\`\`\`

## 卸载

1. 停止 PostgreSQL 容器 (如果使用 Docker)：\`docker stop kimbap-postgres && docker rm kimbap-postgres\`
2. 删除整个应用目录即可完全卸载

## 技术支持

如遇问题，请查看：
1. 终端输出的错误信息
2. 访问项目 GitHub 页面提交 Issue
3. 联系技术支持团队

---

KIMBAP Console v1.0.0
构建日期: ${new Date().toISOString().split('T')[0]}
平台: ${this.platform}-${this.arch}`;

    fs.writeFileSync(path.join(this.outputDir, 'README.txt'), readme);
    
    console.log('✅ Documentation created');
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
}

// 执行构建完成
if (require.main === module) {
  const completer = new BuildCompleter();
  completer.complete();
}

module.exports = BuildCompleter;