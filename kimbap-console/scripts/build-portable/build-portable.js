#!/usr/bin/env node

/**
 * Kimbap Console 
 * 
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
    
    // 
    this.nodeVersion = '20.11.0';  // LTS version
    this.postgresVersion = '16.1';
    
    console.log(`🚀 Building portable package for ${this.platform}-${this.arch}`);
  }

  async build() {
    try {
      console.log('📦 Starting portable build process...');
      
      // 1. 
      await this.prepare();
      
      // 2.  Next.js 
      await this.buildNextApp();
      
      // 3.  Node.js 
      await this.downloadNode();
      
      // 4.  PostgreSQL 
      await this.downloadPostgreSQL();
      
      // 5. 
      await this.copyAppFiles();
      
      // 6. 
      await this.createStartupScripts();
      
      // 7. 
      await this.createConfigFiles();
      
      // 8. 
      await this.createDocumentation();
      
      // 9. 
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
    
    // 
    if (fs.existsSync(this.outputDir)) {
      fs.rmSync(this.outputDir, { recursive: true, force: true });
    }
    
    // 
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
      // （devDependencies）
      execSync('npm ci', { stdio: 'inherit' });
      
      // Next.js（）
      await this.createPortableNextConfig();
      
      // 
      execSync('NODE_ENV=production NEXT_CONFIG=next.config.portable.mjs npm run build', { stdio: 'inherit' });
      
      // 
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
  //  - API routesSSR
  output: undefined, // API routes

  // 
  images: {
    unoptimized: true,
    formats: ['image/webp', 'image/avif'],
    minimumCacheTTL: 60,
    dangerouslyAllowSVG: true,
    contentSecurityPolicy: "default-src 'self'; script-src 'none'; sandbox;"
  },

  // 
  experimental: {
    optimizeCss: false,
    optimizePackageImports: ['@radix-ui/react-*', 'lucide-react']
  },

  // TypeScript 
  typescript: {
    ignoreBuildErrors: true
  },

  // ESLint 
  eslint: {
    ignoreDuringBuilds: false
  },

  // 
  compress: true,

  // 
  reactStrictMode: true,

  //  Next.js 
  poweredByHeader: false,

  // Webpack 
  webpack: (config, { dev, isServer }) => {
    // Node.js  fallback 
    if (!isServer) {
      config.resolve.fallback = {
        ...config.resolve.fallback,
        fs: false,
        net: false,
        tls: false
      }
    }

    // 
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

    // 
    const originalConfig = path.join(this.rootDir, 'next.config.mjs');
    const backupConfig = path.join(this.rootDir, 'next.config.mjs.backup');
    fs.copyFileSync(originalConfig, backupConfig);

    // 
    const portableConfigPath = path.join(this.rootDir, 'next.config.portable.mjs');
    fs.writeFileSync(portableConfigPath, portableConfig);
    
    // 
    fs.copyFileSync(portableConfigPath, originalConfig);
  }

  async restoreOriginalNextConfig() {
    const originalConfig = path.join(this.rootDir, 'next.config.mjs');
    const backupConfig = path.join(this.rootDir, 'next.config.mjs.backup');
    const portableConfig = path.join(this.rootDir, 'next.config.portable.mjs');
    
    // 
    if (fs.existsSync(backupConfig)) {
      fs.copyFileSync(backupConfig, originalConfig);
      fs.unlinkSync(backupConfig);
    }
    
    // 
    if (fs.existsSync(portableConfig)) {
      fs.unlinkSync(portableConfig);
    }
  }

  async downloadNode() {
    console.log('📥 Downloading and extracting Node.js...');
    
    const downloader = new DependencyDownloader();
    const extractor = new ExtractUtils();
    
    try {
      //  Node.js
      await downloader.downloadNode();
      
      // 
      const nodeUrl = downloader.getNodeDownloadUrl();
      const fileName = path.basename(nodeUrl);
      const archivePath = path.join(downloader.tempDir, fileName);
      const extractDir = path.join(this.outputDir, 'node');
      
      // 
      await extractor.extractArchive(archivePath, extractDir);
      
      // 
      await extractor.organizeExtractedFiles(extractDir);
      
      // 
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
      //  PostgreSQL，
      const systemPgPath = this.findSystemPostgreSQL();
      
      if (systemPgPath) {
        console.log(`📋 Found system PostgreSQL at: ${systemPgPath}`);
        await this.copySystemPostgreSQL(systemPgPath);
      } else {
        console.log('📦 Downloading portable PostgreSQL...');
        await this.downloadPortablePostgreSQL();
      }
      
      // 
      await this.createDatabaseInitScript();
      
      console.log('✅ PostgreSQL setup completed');
    } catch (error) {
      console.warn('⚠️  PostgreSQL setup failed, falling back to external database requirement');
      console.warn('Error:', error.message);
      
      // 
      await this.createPostgreSQLInstructions();
    }
  }

  findSystemPostgreSQL() {
    //  PostgreSQL
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
    
    //  which/where 
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
    
    // 
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

    // 
    await this.copyPostgreSQLLibraries(systemPgPath, pgDir);
  }

  async copyPostgreSQLLibraries(systemPgPath, pgDir) {
    //  PostgreSQL 
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
            
            for (const libFile of libFiles.slice(0, 10)) { // 
              try {
                const srcPath = path.join(libDir, libFile);
                const destPath = path.join(destLibDir, libFile);
                fs.copyFileSync(srcPath, destPath);
              } catch (error) {
                // 
              }
            }
            
            console.log(`✅ Copied ${libFiles.length} PostgreSQL libraries`);
            break;
          }
        } catch (error) {
          // 
        }
      }
    }
  }

  async downloadPortablePostgreSQL() {
    //  PostgreSQL，
    // ，
    console.log('⚠️  Portable PostgreSQL download not implemented yet');
    console.log('📝 Creating setup instructions instead...');
    
    await this.createPostgreSQLInstructions();
  }

  async createDatabaseInitScript() {
    console.log('📝 Creating database initialization scripts...');
    
    const scriptsDir = path.join(this.outputDir, 'scripts');
    
    // 
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

## 
 PostgreSQL 。

## 

### 1:  Docker ()
\`\`\`bash
docker run --name kimbap-postgres \\
  -e POSTGRES_USER=kimbap \\
  -e POSTGRES_PASSWORD=kimbap123 \\
  -e POSTGRES_DB=kimbap_db \\
  -p 5432:5432 -d postgres:16
\`\`\`

### 2:  PostgreSQL

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
1.  PostgreSQL 16 
2. 
3.  pgAdmin  kimbap_db

### 3: 
 \`app/.env.local\`  DATABASE_URL 。

## 
:
\`\`\`bash
psql postgresql://kimbap:kimbap123@localhost:5432/kimbap_db -c "SELECT version();"
\`\`\`

---
，: https://www.postgresql.org/download/
`;

    const pgDir = path.join(this.outputDir, 'postgresql');
    fs.writeFileSync(path.join(pgDir, 'SETUP.md'), instructions);
  }

  async copyAppFiles() {
    console.log('📋 Copying application files...');
    
    const appDir = path.join(this.outputDir, 'app');
    
    // 
    this.copyDir(path.join(this.rootDir, '.next'), path.join(appDir, '.next'));
    this.copyDir(path.join(this.rootDir, 'public'), path.join(appDir, 'public'));
    this.copyDir(path.join(this.rootDir, 'prisma'), path.join(appDir, 'prisma'));
    this.copyDir(path.join(this.rootDir, 'scripts'), path.join(appDir, 'scripts'));
    
    // 
    fs.copyFileSync(path.join(this.rootDir, 'package.json'), path.join(appDir, 'package.json'));
    fs.copyFileSync(path.join(this.rootDir, 'next.config.mjs'), path.join(appDir, 'next.config.mjs'));
    
    // 
    process.chdir(appDir);
    execSync('npm ci --production', { stdio: 'inherit' });
    
    // （）
    const prodEnv = `NODE_ENV=production
# URL
DATABASE_URL=postgresql://kimbap:kimbap123@localhost:5432/kimbap_db
# （）
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

REM 
set "SCRIPT_DIR=%~dp0"
set "PATH=%SCRIPT_DIR%postgresql\\bin;%SCRIPT_DIR%node;%PATH%"
set "PGDATA=%SCRIPT_DIR%postgresql\\data"
set "DATABASE_URL=postgresql://kimbap:kimbap123@localhost:5432/kimbap_db"
set "NODE_ENV=production"

REM 
netstat -an | find "3000" > nul
if %errorlevel% == 0 (
    echo ⚠️  Port 3000 is already in use!
    echo Please close the application using port 3000 and try again.
    pause
    exit /b 1
)

REM （）
if not exist "%PGDATA%" (
    echo 🔧 Initializing database for first time...
    echo This may take a few moments...
    
    REM 
    mkdir "%PGDATA%"
    
    REM 
    initdb -D "%PGDATA%" -U kimbap --auth-local=trust --auth-host=md5
    if %errorlevel% neq 0 (
        echo ❌ Database initialization failed!
        pause
        exit /b 1
    )
    
    REM 
    pg_ctl -D "%PGDATA%" -l "%SCRIPT_DIR%logs\\postgresql.log" start
    if %errorlevel% neq 0 (
        echo ❌ Failed to start database!
        pause
        exit /b 1
    )
    
    REM 
    timeout /t 5 > nul
    
    REM 
    createdb -U kimbap kimbap_db
    if %errorlevel% neq 0 (
        echo ❌ Failed to create database!
        pause
        exit /b 1
    )
    
    REM 
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
    
    REM 
    pg_ctl -D "%PGDATA%" -l "%SCRIPT_DIR%logs\\postgresql.log" start
    if %errorlevel% neq 0 (
        echo ❌ Failed to start database!
        pause
        exit /b 1
    )
)

REM 
timeout /t 3 > nul

REM 
echo 🚀 Starting Kimbap Console...
cd /d "%SCRIPT_DIR%app"
..\\node\\node.exe node_modules\\next\\dist\\bin\\next start -p 3000

REM ，
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

# Kimbap Console  -  PostgreSQL 
echo "========================================"
echo "       Kimbap Console Starting"
echo "========================================"
echo

# 
SCRIPT_DIR="$( cd "$( dirname "\${BASH_SOURCE[0]}" )" && pwd )"
SCRIPT_DIR="$( dirname "$SCRIPT_DIR" )"

# 
export PATH="$SCRIPT_DIR/node/bin:$PATH"
export NODE_ENV="production"

# 
mkdir -p "$SCRIPT_DIR/logs"
mkdir -p "$SCRIPT_DIR/postgresql/data"

# 
if lsof -Pi :3000 -sTCP:LISTEN -t >/dev/null ; then
    echo "⚠️  Port 3000 is already in use!"
    echo "Please close the application using port 3000 and try again."
    exit 1
fi

# PostgreSQL 
setup_postgresql() {
    echo "🔍 Checking PostgreSQL setup..."
    
    #  PostgreSQL
    if [ -f "$SCRIPT_DIR/postgresql/bin/postgres" ]; then
        echo "✅ Found embedded PostgreSQL"
        export PATH="$SCRIPT_DIR/postgresql/bin:$PATH"
        export PGDATA="$SCRIPT_DIR/postgresql/data"
        export DATABASE_URL="postgresql://kimbap:kimbap123@localhost:5432/kimbap_db"
        
        # （）
        if [ ! -d "$PGDATA/base" ]; then
            echo "🔧 Initializing embedded database..."
            
            # 
            if [ -f "$SCRIPT_DIR/scripts/init-db.sh" ]; then
                "$SCRIPT_DIR/scripts/init-db.sh"
            else
                # 
                initdb -D "$PGDATA" -U kimbap --auth-local=trust --auth-host=md5
                pg_ctl -D "$PGDATA" -l "$SCRIPT_DIR/logs/postgresql.log" start
                sleep 5
                createdb -U kimbap kimbap_db
            fi
        else
            echo "🔄 Starting embedded database..."
            pg_ctl -D "$PGDATA" -l "$SCRIPT_DIR/logs/postgresql.log" start
        fi
        
        # 
        sleep 3
        
        # 
        cd "$SCRIPT_DIR/app"
        ../node/bin/node node_modules/.bin/prisma migrate deploy
        
        echo "✅ Embedded PostgreSQL ready"
        return 0
        
    else
        echo "📝 No embedded PostgreSQL found"
        echo "🔍 Checking for external PostgreSQL..."
        
        #  PostgreSQL 
        if command -v psql >/dev/null 2>&1; then
            echo "✅ Found system PostgreSQL"
            export DATABASE_URL="postgresql://kimbap:kimbap123@localhost:5432/kimbap_db"
            
            # 
            if psql "$DATABASE_URL" -c "SELECT 1;" >/dev/null 2>&1; then
                echo "✅ Database connection successful"
                
                # 
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

# 
cleanup() {
    echo ""
    echo "🛑 Stopping services..."
    
    #  PostgreSQL，
    if [ -f "$SCRIPT_DIR/postgresql/bin/postgres" ] && [ -d "$PGDATA" ]; then
        pg_ctl -D "$PGDATA" stop -m fast >/dev/null 2>&1
        echo "✅ PostgreSQL stopped"
    fi
    
    echo "✅ Kimbap Console stopped."
}

# 
trap cleanup EXIT INT TERM

#  PostgreSQL
if ! setup_postgresql; then
    echo ""
    echo "❌ PostgreSQL setup failed. Please resolve database issues and try again."
    exit 1
fi

# 
echo "🚀 Starting Kimbap Console..."
cd "$SCRIPT_DIR/app"
../node/bin/node node_modules/next/dist/bin/next start -p 3000`;

    const scriptPath = path.join(this.outputDir, 'scripts', 'start.sh');
    fs.writeFileSync(scriptPath, script);
    
    // 
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
    
    const readme = `# Kimbap Console 

## 

### Windows
1.  \`scripts/start.bat\` 
2. （）
3.  http://localhost:3000

### Mac/Linux
1. ，
2.  \`./scripts/start.sh\`
3. （）
4.  http://localhost:3000

## 

- ****:  2GB RAM
- ****:  500MB 
- ****: 
  - Windows 10 
  - macOS 10.14 
  - Ubuntu 18.04 

## 

\`\`\`
kimbap-console/
├── app/                # 
├── postgresql/         # PostgreSQL 
├── node/              # Node.js 
├── scripts/           # 
├── config/            # 
├── logs/              # 
└── README.txt         # 
\`\`\`

## 

### 1.  3000 
-  3000 
-  config/config.json 

### 2. 
- 
-  logs/postgresql.log 

### 3. 
- 
-  logs/ 

## 

 \`postgresql/data\` ，。

## 

。

## 

，：
1. logs/ 
2.  GitHub  Issue
3. 

---

Kimbap Console v1.0.0
: ${new Date().toISOString().split('T')[0]}
: ${this.platform}-${this.arch}`;

    fs.writeFileSync(path.join(this.outputDir, 'README.txt'), readme);
    
    console.log('✅ Documentation created');
  }

  // 
  getNodeDownloadUrl() {
    const baseUrl = 'https://nodejs.org/dist';
    const fileName = this.platform === 'win32' 
      ? `node-v${this.nodeVersion}-win-${this.arch}.zip`
      : `node-v${this.nodeVersion}-${this.platform}-${this.arch}.tar.xz`;
    
    return `${baseUrl}/v${this.nodeVersion}/${fileName}`;
  }

  getPostgreSQLDownloadUrl() {
    //  PostgreSQL 
    // 
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
          // 
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
    // 
    //  node  zlib  yauzl, tar 
    console.log(`Extracting ${archive} to ${destination}`);
    
    // ：
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

// 
if (require.main === module) {
  const builder = new PortableBuilder();
  builder.build();
}

module.exports = PortableBuilder;