#!/usr/bin/env node

/**
 *  - 
 * ：、PostgreSQL、
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
    
    // 
    console.log('  📦 Copying .next build output...');
    this.copyDir(path.join(this.rootDir, '.next'), path.join(appDir, '.next'));
    
    console.log('  📁 Copying public assets...');
    this.copyDir(path.join(this.rootDir, 'public'), path.join(appDir, 'public'));
    
    console.log('  🗄️  Copying prisma schema...');
    this.copyDir(path.join(this.rootDir, 'prisma'), path.join(appDir, 'prisma'));
    
    //  node_modules ()
    console.log('  📚 Copying node_modules (this may take a while)...');
    this.copyDir(path.join(this.rootDir, 'node_modules'), path.join(appDir, 'node_modules'));
    
    // 
    console.log('  ⚙️  Copying configuration files...');
    fs.copyFileSync(path.join(this.rootDir, 'package.json'), path.join(appDir, 'package.json'));
    
    //  next.config.mjs
    const portableNextConfig = `/** @type {import('next').NextConfig} */
const nextConfig = {
  //  - API routes
  output: undefined,

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
  poweredByHeader: false
}

export default nextConfig`;

    fs.writeFileSync(path.join(appDir, 'next.config.mjs'), portableNextConfig);
    
    // 
    const prodEnv = `NODE_ENV=production
DATABASE_URL=postgresql://kimbap:kimbap123@localhost:5432/kimbap_db`;
    
    fs.writeFileSync(path.join(appDir, '.env.local'), prodEnv);
    
    console.log('✅ Application files copied');
  }

  async setupPostgreSQL() {
    console.log('🔍 Checking for PostgreSQL installations...');
    
    try {
      //  PostgreSQL
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
        // ，
      }
    }
    
    //  which 
    try {
      const { execSync } = require('child_process');
      const result = execSync('which postgres', { encoding: 'utf8', stdio: 'pipe' });
      const pgPath = result.trim();
      if (pgPath && fs.existsSync(pgPath)) {
        console.log(`🔍 Found via which: ${pgPath}`);
        return path.dirname(pgPath);
      }
    } catch (error) {
      // which  postgres  PATH 
    }
    
    return null;
  }

  async copySystemPostgreSQL(systemPgPath) {
    console.log('📋 Copying system PostgreSQL binaries...');
    
    const pgDir = path.join(this.outputDir, 'postgresql');
    fs.mkdirSync(pgDir, { recursive: true });
    
    //  PostgreSQL 
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

    // 
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
            for (const libFile of libFiles.slice(0, 15)) { // 
              try {
                const srcPath = path.join(libDir, libFile);
                const destPath = path.join(destLibDir, libFile);
                fs.copyFileSync(srcPath, destPath);
                copiedLibs++;
              } catch (error) {
                // 
              }
            }
            
            if (copiedLibs > 0) {
              console.log(`✅ Copied ${copiedLibs} PostgreSQL libraries`);
            }
            break;
          }
        } catch (error) {
          // 
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

#  PostgreSQL 
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
    const instructions = `# PostgreSQL Setup for Kimbap Console

## 
 PostgreSQL 。

## 

### 1:  Docker () ⭐
\`\`\`bash
docker run --name kimbap-postgres \\
  -e POSTGRES_USER=kimbap \\
  -e POSTGRES_PASSWORD=kimbap123 \\
  -e POSTGRES_DB=kimbap_db \\
  -p 5432:5432 -d postgres:16
\`\`\`

### 2:  PostgreSQL

#### macOS ( Homebrew):
\`\`\`bash
#  PostgreSQL
brew install postgresql@16
brew services start postgresql@16

# 
createuser kimbap
createdb -U kimbap kimbap_db
\`\`\`

#### Linux (Ubuntu/Debian):
\`\`\`bash
sudo apt update
sudo apt install postgresql-16 postgresql-client-16
sudo systemctl start postgresql

# 
sudo -u postgres createuser kimbap
sudo -u postgres createdb -O kimbap kimbap_db
\`\`\`

#### Windows:
1.  https://www.postgresql.org/download/windows/  PostgreSQL 16
2. 
3.  pgAdmin  \`kimbap\`  \`kimbap_db\`

### 3: 
 \`app/.env.local\`  DATABASE_URL：
\`\`\`
DATABASE_URL=postgresql://username:password@host:port/database
\`\`\`

## 
:
\`\`\`bash
psql "postgresql://kimbap:kimbap123@localhost:5432/kimbap_db" -c "SELECT version();"
\`\`\`

## 

### 
-  PostgreSQL 
-  5432 
- 

### 
- 
-  \`kimbap\`  \`kimbap_db\` 

### 
-  \`createdb -U kimbap kimbap_db\` 
-  \`.env.local\` 

---

📖 : https://www.postgresql.org/docs/
🐳 Docker : https://hub.docker.com/_/postgres
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
title Kimbap Console
echo ========================================
echo         Kimbap Console Starting
echo ========================================
echo.

REM 
set "SCRIPT_DIR=%~dp0"
set "PATH=%SCRIPT_DIR%node\\bin;%PATH%"
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

REM PostgreSQL
echo 🔍 Checking PostgreSQL connection...
echo Make sure PostgreSQL is running on localhost:5432
echo Database: kimbap_db, User: kimbap, Password: kimbap123
echo.

REM 
echo 🚀 Starting Kimbap Console...
cd /d "%SCRIPT_DIR%app"
..\\node\\bin\\node.exe node_modules\\next\\dist\\bin\\next start -p 3000

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

# Kimbap Console 
echo "========================================"
echo "       Kimbap Console Starting"
echo "========================================"
echo

# 
SCRIPT_DIR="$( cd "$( dirname "\${BASH_SOURCE[0]}" )" && pwd )"
SCRIPT_DIR="$( dirname "$SCRIPT_DIR" )"

# 
export PATH="$SCRIPT_DIR/node/bin:$PATH"
export DATABASE_URL="postgresql://kimbap:kimbap123@localhost:5432/kimbap_db"
export NODE_ENV="production"

# 
if lsof -Pi :3000 -sTCP:LISTEN -t >/dev/null ; then
    echo "⚠️  Port 3000 is already in use!"
    echo "Please close the application using port 3000 and try again."
    exit 1
fi

# PostgreSQL
echo "🔍 Checking PostgreSQL connection..."
echo "Make sure PostgreSQL is running on localhost:5432"
echo "Database: kimbap_db, User: kimbap, Password: kimbap123"
echo

# 
echo "🚀 Starting Kimbap Console..."
cd "$SCRIPT_DIR/app"
../node/bin/node node_modules/next/dist/bin/next start -p 3000

echo
echo "✅ Kimbap Console stopped."`;

    const scriptPath = path.join(this.outputDir, 'scripts', 'start.sh');
    fs.writeFileSync(scriptPath, script);
    
    // 
    fs.chmodSync(scriptPath, 0o755);
  }

  async createConfigFiles() {
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
    const readme = `# Kimbap Console 

## 

### 
1. **PostgreSQL**:  PostgreSQL 
   - 1:  Docker ()
     \`\`\`bash
     docker run --name kimbap-postgres -e POSTGRES_USER=kimbap -e POSTGRES_PASSWORD=kimbap123 -e POSTGRES_DB=kimbap_db -p 5432:5432 -d postgres:16
     \`\`\`
   - 2:  PostgreSQL  kimbap_db

### Windows
1.  PostgreSQL 
2.  \`scripts/start.bat\` 
3.  http://localhost:3000

### Mac/Linux
1.  PostgreSQL 
2. ，
3.  \`./scripts/start.sh\`
4.  http://localhost:3000

## 

- ****:  2GB RAM
- ****:  500MB 
- ****: PostgreSQL 16.x ( Docker )
- ****: 
  - Windows 10 
  - macOS 10.14 
  - Ubuntu 18.04 

## 

\`\`\`
kimbap-console/
├── app/                #  (.next, node_modules, etc.)
├── node/              # Node.js 
├── postgresql/         # PostgreSQL 
├── scripts/           # 
├── config/            # 
└── README.txt         # 
\`\`\`

## 

### 1.  3000 
-  3000 
-  config/config.json 

### 2. 
-  PostgreSQL  localhost:5432
-  kimbap/kimbap123
-  kimbap_db 

### 3. 
- 
- 

## 

 Docker PostgreSQL，：
\`\`\`bash
docker exec kimbap-postgres pg_dump -U kimbap kimbap_db > backup.sql
\`\`\`

## 

1.  PostgreSQL  ( Docker)：\`docker stop kimbap-postgres && docker rm kimbap-postgres\`
2. 

## 

，：
1. 
2.  GitHub  Issue
3. 

---

Kimbap Console v1.0.0
: ${new Date().toISOString().split('T')[0]}
: ${this.platform}-${this.arch}`;

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

// 
if (require.main === module) {
  const completer = new BuildCompleter();
  completer.complete();
}

module.exports = BuildCompleter;