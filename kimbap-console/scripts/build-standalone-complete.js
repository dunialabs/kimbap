#!/usr/bin/env node

/**
 * 完整的独立部署包构建脚本
 * 自动处理所有依赖和配置
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');
const https = require('https');
const http = require('http');
const { promisify } = require('util');
const { pipeline } = require('stream');
const zlib = require('zlib');

class CompleteStandaloneBuilder {
  constructor(targetArch = null, targetPlatform = null) {
    this.rootDir = process.cwd();
    this.platform = targetPlatform || process.platform;
    this.arch = targetArch || process.arch;
    this.outputDir = path.join(this.rootDir, 'dist', `kimbap-console-${this.platform}-${this.arch}`);
    this.tempDir = path.join(this.rootDir, '.temp-build');
    
    console.log(`🚀 Building complete standalone package for ${this.platform}-${this.arch}`);
    
    // PostgreSQL 下载配置 - 使用官方便携式版本
    this.pgDownloadUrls = {
      'darwin-x64': 'https://sbp.enterprisedb.com/getfile.jsp?fileid=1258893',
      'darwin-arm64': 'https://sbp.enterprisedb.com/getfile.jsp?fileid=1258893',
      'linux-x64': 'https://sbp.enterprisedb.com/getfile.jsp?fileid=1258749',
      'win32-x64': 'https://sbp.enterprisedb.com/getfile.jsp?fileid=1258796'
    };
  }

  // 下载文件的工具方法
  async downloadFile(url, destPath) {
    const pipelineAsync = promisify(pipeline);
    
    return new Promise((resolve, reject) => {
      const request = url.startsWith('https') ? https : http;
      
      console.log(`📥 Downloading from ${url}...`);
      
      request.get(url, (response) => {
        if (response.statusCode === 302 || response.statusCode === 301) {
          // 处理重定向
          return this.downloadFile(response.headers.location, destPath).then(resolve).catch(reject);
        }
        
        if (response.statusCode !== 200) {
          reject(new Error(`下载失败: ${response.statusCode}`));
          return;
        }
        
        const fileStream = fs.createWriteStream(destPath);
        const totalBytes = parseInt(response.headers['content-length'], 10);
        let downloadedBytes = 0;
        
        response.on('data', (chunk) => {
          downloadedBytes += chunk.length;
          if (totalBytes) {
            const progress = ((downloadedBytes / totalBytes) * 100).toFixed(1);
            process.stdout.write(`\r📊 Progress: ${progress}%`);
          }
        });
        
        pipelineAsync(response, fileStream)
          .then(() => {
            console.log('\n✅ Download completed');
            resolve();
          })
          .catch(reject);
      }).on('error', reject);
    });
  }

  // 解压文件的工具方法
  async extractArchive(archivePath, destDir) {
    const archiveExt = path.extname(archivePath);
    
    console.log(`📦 Extracting ${archivePath}...`);
    
    if (archiveExt === '.zip') {
      // 使用 node 自带的能力或系统命令解压 ZIP
      try {
        execSync(`unzip -q "${archivePath}" -d "${destDir}"`, { stdio: 'inherit' });
      } catch (error) {
        throw new Error(`解压 ZIP 失败: ${error.message}`);
      }
    } else if (archivePath.endsWith('.tar.gz')) {
      // 解压 tar.gz
      try {
        execSync(`tar -xzf "${archivePath}" -C "${destDir}"`, { stdio: 'inherit' });
      } catch (error) {
        throw new Error(`解压 tar.gz 失败: ${error.message}`);
      }
    }
    
    console.log('✅ Extraction completed');
  }

  async build() {
    try {
      // 1. 编译后端
      await this.buildBackend();
      
      // 2. 构建前端（standalone模式）
      await this.buildFrontend();
      
      // 3. 准备输出目录
      await this.prepareOutput();
      
      // 4. 设置Node.js运行时
      await this.setupNodeRuntime();
      
      // 5. 设置PostgreSQL便携版
      await this.setupPostgreSQL();
      
      // 6. 复制应用文件
      await this.copyApplicationFiles();
      
      // 7. 修复路径和配置
      await this.fixPathsAndConfigs();
      
      // 8. 创建启动脚本
      await this.createStartupScripts();
      
      // 9. 创建文档
      await this.createDocumentation();
      
      // 10. 创建平台特定的可执行文件
      await this.createExecutables();
      
      console.log(`✅ Build completed successfully!`);
      console.log(`📁 Output: ${this.outputDir}`);
      
    } catch (error) {
      console.error('❌ Build failed:', error.message);
      process.exit(1);
    }
  }

  async buildBackend() {
    console.log('🔨 Building backend...');
    execSync('npm run build:backend', { stdio: 'inherit' });
  }

  async buildFrontend() {
    console.log('🔨 Building frontend (standalone mode)...');
    
    // 确保使用standalone配置
    const configPath = path.join(this.rootDir, 'next.config.mjs');
    let config = fs.readFileSync(configPath, 'utf8');
    
    // 临时确保output为standalone
    if (!config.includes("output: 'standalone'")) {
      config = config.replace(
        '/** @type {import(\'next\').NextConfig} */\nconst nextConfig = {',
        '/** @type {import(\'next\').NextConfig} */\nconst nextConfig = {\n  output: \'standalone\','
      );
      fs.writeFileSync(configPath, config);
    }
    
    execSync('npm run build:frontend', { stdio: 'inherit' });
  }

  async prepareOutput() {
    console.log('📁 Preparing output directory...');
    
    if (fs.existsSync(this.outputDir)) {
      fs.rmSync(this.outputDir, { recursive: true, force: true });
    }
    
    fs.mkdirSync(this.outputDir, { recursive: true });
    fs.mkdirSync(path.join(this.outputDir, 'app'), { recursive: true });
  }

  async setupNodeRuntime() {
    console.log('📥 Setting up Node.js runtime...');
    
    const nodeDir = path.join(this.outputDir, 'node');
    const tempNodePath = path.join(this.rootDir, 'temp', `node-v20.11.0-${this.platform}-${this.arch}.tar.gz`);
    
    // 如果缓存存在，使用缓存
    if (fs.existsSync(tempNodePath)) {
      console.log('✅ Using cached Node.js');
      await this.extractNode(tempNodePath, nodeDir);
    } else {
      console.log('📥 Downloading Node.js...');
      fs.mkdirSync(path.dirname(tempNodePath), { recursive: true });
      await this.downloadNode(tempNodePath);
      await this.extractNode(tempNodePath, nodeDir);
    }
  }

  async setupPostgreSQL() {
    console.log('🐘 Setting up embedded PostgreSQL...');
    
    const pgDir = path.join(this.outputDir, 'postgresql');
    fs.mkdirSync(pgDir, { recursive: true });
    
    // 根据平台下载便携式PostgreSQL
    if (this.platform === 'darwin') {
      await this.setupPostgreSQLMac(pgDir);
    } else if (this.platform === 'linux') {
      await this.setupPostgreSQLLinux(pgDir);
    } else if (this.platform === 'win32') {
      await this.setupPostgreSQLWindows(pgDir);
    }
  }

  async setupPostgreSQLMac(pgDir) {
    console.log('📥 Setting up PostgreSQL for macOS...');
    
    // 首先尝试本地 PostgreSQL 安装路径
    const pgPaths = [
      '/opt/homebrew/opt/postgresql@16',
      '/opt/homebrew/opt/postgresql@15',
      '/usr/local/opt/postgresql@16', 
      '/usr/local/opt/postgresql@15',
      '/opt/homebrew/opt/postgresql',
      '/usr/local/opt/postgresql'
    ];
    
    let pgFound = false;
    for (const localPgPath of pgPaths) {
      if (fs.existsSync(localPgPath)) {
        console.log(`✅ Using local PostgreSQL from: ${localPgPath}`);
        pgFound = await this.copyLocalPostgreSQL(localPgPath, pgDir);
        if (pgFound) break;
      }
    }
    
    // 如果没有找到本地安装，自动下载
    if (!pgFound) {
      console.log('🔍 Local PostgreSQL not found, downloading portable version...');
      pgFound = await this.downloadPostgreSQL(pgDir);
    }
    
    if (!pgFound) {
      console.log('⚠️  Failed to setup PostgreSQL. Standalone package will require Docker or system PostgreSQL.');
      // 创建占位目录
      fs.mkdirSync(path.join(pgDir, 'bin'), { recursive: true });
    }
    
    // 创建数据目录和启动脚本
    fs.mkdirSync(path.join(pgDir, 'data'), { recursive: true });
    this.createPortablePostgreSQLScripts(pgDir);
  }

  // 复制本地 PostgreSQL 安装
  async copyLocalPostgreSQL(localPgPath, pgDir) {
    try {
      const essentialBins = [
        'postgres', 'pg_ctl', 'initdb', 'createdb', 'createuser', 
        'psql', 'pg_isready', 'pg_dump', 'pg_restore'
      ];
      
      fs.mkdirSync(path.join(pgDir, 'bin'), { recursive: true });
      
      for (const bin of essentialBins) {
        const srcPath = path.join(localPgPath, 'bin', bin);
        if (fs.existsSync(srcPath)) {
          const destPath = path.join(pgDir, 'bin', bin);
          fs.copyFileSync(srcPath, destPath);
          fs.chmodSync(destPath, 0o755);
        }
      }
      
      // 复制必要的库文件
      const libSrcPath = path.join(localPgPath, 'lib');
      const libDestPath = path.join(pgDir, 'lib');
      if (fs.existsSync(libSrcPath)) {
        fs.mkdirSync(libDestPath, { recursive: true });
        
        const libFiles = fs.readdirSync(libSrcPath).filter(file => 
          file.startsWith('libpq') || 
          file.startsWith('libpgcommon') || 
          file.startsWith('libpgport')
        );
        
        for (const libFile of libFiles) {
          const srcPath = path.join(libSrcPath, libFile);
          const destPath = path.join(libDestPath, libFile);
          if (fs.statSync(srcPath).isFile()) {
            fs.copyFileSync(srcPath, destPath);
          }
        }
      }
      
      // 复制 share 文件
      const shareSrcPath = path.join(localPgPath, 'share', 'postgresql');
      if (fs.existsSync(shareSrcPath)) {
        this.copyDir(shareSrcPath, path.join(pgDir, 'share', 'postgresql'));
      }
      
      return true;
    } catch (error) {
      console.error('❌ Error copying local PostgreSQL:', error.message);
      return false;
    }
  }

  // 下载并安装 PostgreSQL
  async downloadPostgreSQL(pgDir) {
    const platformKey = `${this.platform}-${this.arch}`;
    const downloadUrl = this.pgDownloadUrls[platformKey];
    
    if (!downloadUrl) {
      console.error(`❌ No PostgreSQL download URL for platform: ${platformKey}`);
      return false;
    }
    
    try {
      // 确保临时目录存在
      fs.mkdirSync(this.tempDir, { recursive: true });
      
      const archiveExt = downloadUrl.endsWith('.zip') ? '.zip' : '.tar.gz';
      const archivePath = path.join(this.tempDir, `postgresql-${platformKey}${archiveExt}`);
      
      // 如果已经下载过，跳过下载
      if (!fs.existsSync(archivePath)) {
        await this.downloadFile(downloadUrl, archivePath);
      } else {
        console.log('✅ Using cached PostgreSQL archive');
      }
      
      // 解压到临时目录
      const extractDir = path.join(this.tempDir, `postgresql-${platformKey}`);
      if (fs.existsSync(extractDir)) {
        fs.rmSync(extractDir, { recursive: true, force: true });
      }
      fs.mkdirSync(extractDir, { recursive: true });
      
      await this.extractArchive(archivePath, extractDir);
      
      // 查找 PostgreSQL 二进制文件目录
      const pgBinaryDir = this.findPostgreSQLBinaries(extractDir);
      if (!pgBinaryDir) {
        throw new Error('PostgreSQL binaries not found in downloaded archive');
      }
      
      console.log(`📋 Copying PostgreSQL binaries from ${pgBinaryDir}...`);
      
      // 复制二进制文件
      const destBinDir = path.join(pgDir, 'bin');
      fs.mkdirSync(destBinDir, { recursive: true });
      
      if (fs.existsSync(path.join(pgBinaryDir, 'bin'))) {
        execSync(`cp -r "${path.join(pgBinaryDir, 'bin')}"/* "${destBinDir}"/`, { stdio: 'inherit' });
      } else {
        execSync(`cp -r "${pgBinaryDir}"/* "${destBinDir}"/`, { stdio: 'inherit' });
      }
      
      // 复制库文件（如果存在）
      const libSrcDir = path.join(pgBinaryDir, '..', 'lib');
      if (fs.existsSync(libSrcDir)) {
        const destLibDir = path.join(pgDir, 'lib');
        fs.mkdirSync(destLibDir, { recursive: true });
        execSync(`cp -r "${libSrcDir}"/* "${destLibDir}"/`, { stdio: 'inherit' });
      }
      
      // 设置执行权限
      execSync(`chmod +x "${destBinDir}"/*`, { stdio: 'inherit' });
      
      console.log('✅ PostgreSQL downloaded and installed successfully');
      return true;
      
    } catch (error) {
      console.error('❌ Error downloading PostgreSQL:', error.message);
      return false;
    }
  }

  // 在解压的目录中查找 PostgreSQL 二进制文件
  findPostgreSQLBinaries(baseDir) {
    const possiblePaths = [
      path.join(baseDir, 'pgsql', 'bin'),
      path.join(baseDir, 'postgresql', 'bin'),
      path.join(baseDir, 'bin'),
      path.join(baseDir, 'usr', 'local', 'pgsql', 'bin'),
    ];
    
    // 递归查找包含 postgres 可执行文件的目录
    const findPostgres = (dir) => {
      try {
        const items = fs.readdirSync(dir, { withFileTypes: true });
        
        for (const item of items) {
          const fullPath = path.join(dir, item.name);
          
          if (item.isDirectory()) {
            const result = findPostgres(fullPath);
            if (result) return result;
          } else if (item.name === 'postgres' || item.name === 'postgres.exe') {
            return dir;
          }
        }
      } catch (error) {
        // 忽略权限错误等
      }
      return null;
    };
    
    // 首先检查常见路径
    for (const pgPath of possiblePaths) {
      if (fs.existsSync(pgPath)) {
        const postgresPath = path.join(pgPath, this.platform === 'win32' ? 'postgres.exe' : 'postgres');
        if (fs.existsSync(postgresPath)) {
          return pgPath;
        }
      }
    }
    
    // 如果常见路径没找到，递归搜索
    return findPostgres(baseDir);
  }

  async setupPostgreSQLLinux(pgDir) {
    console.log('📥 Setting up PostgreSQL for Linux...');
    
    // 先尝试系统 PostgreSQL
    const pgPaths = [
      '/usr/lib/postgresql/16',
      '/usr/lib/postgresql/15', 
      '/usr/local/pgsql',
      '/opt/postgresql'
    ];
    
    let pgFound = false;
    for (const localPgPath of pgPaths) {
      if (fs.existsSync(path.join(localPgPath, 'bin', 'postgres'))) {
        console.log(`✅ Using local PostgreSQL from: ${localPgPath}`);
        pgFound = await this.copyLocalPostgreSQL(localPgPath, pgDir);
        if (pgFound) break;
      }
    }
    
    // 如果没有找到本地安装，自动下载
    if (!pgFound) {
      console.log('🔍 Local PostgreSQL not found, downloading portable version...');
      pgFound = await this.downloadPostgreSQL(pgDir);
    }
    
    if (!pgFound) {
      console.log('⚠️  Failed to setup PostgreSQL. Standalone package will require Docker or system PostgreSQL.');
      fs.mkdirSync(path.join(pgDir, 'bin'), { recursive: true });
    }
    
    fs.mkdirSync(path.join(pgDir, 'data'), { recursive: true });
    this.createPortablePostgreSQLScripts(pgDir);
  }

  async setupPostgreSQLWindows(pgDir) {
    console.log('📥 Setting up PostgreSQL for Windows...');
    
    // 先尝试系统 PostgreSQL
    const pgPaths = [
      'C:\\Program Files\\PostgreSQL\\16',
      'C:\\Program Files\\PostgreSQL\\15',
      'C:\\PostgreSQL\\16',
      'C:\\PostgreSQL\\15'
    ];
    
    let pgFound = false;
    for (const localPgPath of pgPaths) {
      if (fs.existsSync(path.join(localPgPath, 'bin', 'postgres.exe'))) {
        console.log(`✅ Using local PostgreSQL from: ${localPgPath}`);
        pgFound = await this.copyLocalPostgreSQL(localPgPath, pgDir);
        if (pgFound) break;
      }
    }
    
    // 如果没有找到本地安装，自动下载
    if (!pgFound) {
      console.log('🔍 Local PostgreSQL not found, downloading portable version...');
      pgFound = await this.downloadPostgreSQL(pgDir);
    }
    
    if (!pgFound) {
      console.log('⚠️  Failed to setup PostgreSQL. Standalone package will require Docker or system PostgreSQL.');
      fs.mkdirSync(path.join(pgDir, 'bin'), { recursive: true });
    }
    
    fs.mkdirSync(path.join(pgDir, 'data'), { recursive: true });
    this.createPortablePostgreSQLScripts(pgDir);
  }

  createPortablePostgreSQLScripts(pgDir) {
    // 创建便携式PostgreSQL启动脚本（不依赖Docker）
    const startScript = `#!/bin/bash
# Portable PostgreSQL Manager
SCRIPT_DIR="$( cd "$( dirname "\${BASH_SOURCE[0]}" )" && pwd )"
PGDIR="$SCRIPT_DIR"
PGDATA="$PGDIR/data"
PGPORT=5432
PGLOG="$PGDIR/postgresql.log"

# 先检查PostgreSQL是否已经在运行
if pg_isready -h localhost -p 5432 2>/dev/null; then
  echo "✅ PostgreSQL is already running on port 5432"
  
  # 检查数据库和用户是否存在
  if psql -h localhost -p 5432 -U kimbap -d kimbap_db -c "SELECT 1" 2>/dev/null; then
    echo "✅ Database kimbap_db and user kimbap exist"
  else
    echo "Creating database and user..."
    # 使用当前用户创建数据库和用户
    createdb -h localhost -p 5432 kimbap_db 2>/dev/null || echo "Database kimbap_db might already exist"
    psql -h localhost -p 5432 -d postgres -c "CREATE USER kimbap WITH PASSWORD 'kimbap123';" 2>/dev/null || echo "User kimbap might already exist"
    psql -h localhost -p 5432 -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE kimbap_db TO kimbap;" 2>/dev/null
    psql -h localhost -p 5432 -d postgres -c "ALTER DATABASE kimbap_db OWNER TO kimbap;" 2>/dev/null
    
    # 创建表
    if [ -f "$PGDIR/init-tables.sql" ]; then
      echo "Creating database tables..."
      psql -h localhost -p 5432 -U kimbap -d kimbap_db -f "$PGDIR/init-tables.sql" 2>/dev/null || echo "Tables might already exist"
    fi
  fi
  exit 0
fi

# 检查是否有内置的PostgreSQL二进制文件
if [ -f "$PGDIR/bin/postgres" ]; then
  echo "🐘 Starting embedded PostgreSQL..."
  
  # 初始化数据库（如果需要）
  if [ ! -f "$PGDATA/PG_VERSION" ]; then
    echo "Initializing database..."
    "$PGDIR/bin/initdb" -D "$PGDATA" --auth-local=trust --auth-host=trust
    
    # 创建配置文件
    cat >> "$PGDATA/postgresql.conf" <<EOF
listen_addresses = 'localhost'
port = 5432
max_connections = 100
shared_buffers = 128MB
EOF
    
    # 启动PostgreSQL
    "$PGDIR/bin/pg_ctl" -D "$PGDATA" -l "$PGLOG" start
    
    # 等待启动
    sleep 3
    
    # 创建数据库和用户
    "$PGDIR/bin/createdb" -h localhost -p 5432 kimbap_db 2>/dev/null || echo "Database might already exist"
    "$PGDIR/bin/psql" -h localhost -p 5432 -d postgres -c "CREATE USER kimbap WITH PASSWORD 'kimbap123';" 2>/dev/null || echo "User might already exist"
    "$PGDIR/bin/psql" -h localhost -p 5432 -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE kimbap_db TO kimbap;" 2>/dev/null
    "$PGDIR/bin/psql" -h localhost -p 5432 -d postgres -c "ALTER DATABASE kimbap_db OWNER TO kimbap;" 2>/dev/null
    
    # 创建表
    if [ -f "$PGDIR/init-tables.sql" ]; then
      echo "Creating database tables..."
      "$PGDIR/bin/psql" -h localhost -p 5432 -U kimbap -d kimbap_db -f "$PGDIR/init-tables.sql" 2>/dev/null || echo "Tables might already exist"
    fi
  else
    # 启动PostgreSQL
    "$PGDIR/bin/pg_ctl" -D "$PGDATA" -l "$PGLOG" start
  fi
  
  echo "✅ PostgreSQL started on port 5432"
  exit 0
else
  # 回退到系统PostgreSQL或Docker
  echo "⚠️  Embedded PostgreSQL not found, trying system PostgreSQL..."
  
  # 检查系统PostgreSQL
  if command -v pg_ctl &> /dev/null; then
    # 使用系统PostgreSQL
    if [ ! -f "$PGDATA/PG_VERSION" ]; then
      initdb -D "$PGDATA" -U kimbap --auth-local=trust --auth-host=md5
      pg_ctl -D "$PGDATA" -l "$PGLOG" start
      sleep 3
      createdb -U kimbap kimbap_db
      psql -U kimbap -c "ALTER USER kimbap PASSWORD 'kimbap123';"
    else
      pg_ctl -D "$PGDATA" -l "$PGLOG" start
    fi
    echo "✅ System PostgreSQL started"
    exit 0
  fi
  
  # 最后尝试Docker
  if command -v docker &> /dev/null; then
    echo "Using Docker as fallback..."
    docker stop kimbap-postgres-embedded 2>/dev/null
    docker rm kimbap-postgres-embedded 2>/dev/null
    
    docker run -d \\
      --name kimbap-postgres-embedded \\
      -e POSTGRES_USER=kimbap \\
      -e POSTGRES_PASSWORD=kimbap123 \\
      -e POSTGRES_DB=kimbap_db \\
      -p 5432:5432 \\
      -v "$PGDATA:/var/lib/postgresql/data" \\
      postgres:16-alpine
    
    # 等待 PostgreSQL 容器就绪
    echo "⏳ Waiting for PostgreSQL container to be ready..."
    for i in {1..15}; do
      if pg_isready -h localhost -p 5432 -U kimbap 2>/dev/null; then
        echo "✅ PostgreSQL container is ready"
        break
      fi
      echo "⏳ Waiting... ($i/15)"
      sleep 2
    done
    
    # 创建数据库表（如果需要）
    if [ -f "$PGDIR/init-tables.sql" ]; then
      echo "📊 Initializing database tables..."
      docker exec kimbap-postgres-embedded psql -U kimbap -d kimbap_db -f /var/lib/postgresql/data/../init-tables.sql 2>/dev/null || echo "Tables initialization completed"
    fi
    
    echo "✅ PostgreSQL started with Docker"
    exit 0
  fi
  
  echo "❌ No PostgreSQL available. Please install PostgreSQL manually."
  exit 1
fi
`;

    const stopScript = `#!/bin/bash
# Stop embedded PostgreSQL
SCRIPT_DIR="$( cd "$( dirname "\${BASH_SOURCE[0]}" )" && pwd )"
PGDIR="$SCRIPT_DIR"
PGDATA="$PGDIR/data"

if [ -f "$PGDIR/bin/pg_ctl" ]; then
  echo "Stopping embedded PostgreSQL..."
  "$PGDIR/bin/pg_ctl" -D "$PGDATA" stop
elif command -v pg_ctl &> /dev/null; then
  echo "Stopping system PostgreSQL..."
  pg_ctl -D "$PGDATA" stop
elif command -v docker &> /dev/null; then
  echo "Stopping Docker PostgreSQL..."
  docker stop kimbap-postgres-embedded 2>/dev/null
  docker rm kimbap-postgres-embedded 2>/dev/null
fi
echo "PostgreSQL stopped."
`;

    // Windows批处理脚本
    const startBat = `@echo off
REM Portable PostgreSQL Manager for Windows
set PGDIR=%~dp0
set PGDATA=%PGDIR%data
set PGPORT=5432
set PGLOG=%PGDIR%postgresql.log

if exist "%PGDIR%bin\\postgres.exe" (
  echo Starting embedded PostgreSQL...
  
  if not exist "%PGDATA%\\PG_VERSION" (
    echo Initializing database...
    "%PGDIR%bin\\initdb.exe" -D "%PGDATA%" -U kimbap --auth-local=trust --auth-host=md5
    "%PGDIR%bin\\pg_ctl.exe" -D "%PGDATA%" -l "%PGLOG%" start
    timeout /t 3 /nobreak >nul
    "%PGDIR%bin\\createdb.exe" -U kimbap kimbap_db
    "%PGDIR%bin\\psql.exe" -U kimbap -c "ALTER USER kimbap PASSWORD 'kimbap123';"
  ) else (
    "%PGDIR%bin\\pg_ctl.exe" -D "%PGDATA%" -l "%PGLOG%" start
  )
  
  echo PostgreSQL started on port 5432
  exit /b 0
)

echo Embedded PostgreSQL not found, please install PostgreSQL manually.
exit /b 1
`;

    // SQL脚本创建表
    const initTablesSQL = `-- Create tables for Kimbap Console

CREATE TABLE IF NOT EXISTS "user" (
  user_id VARCHAR(64) PRIMARY KEY,
  status INT NOT NULL,
  role INT NOT NULL,
  permissions TEXT NOT NULL,
  server_api_keys TEXT NOT NULL,
  expires_at INT DEFAULT 0,
  created_at INT DEFAULT 0,
  updated_at INT DEFAULT 0,
  ratelimit INT NOT NULL,
  name VARCHAR(128) NOT NULL,
  encrypted_token TEXT,
  proxy_id INT DEFAULT 0,
  notes TEXT
);

CREATE TABLE IF NOT EXISTS server (
  server_id VARCHAR(128) PRIMARY KEY,
  server_name VARCHAR(128) NOT NULL,
  enabled BOOLEAN DEFAULT true,
  launch_config TEXT NOT NULL,
  capabilities TEXT NOT NULL,
  created_at INT DEFAULT 0,
  updated_at INT DEFAULT 0,
  allow_user_input BOOLEAN DEFAULT false,
  proxy_id INT DEFAULT 0,
  tool_tmpl_id VARCHAR(128)
);

CREATE TABLE IF NOT EXISTS server_user (
  server_user_id VARCHAR(128) PRIMARY KEY,
  server_id VARCHAR(128) NOT NULL,
  user_id VARCHAR(64) NOT NULL,
  role INT NOT NULL,
  created_at INT DEFAULT 0,
  updated_at INT DEFAULT 0
);

CREATE TABLE IF NOT EXISTS access_token (
  token_id VARCHAR(128) PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  token_hash VARCHAR(128) NOT NULL,
  name VARCHAR(128) NOT NULL,
  expires_at INT DEFAULT 0,
  created_at INT DEFAULT 0,
  last_used_at INT DEFAULT 0,
  permissions TEXT
);

CREATE TABLE IF NOT EXISTS tool (
  tool_id VARCHAR(128) PRIMARY KEY,
  server_id VARCHAR(128) NOT NULL,
  tool_name VARCHAR(128) NOT NULL,
  config TEXT,
  status INT NOT NULL,
  created_at INT DEFAULT 0,
  updated_at INT DEFAULT 0
);

CREATE TABLE IF NOT EXISTS activity (
  activity_id VARCHAR(128) PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  server_id VARCHAR(128),
  action VARCHAR(128) NOT NULL,
  details TEXT,
  ip_address VARCHAR(45),
  user_agent TEXT,
  created_at INT DEFAULT 0
);

CREATE TABLE IF NOT EXISTS server_metric (
  metric_id VARCHAR(128) PRIMARY KEY,
  server_id VARCHAR(128) NOT NULL,
  metric_type VARCHAR(64) NOT NULL,
  value REAL NOT NULL,
  metadata TEXT,
  created_at INT DEFAULT 0
);

CREATE TABLE IF NOT EXISTS ip_whitelist (
  id SERIAL PRIMARY KEY,
  ip VARCHAR(128) NOT NULL DEFAULT '',
  addtime INT DEFAULT 0
);

CREATE TABLE IF NOT EXISTS proxy (
  id SERIAL PRIMARY KEY,
  name VARCHAR(128) NOT NULL,
  addtime INT DEFAULT 0,
  proxy_key VARCHAR(255) DEFAULT '',
  start_port INT DEFAULT 3002
);

CREATE TABLE IF NOT EXISTS log (
  id SERIAL PRIMARY KEY,
  timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  user_id VARCHAR(64) NOT NULL,
  type INT NOT NULL,
  request_content TEXT,
  response_content TEXT,
  error_content TEXT,
  "serverID" VARCHAR(128)
);

CREATE TABLE IF NOT EXISTS mcp_events (
  id SERIAL PRIMARY KEY,
  event_id VARCHAR(255) UNIQUE NOT NULL,
  stream_id VARCHAR(255) NOT NULL,
  session_id VARCHAR(255) NOT NULL,
  message_type VARCHAR(50) NOT NULL,
  message_data TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  expires_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_mcp_events_stream_id ON mcp_events(stream_id);
CREATE INDEX IF NOT EXISTS idx_mcp_events_session_id ON mcp_events(session_id);
CREATE INDEX IF NOT EXISTS idx_mcp_events_created_at ON mcp_events(created_at);
CREATE INDEX IF NOT EXISTS idx_mcp_events_expires_at ON mcp_events(expires_at);

CREATE TABLE IF NOT EXISTS dns_conf (
  subdomain VARCHAR(128) DEFAULT '',
  type INT DEFAULT 0,
  public_ip VARCHAR(128) DEFAULT '',
  id SERIAL PRIMARY KEY,
  addtime INT DEFAULT 0,
  update_time INT DEFAULT 0,
  tunnel_id VARCHAR(256) DEFAULT ''
);`;

    // 保存脚本
    fs.writeFileSync(path.join(pgDir, 'init-tables.sql'), initTablesSQL);
    fs.writeFileSync(path.join(pgDir, 'start.sh'), startScript);
    fs.chmodSync(path.join(pgDir, 'start.sh'), 0o755);
    
    fs.writeFileSync(path.join(pgDir, 'stop.sh'), stopScript);
    fs.chmodSync(path.join(pgDir, 'stop.sh'), 0o755);
    
    fs.writeFileSync(path.join(pgDir, 'start.bat'), startBat);
  }

  createPostgreSQLScripts(pgDir) {
    // 创建使用Docker的PostgreSQL启动脚本
    const startScript = `#!/bin/bash
# Embedded PostgreSQL Manager
PGDATA="${pgDir}/data"
PGPORT=5432

# 检查Docker是否可用
if command -v docker &> /dev/null; then
  # 停止可能存在的容器
  docker stop kimbap-postgres-embedded 2>/dev/null
  docker rm kimbap-postgres-embedded 2>/dev/null
  
  echo "🐳 Starting PostgreSQL with Docker..."
  docker run -d \\
    --name kimbap-postgres-embedded \\
    -e POSTGRES_USER=kimbap \\
    -e POSTGRES_PASSWORD=kimbap123 \\
    -e POSTGRES_DB=kimbap_db \\
    -p 5432:5432 \\
    -v "$PGDATA:/var/lib/postgresql/data" \\
    postgres:16-alpine
  
  # 等待PostgreSQL启动
  echo "Waiting for PostgreSQL to start..."
  sleep 5
  
  # 检查连接
  for i in {1..10}; do
    if docker exec kimbap-postgres-embedded pg_isready -U kimbap -d kimbap_db 2>/dev/null; then
      echo "✅ PostgreSQL is ready!"
      exit 0
    fi
    sleep 2
  done
  echo "⚠️  PostgreSQL startup timeout"
  exit 1
else
  echo "❌ Docker not found. Please install Docker or PostgreSQL manually."
  echo "   Manual PostgreSQL setup:"
  echo "   1. Install PostgreSQL 16"
  echo "   2. Create database: CREATE DATABASE kimbap_db;"
  echo "   3. Create user: CREATE USER kimbap WITH PASSWORD 'kimbap123';"
  echo "   4. Grant privileges: GRANT ALL ON DATABASE kimbap_db TO kimbap;"
  exit 1
fi
`;

    const stopScript = `#!/bin/bash
# Stop embedded PostgreSQL
if command -v docker &> /dev/null; then
  echo "Stopping PostgreSQL..."
  docker stop kimbap-postgres-embedded 2>/dev/null
  docker rm kimbap-postgres-embedded 2>/dev/null
  echo "PostgreSQL stopped."
fi
`;

    // Windows批处理脚本
    const startBat = `@echo off
REM Embedded PostgreSQL Manager for Windows

docker --version >nul 2>&1
if %errorlevel% neq 0 (
  echo Docker not found. Please install Docker Desktop for Windows.
  echo Or install PostgreSQL manually.
  pause
  exit /b 1
)

echo Stopping existing container...
docker stop kimbap-postgres-embedded 2>nul
docker rm kimbap-postgres-embedded 2>nul

echo Starting PostgreSQL with Docker...
docker run -d ^
  --name kimbap-postgres-embedded ^
  -e POSTGRES_USER=kimbap ^
  -e POSTGRES_PASSWORD=kimbap123 ^
  -e POSTGRES_DB=kimbap_db ^
  -p 5432:5432 ^
  postgres:16-alpine

echo Waiting for PostgreSQL to start...
timeout /t 5 /nobreak >nul

echo PostgreSQL started on port 5432
`;

    // 保存脚本
    fs.writeFileSync(path.join(pgDir, 'start.sh'), startScript);
    fs.chmodSync(path.join(pgDir, 'start.sh'), 0o755);
    
    fs.writeFileSync(path.join(pgDir, 'stop.sh'), stopScript);
    fs.chmodSync(path.join(pgDir, 'stop.sh'), 0o755);
    
    fs.writeFileSync(path.join(pgDir, 'start.bat'), startBat);
  }

  async downloadNode(destination) {
    const version = '20.11.0';
    const baseUrl = 'https://nodejs.org/dist';
    
    let fileName;
    if (this.platform === 'win32') {
      fileName = `node-v${version}-win-${this.arch}.zip`;
    } else if (this.platform === 'darwin') {
      fileName = `node-v${version}-darwin-${this.arch}.tar.gz`;
    } else {
      fileName = `node-v${version}-linux-${this.arch}.tar.xz`;
    }
    
    const url = `${baseUrl}/v${version}/${fileName}`;
    
    // 使用curl下载
    execSync(`curl -L ${url} -o ${destination}`, { stdio: 'inherit' });
  }

  async extractNode(archivePath, nodeDir) {
    console.log('📦 Extracting Node.js...');
    
    const tempExtractDir = path.join(path.dirname(archivePath), 'node-extract');
    fs.mkdirSync(tempExtractDir, { recursive: true });
    
    // 根据目标平台选择解压命令，而不是当前平台
    if (this.platform === 'win32') {
      // Windows平台的文件是zip格式，在非Windows系统上用unzip解压
      if (process.platform === 'win32') {
        execSync(`powershell -command "Expand-Archive -Path '${archivePath}' -DestinationPath '${tempExtractDir}' -Force"`, { stdio: 'pipe' });
      } else {
        // 在Mac/Linux上为Windows构建时使用unzip
        execSync(`unzip -q "${archivePath}" -d "${tempExtractDir}"`, { stdio: 'pipe' });
      }
    } else {
      // Linux/Mac平台使用tar
      execSync(`tar -xf "${archivePath}" -C "${tempExtractDir}"`, { stdio: 'pipe' });
    }
    
    const extractedDirs = fs.readdirSync(tempExtractDir);
    const nodeSourceDir = path.join(tempExtractDir, extractedDirs[0]);
    
    this.copyDir(nodeSourceDir, nodeDir);
    fs.rmSync(tempExtractDir, { recursive: true, force: true });
  }

  async copyApplicationFiles() {
    console.log('📋 Copying application files...');
    
    const appDir = path.join(this.outputDir, 'app');
    const standaloneDir = path.join(this.rootDir, '.next/standalone');
    
    if (!fs.existsSync(standaloneDir)) {
      throw new Error('Standalone build not found. Build may have failed.');
    }
    
    // 1. 复制整个standalone构建
    console.log('✅ Copying complete standalone build');
    this.copyDir(path.join(standaloneDir, '.next'), path.join(appDir, '.next'));
    this.copyDir(path.join(standaloneDir, 'node_modules'), path.join(appDir, 'node_modules'));
    fs.copyFileSync(path.join(standaloneDir, 'server.js'), path.join(appDir, 'server.js'));
    fs.copyFileSync(path.join(standaloneDir, 'package.json'), path.join(appDir, 'package.json'));
    
    // 2. 复制静态文件（关键！）
    console.log('📁 Copying static assets...');
    this.copyDir(path.join(this.rootDir, '.next/static'), path.join(appDir, '.next/static'));
    
    // 3. 复制public和prisma
    this.copyDir(path.join(this.rootDir, 'public'), path.join(appDir, 'public'));
    this.copyDir(path.join(this.rootDir, 'prisma'), path.join(appDir, 'prisma'));
    
    // 4. 复制后端proxy-server
    const proxyServerDir = path.join(this.rootDir, 'proxy-server');
    if (fs.existsSync(proxyServerDir)) {
      this.copyDir(proxyServerDir, path.join(appDir, 'proxy-server'));
      // 创建ES模块配置
      fs.writeFileSync(
        path.join(appDir, 'proxy-server/package.json'),
        JSON.stringify({ type: 'module' }, null, 2)
      );
    }
    
    // 5. 复制数据库配置脚本（修复版）
    this.createDatabaseConfigScript(appDir);
    
    // 6. 创建package.json
    const pkg = {
      name: 'kimbap-console',
      version: '1.0.0',
      private: true,
      scripts: {
        start: 'node .next/standalone/server.js',
        'start:backend': 'node proxy-server/index.js'
      }
    };
    fs.writeFileSync(path.join(appDir, 'package.json'), JSON.stringify(pkg, null, 2));
    
    // 7. 复制next.config.mjs
    fs.copyFileSync(
      path.join(this.rootDir, 'next.config.standalone.mjs'),
      path.join(appDir, 'next.config.mjs')
    );
    
    // 8. 创建环境配置
    const envConfig = `NODE_ENV=production
DATABASE_URL=postgresql://kimbap:kimbap123@localhost:5432/kimbap_db`;
    
    fs.writeFileSync(path.join(appDir, '.env.local'), envConfig);
  }

  createDatabaseConfigScript(appDir) {
    fs.mkdirSync(path.join(appDir, 'scripts'), { recursive: true });
    
    // 直接复制我们修复过的数据库配置脚本
    const sourceScript = path.join(this.rootDir, 'scripts', 'database-config.js');
    const targetScript = path.join(appDir, 'scripts', 'database-config.js');
    
    if (fs.existsSync(sourceScript)) {
      fs.copyFileSync(sourceScript, targetScript);
      fs.chmodSync(targetScript, 0o755);
      return;
    }
    
    // 如果源文件不存在，创建基础版本
    const script = `#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

class DatabaseConfig {
  constructor() {
    this.rootDir = process.cwd();
    this.envPath = path.join(this.rootDir, '.env.local');
    
    this.configs = {
      local: {
        host: 'localhost',
        port: 5432,
        database: 'kimbap_db',
        username: 'kimbap',
        password: 'kimbap123'
      }
    };
  }

  async detectBestConfig() {
    console.log('🔍 Detecting database configuration...');
    
    if (await this.testLocalConnection()) {
      console.log('✅ Using local PostgreSQL');
      return this.generateDatabaseUrl('local');
    }
    
    throw new Error('No PostgreSQL database available');
  }

  async testLocalConnection() {
    try {
      const { Client } = require('pg');
      const client = new Client({
        host: this.configs.local.host,
        port: this.configs.local.port,
        database: this.configs.local.database,
        user: this.configs.local.username,
        password: this.configs.local.password,
        connectionTimeoutMillis: 5000
      });
      
      await client.connect();
      await client.query('SELECT 1');
      await client.end();
      return true;
    } catch (error) {
      return false;
    }
  }

  generateDatabaseUrl(type) {
    const config = this.configs[type];
    return \`postgresql://\${config.username}:\${config.password}@\${config.host}:\${config.port}/\${config.database}\`;
  }

  async updateEnvFile(databaseUrl) {
    let envContent = '';
    if (fs.existsSync(this.envPath)) {
      envContent = fs.readFileSync(this.envPath, 'utf8');
    }
    
    if (envContent.includes('DATABASE_URL=')) {
      envContent = envContent.replace(/DATABASE_URL=.*/g, \`DATABASE_URL=\${databaseUrl}\`);
    } else {
      envContent += \`\\nDATABASE_URL=\${databaseUrl}\`;
    }
    
    fs.writeFileSync(this.envPath, envContent);
    process.env.DATABASE_URL = databaseUrl;
  }

  async runMigrations(databaseUrl) {
    try {
      process.env.DATABASE_URL = databaseUrl;
      
      // Skip Prisma operations in standalone build - client already generated
      console.log('✅ Using pre-generated Prisma client');
      
      return true;
    } catch (error) {
      console.error('❌ Migration failed:', error.message);
      return false;
    }
  }

  async run(command) {
    try {
      const databaseUrl = await this.detectBestConfig();
      
      if (command === 'validate') {
        await this.updateEnvFile(databaseUrl);
        await this.runMigrations(databaseUrl);
        console.log('✅ Database configuration completed');
      }
      
      return 0;
    } catch (error) {
      console.error('❌ Database configuration failed:', error.message);
      return 1;
    }
  }
}

if (require.main === module) {
  const config = new DatabaseConfig();
  const command = process.argv[2] || 'test';
  config.run(command).then(process.exit);
}

module.exports = DatabaseConfig;`;

    fs.writeFileSync(path.join(appDir, 'scripts/database-config.js'), script);
  }

  async fixPathsAndConfigs() {
    console.log('🔧 Fixing paths and configurations...');
    
    const appDir = path.join(this.outputDir, 'app');
    
    // 修复proxy-server的prisma导入路径
    const prismaConfigPath = path.join(appDir, 'proxy-server/config/prisma.js');
    if (fs.existsSync(prismaConfigPath)) {
      let content = fs.readFileSync(prismaConfigPath, 'utf8');
      // 修复为正确的相对路径 - 使用default.js
      content = content.replace(
        /from ['"].*@prisma\/client.*['"]/g,
        "from '../../node_modules/@prisma/client/default.js'"
      );
      fs.writeFileSync(prismaConfigPath, content);
    }
  }

  async createStartupScripts() {
    console.log('📝 Creating startup scripts...');
    
    const scriptsDir = path.join(this.outputDir, 'scripts');
    fs.mkdirSync(scriptsDir, { recursive: true });
    
    // Unix/Mac启动脚本
    const unixScript = `#!/bin/bash

echo "========================================" 
echo "       Kimbap Console Starting"
echo "========================================"

SCRIPT_DIR="$( cd "$( dirname "\${BASH_SOURCE[0]}" )" && pwd )"
ROOT_DIR="$( dirname "$SCRIPT_DIR" )"

cd "$ROOT_DIR/app"

# 设置环境变量
export NODE_ENV=production
export PATH="$ROOT_DIR/node/bin:$PATH"

# 启动PostgreSQL（如果需要）
echo "🐘 Checking PostgreSQL..."
if ! pg_isready -h localhost -p 5432 -U kimbap 2>/dev/null; then
  echo "Starting embedded PostgreSQL..."
  bash "$ROOT_DIR/postgresql/start.sh"
  if [ $? -ne 0 ]; then
    echo "❌ Failed to start PostgreSQL"
    echo "Please ensure Docker is running or PostgreSQL is installed"
    exit 1
  fi
  sleep 3
fi

# 检测数据库连接
echo "🔍 Validating database connection..."
if ../node/bin/node scripts/database-config.js validate; then
  echo "✅ Database ready"
else
  echo "❌ Database setup failed"
  exit 1
fi

# 读取DATABASE_URL
export DATABASE_URL=$(grep "^DATABASE_URL=" .env.local | cut -d'=' -f2- | tr -d '"')

# 初始化数据库表（如果需要）
echo "📊 Initializing database tables..."
../node/bin/node -e "
const { PrismaClient } = require('@prisma/client');
const prisma = new PrismaClient();
prisma.\$connect()
  .then(() => {
    console.log('Database tables initialized');
    return prisma.\$disconnect();
  })
  .catch((err) => {
    console.log('Creating database tables...');
    const { execSync } = require('child_process');
    try {
      execSync('npx prisma db push --accept-data-loss', { stdio: 'inherit' });
      console.log('Database tables created successfully');
    } catch (e) {
      console.error('Failed to create tables:', e.message);
    }
  });
"

# 启动后端
echo "🔧 Starting backend..."
../node/bin/node proxy-server/index.js &
BACKEND_PID=$!

sleep 2

# 启动前端
echo "🎨 Starting frontend..."
if [ -f "server.js" ]; then
  ../node/bin/node server.js &
else
  ../node/bin/node node_modules/next/dist/bin/next start &
fi
FRONTEND_PID=$!

echo "📱 Open http://localhost:3000"
echo "🛑 Press Ctrl+C to stop"

# 清理函数
cleanup() {
  echo "Stopping services..."
  kill $BACKEND_PID $FRONTEND_PID 2>/dev/null
  
  # 停止PostgreSQL
  if [ -f "$ROOT_DIR/postgresql/stop.sh" ]; then
    echo "Stopping PostgreSQL..."
    bash "$ROOT_DIR/postgresql/stop.sh"
  fi
  
  exit 0
}

trap cleanup EXIT INT TERM
wait`;

    fs.writeFileSync(path.join(scriptsDir, 'start.sh'), unixScript);
    fs.chmodSync(path.join(scriptsDir, 'start.sh'), 0o755);
    
    // Windows启动脚本
    const winScript = `@echo off
title Kimbap Console

echo ========================================
echo        Kimbap Console Starting  
echo ========================================

set "SCRIPT_DIR=%~dp0"
set "ROOT_DIR=%SCRIPT_DIR%..\\"
set "NODE_ENV=production"

cd /d "%ROOT_DIR%\\app"

REM 检测数据库
..\\node\\node.exe scripts\\database-config.js validate
if %errorlevel% neq 0 (
  echo Database setup failed!
  pause
  exit /b 1
)

REM 启动服务
echo Starting services...
start "Backend" /min ..\\node\\node.exe proxy-server\\index.js
timeout /t 2 > nul
if exist "server.js" (
  start "Frontend" ..\\node\\node.exe server.js
) else (
  start "Frontend" ..\\node\\node.exe node_modules\\next\\dist\\bin\\next start
)

echo Open http://localhost:3000
pause`;

    fs.writeFileSync(path.join(scriptsDir, 'start.bat'), winScript);
  }

  async createDocumentation() {
    console.log('📄 Creating documentation...');
    
    const readme = `# Kimbap Console Deployment Package

## Quick Start

### Prerequisites
- PostgreSQL database (local or cloud)
- Port 3000 and 3002 available

### Start Application

**Windows:**
\`\`\`
scripts\\start.bat
\`\`\`

**Mac/Linux:**
\`\`\`bash
./scripts/start.sh
\`\`\`

### Access
Open browser: http://localhost:3000

## Database Setup

The application will auto-detect PostgreSQL. If not found, install using:

**Docker:**
\`\`\`bash
docker run --name kimbap-postgres \\
  -e POSTGRES_USER=kimbap \\
  -e POSTGRES_PASSWORD=kimbap123 \\
  -e POSTGRES_DB=kimbap_db \\
  -p 5432:5432 -d postgres:16
\`\`\`

**Local Installation:**
- Mac: \`brew install postgresql@16\`
- Linux: \`apt install postgresql-16\`
- Windows: Download from postgresql.org

---
Built on: ${new Date().toISOString()}
Platform: ${this.platform}-${this.arch}`;

    fs.writeFileSync(path.join(this.outputDir, 'README.md'), readme);
  }

  // 辅助方法
  copyDir(src, dest) {
    if (!fs.existsSync(src)) return;
    
    fs.mkdirSync(dest, { recursive: true });
    
    const entries = fs.readdirSync(src, { withFileTypes: true });
    
    for (const entry of entries) {
      const srcPath = path.join(src, entry.name);
      const destPath = path.join(dest, entry.name);
      
      if (entry.isSymbolicLink()) {
        // 处理符号链接：读取目标并复制实际内容
        try {
          const linkTarget = fs.readlinkSync(srcPath);
          let realPath;
          
          if (path.isAbsolute(linkTarget)) {
            realPath = linkTarget;
          } else {
            realPath = path.resolve(path.dirname(srcPath), linkTarget);
          }
          
          if (fs.existsSync(realPath)) {
            const stat = fs.lstatSync(realPath);
            if (stat.isDirectory()) {
              this.copyDir(realPath, destPath);
            } else {
              fs.copyFileSync(realPath, destPath);
            }
          }
        } catch (error) {
          console.warn(`⚠️  Failed to resolve symlink: ${srcPath} -> ${error.message}`);
        }
      } else if (entry.isDirectory()) {
        this.copyDir(srcPath, destPath);
      } else {
        fs.copyFileSync(srcPath, destPath);
      }
    }
  }

  // 创建平台特定的可执行文件
  async createExecutables() {
    console.log('🚀 Creating platform-specific executables...');
    
    if (this.platform === 'win32') {
      await this.createWindowsExecutable();
    } else if (this.platform === 'darwin') {
      await this.createMacOSApp();
    } else if (this.platform === 'linux') {
      await this.createLinuxDesktopFile();
    }
  }

  // 创建 Windows 可执行文件
  async createWindowsExecutable() {
    console.log('📁 Creating Windows executable...');
    
    // 创建主启动批处理文件
    const mainBatContent = `@echo off
title Kimbap Console
cd /d "%~dp0"
call scripts\\start.bat
pause`;
    
    const mainBatPath = path.join(this.outputDir, 'Kimbap-Console.bat');
    fs.writeFileSync(mainBatPath, mainBatContent);
    
    // 尝试使用 Node.js 创建一个简单的 .exe 启动器
    const exeLauncherContent = `#!/usr/bin/env node
const { spawn } = require('child_process');
const path = require('path');

const scriptDir = __dirname;
const startScript = path.join(scriptDir, 'scripts', 'start.bat');

// 启动应用
spawn('cmd', ['/c', startScript], {
  cwd: scriptDir,
  stdio: 'inherit'
});`;

    const exeLauncherPath = path.join(this.outputDir, 'launcher.js');
    fs.writeFileSync(exeLauncherPath, exeLauncherContent);

    // 创建双击启动的批处理文件
    const quickStartContent = `@echo off
title Kimbap Console
cd /d "%~dp0"
echo Starting Kimbap Console...
call scripts\\start.bat`;
    
    const quickStartPath = path.join(this.outputDir, 'Start-Kimbap-Console.bat');
    fs.writeFileSync(quickStartPath, quickStartContent);
    
    // 创建PowerShell启动脚本
    const psContent = `# Kimbap Console PowerShell Launcher
Set-Location $PSScriptRoot
Write-Host "Starting Kimbap Console..." -ForegroundColor Green
& .\\scripts\\start.bat`;
    
    const psPath = path.join(this.outputDir, 'Start-Kimbap-Console.ps1');
    fs.writeFileSync(psPath, psContent);
    
    console.log('✅ Windows executables created:');
    console.log('   - Kimbap-Console.bat (with pause)');
    console.log('   - Start-Kimbap-Console.bat (auto-start)');
    console.log('   - Start-Kimbap-Console.ps1 (PowerShell)');
    console.log('   - launcher.js (Node.js)');
  }

  // 创建 macOS .app 应用包
  async createMacOSApp() {
    console.log('📱 Creating macOS .app bundle...');
    
    const appName = 'Kimbap Console.app';
    const appPath = path.join(this.outputDir, appName);
    const contentsPath = path.join(appPath, 'Contents');
    const macOSPath = path.join(contentsPath, 'MacOS');
    const resourcesPath = path.join(contentsPath, 'Resources');
    
    // 创建 .app 目录结构
    fs.mkdirSync(macOSPath, { recursive: true });
    fs.mkdirSync(resourcesPath, { recursive: true });
    
    // 创建 Info.plist
    const infoPlistContent = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>Kimbap Console</string>
    <key>CFBundleIdentifier</key>
    <string>com.kimbap.console</string>
    <key>CFBundleName</key>
    <string>Kimbap Console</string>
    <key>CFBundleVersion</key>
    <string>1.0.0</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0.0</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleSignature</key>
    <string>Kimbap</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.14</string>
    <key>NSHighResolutionCapable</key>
    <true/>
</dict>
</plist>`;
    
    fs.writeFileSync(path.join(contentsPath, 'Info.plist'), infoPlistContent);
    
    // 创建主可执行文件
    const mainExecutableContent = `#!/bin/bash
# Kimbap Console macOS App Launcher
SCRIPT_DIR="$( cd "$( dirname "\${BASH_SOURCE[0]}" )" && pwd )"
APP_DIR="$( dirname "$( dirname "$SCRIPT_DIR" )" )"

# 切换到应用目录
cd "$APP_DIR"

# 启动控制台应用
./scripts/start.sh

# 打开浏览器
sleep 3
open http://localhost:3000`;
    
    const executablePath = path.join(macOSPath, 'Kimbap Console');
    fs.writeFileSync(executablePath, mainExecutableContent);
    fs.chmodSync(executablePath, 0o755);
    
    // 创建启动脚本
    const startScriptContent = `#!/bin/bash
# Quick launcher for Kimbap Console
cd "$( dirname "\${BASH_SOURCE[0]}" )"
./scripts/start.sh`;
    
    const quickStartPath = path.join(this.outputDir, 'Start-Kimbap-Console.command');
    fs.writeFileSync(quickStartPath, startScriptContent);
    fs.chmodSync(quickStartPath, 0o755);
    
    console.log('✅ macOS .app bundle created: Kimbap Console.app');
    console.log('✅ Quick launcher created: Start-Kimbap-Console.command');
  }

  // 创建 Linux 桌面文件
  async createLinuxDesktopFile() {
    console.log('🐧 Creating Linux desktop file...');
    
    const desktopContent = `[Desktop Entry]
Version=1.0
Type=Application
Name=Kimbap Console
Comment=Kimbap Console Application
Exec=bash "%k/scripts/start.sh"
Icon=%k/icon.png
Path=%k
Terminal=false
StartupNotify=true
Categories=Development;Network;
Keywords=Kimbap;Console;AI;`;
    
    const desktopPath = path.join(this.outputDir, 'Kimbap-Console.desktop');
    fs.writeFileSync(desktopPath, desktopContent);
    fs.chmodSync(desktopPath, 0o755);
    
    // 创建简单的启动脚本
    const startScriptContent = `#!/bin/bash
# Kimbap Console Linux Launcher
SCRIPT_DIR="$( cd "$( dirname "\${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

echo "Starting Kimbap Console..."
./scripts/start.sh

# 打开浏览器 (如果有桌面环境)
if command -v xdg-open >/dev/null 2>&1; then
    sleep 3
    xdg-open http://localhost:3000
fi`;
    
    const launcherPath = path.join(this.outputDir, 'start-kimbap-console');
    fs.writeFileSync(launcherPath, startScriptContent);
    fs.chmodSync(launcherPath, 0o755);
    
    // 创建简单的图标 (如果不存在)
    const iconPath = path.join(this.outputDir, 'icon.png');
    if (!fs.existsSync(iconPath)) {
      // 创建一个简单的文本图标文件
      const iconTextContent = `Kimbap Console Icon - Replace with actual PNG icon`;
      fs.writeFileSync(iconPath, iconTextContent);
    }
    
    // 创建服务脚本 (systemd service)
    const serviceContent = `[Unit]
Description=Kimbap Console Service
After=network.target

[Service]
Type=simple
User=%i
WorkingDirectory=%h/.kimbap-console
ExecStart=%h/.kimbap-console/scripts/start.sh
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target`;
    
    const servicePath = path.join(this.outputDir, 'kimbap-console.service');
    fs.writeFileSync(servicePath, serviceContent);
    
    // 创建安装脚本
    const installContent = `#!/bin/bash
# Kimbap Console Linux Installation Script
SCRIPT_DIR="$( cd "$( dirname "\${BASH_SOURCE[0]}" )" && pwd )"
TARGET_DIR="$HOME/.kimbap-console"

echo "Installing Kimbap Console to $TARGET_DIR..."

# 创建目标目录
mkdir -p "$TARGET_DIR"

# 复制文件
cp -r "$SCRIPT_DIR"/* "$TARGET_DIR/"

# 设置权限
chmod +x "$TARGET_DIR/start-kimbap-console"
chmod +x "$TARGET_DIR/scripts/start.sh"

# 创建桌面快捷方式
if [ -d "$HOME/Desktop" ]; then
    cp "$TARGET_DIR/Kimbap-Console.desktop" "$HOME/Desktop/"
fi

echo "✅ Kimbap Console installed successfully!"
echo "   Desktop shortcut: ~/Desktop/Kimbap-Console.desktop"
echo "   Command line: ~/.kimbap-console/start-kimbap-console"`;
    
    const installPath = path.join(this.outputDir, 'install.sh');
    fs.writeFileSync(installPath, installContent);
    fs.chmodSync(installPath, 0o755);
    
    console.log('✅ Linux executables created:');
    console.log('   - Kimbap-Console.desktop (desktop shortcut)');
    console.log('   - start-kimbap-console (launcher script)');
    console.log('   - kimbap-console.service (systemd service)');
    console.log('   - install.sh (installation script)');
  }
}

// 执行构建
if (require.main === module) {
  const targetArch = process.argv[2]; // 支持指定架构: node script.js x64
  const targetPlatform = process.argv[3]; // 支持指定平台: node script.js x64 linux
  
  if (targetArch && !['x64', 'arm64'].includes(targetArch)) {
    console.error('❌ Invalid architecture. Use: x64 or arm64');
    process.exit(1);
  }
  
  if (targetPlatform && !['darwin', 'linux', 'win32'].includes(targetPlatform)) {
    console.error('❌ Invalid platform. Use: darwin, linux, or win32');
    process.exit(1);
  }
  
  const builder = new CompleteStandaloneBuilder(targetArch, targetPlatform);
  builder.build();
}

module.exports = CompleteStandaloneBuilder;