#!/usr/bin/env node

/**
 *  - PostgreSQL
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

class DatabaseConfig {
  constructor() {
    this.rootDir = process.cwd();
    this.envPath = path.join(this.rootDir, '.env.local');
    
    // 
    this.configs = {
      local: {
        host: 'localhost',
        port: 5432,
        database: 'kimbap_db',
        username: 'kimbap',
        password: 'kimbap123'
      },
      cloud: {
        // 
        host: process.env.CLOUD_DB_HOST || '',
        port: parseInt(process.env.CLOUD_DB_PORT) || 5432,
        database: process.env.CLOUD_DB_NAME || 'kimbap_db',
        username: process.env.CLOUD_DB_USER || '',
        password: process.env.CLOUD_DB_PASSWORD || ''
      }
    };
  }

  /**
   *  psql 
   */
  findPsqlPath() {
    // 1. PostgreSQLpsql
    const embeddedPsql = path.join(this.rootDir, 'postgresql', 'bin', 
      process.platform === 'win32' ? 'psql.exe' : 'psql');
    if (fs.existsSync(embeddedPsql)) {
      return embeddedPsql;
    }
    
    // 2. PostgreSQL
    const commonPaths = [];
    if (process.platform === 'win32') {
      commonPaths.push(
        'C:\\Program Files\\PostgreSQL\\15\\bin\\psql.exe',
        'C:\\Program Files\\PostgreSQL\\14\\bin\\psql.exe',
        'C:\\Program Files\\PostgreSQL\\13\\bin\\psql.exe'
      );
    } else if (process.platform === 'darwin') {
      commonPaths.push(
        '/opt/homebrew/bin/psql',
        '/usr/local/bin/psql',
        '/opt/homebrew/opt/postgresql@15/bin/psql',
        '/usr/local/opt/postgresql@15/bin/psql'
      );
    } else { // Linux
      commonPaths.push(
        '/usr/bin/psql',
        '/usr/local/bin/psql',
        '/opt/postgresql/bin/psql'
      );
    }
    
    for (const psqlPath of commonPaths) {
      if (fs.existsSync(psqlPath)) {
        return psqlPath;
      }
    }
    
    // 3.  PATH psql
    try {
      execSync('psql --version', { stdio: 'pipe' });
      return 'psql';
    } catch (error) {
      return null; // psql
    }
  }

  /**
   * 
   */
  async detectBestConfig() {
    console.log('🔍 Detecting optimal database configuration...');
    
    // 1. 
    if (await this.testCloudConnection()) {
      console.log('✅ Using cloud PostgreSQL database');
      return this.generateDatabaseUrl('cloud');
    }
    
    // 2. PostgreSQL（）
    if (await this.startEmbeddedPostgreSQL()) {
      console.log('✅ Using embedded PostgreSQL database');
      return this.generateDatabaseUrl('local');
    }
    
    // 3. PostgreSQL
    if (await this.testLocalConnection()) {
      console.log('✅ Using local PostgreSQL database');
      return this.generateDatabaseUrl('local');
    }
    
    // 4. 
    this.printSetupInstructions();
    throw new Error('No PostgreSQL database available. Please set up a database first.');
  }

  /**
   * 
   */
  async testCloudConnection() {
    if (!this.configs.cloud.host || !this.configs.cloud.username) {
      return false;
    }
    
    try {
      const testUrl = this.generateDatabaseUrl('cloud');
      console.log('🔗 Testing cloud database connection...');
      
      // psql，psql
      const psqlPath = this.findPsqlPath();
      if (!psqlPath) {
        console.log('⚠️  No psql client found for cloud connection test');
        return false;
      }
      
      const testCommand = `"${psqlPath}" "${testUrl}" -c "SELECT 1;" --quiet`;
      execSync(testCommand, { stdio: 'pipe', timeout: 10000 });
      
      return true;
    } catch (error) {
      console.log('⚠️  Cloud database connection failed');
      return false;
    }
  }

  /**
   * 
   */
  async testLocalConnection() {
    try {
      const testUrl = this.generateDatabaseUrl('local');
      console.log('🔗 Testing local database connection...');
      
      // psql
      const psqlPath = this.findPsqlPath();
      if (!psqlPath) {
        console.log('⚠️  No psql client found');
        return false;
      }
      
      // ， Docker 
      for (let attempt = 1; attempt <= 6; attempt++) {
        try {
          const testCommand = `"${psqlPath}" "${testUrl}" -c "SELECT 1;" --quiet`;
          execSync(testCommand, { stdio: 'pipe', timeout: 3000 });
          
          if (attempt > 1) {
            console.log(`✅ Database connection established (attempt ${attempt})`);
          }
          return true;
        } catch (error) {
          if (attempt < 6) {
            console.log(`⏳ Waiting for database... (attempt ${attempt}/6)`);
            await new Promise(resolve => setTimeout(resolve, 2000)); // 2
          }
        }
      }
      
      console.log('⚠️  Local database connection failed after 6 attempts');
      return false;
    } catch (error) {
      console.log('⚠️  Local database connection failed');
      return false;
    }
  }

  /**
   * PostgreSQL
   */
  async startEmbeddedPostgreSQL() {
    const pgDir = path.join(this.rootDir, 'postgresql');
    const pgBin = path.join(pgDir, 'bin');
    const pgData = path.join(pgDir, 'data');
    
    // PostgreSQL
    const pgExecutable = path.join(pgBin, process.platform === 'win32' ? 'postgres.exe' : 'postgres');
    if (!fs.existsSync(pgExecutable)) {
      return false;
    }
    
    try {
      console.log('🚀 Starting embedded PostgreSQL...');
      
      // ，
      if (!fs.existsSync(pgData)) {
        console.log('🔧 Initializing embedded database...');
        
        const initdbPath = path.join(pgBin, process.platform === 'win32' ? 'initdb.exe' : 'initdb');
        const initCommand = `"${initdbPath}" -D "${pgData}" -U kimbap --auth-local=trust --auth-host=md5`;
        execSync(initCommand, { stdio: 'inherit' });
      }
      
      // PostgreSQL
      const pgCtlPath = path.join(pgBin, process.platform === 'win32' ? 'pg_ctl.exe' : 'pg_ctl');
      const logPath = path.join(pgDir, 'postgresql.log');
      const startCommand = `"${pgCtlPath}" -D "${pgData}" -l "${logPath}" start`;
      execSync(startCommand, { stdio: 'pipe' });
      
      // 
      await this.sleep(3000);
      
      // ，
      try {
        const testUrl = this.generateDatabaseUrl('local');
        const psqlPath = path.join(pgBin, process.platform === 'win32' ? 'psql.exe' : 'psql');
        execSync(`"${psqlPath}" "${testUrl}" -c "SELECT 1;" --quiet`, { stdio: 'pipe' });
      } catch (error) {
        // ，
        console.log('🗃️  Creating application database...');
        const createdbPath = path.join(pgBin, process.platform === 'win32' ? 'createdb.exe' : 'createdb');
        const createCommand = `"${createdbPath}" -U kimbap kimbap_db`;
        execSync(createCommand, { stdio: 'pipe' });
      }
      
      return true;
    } catch (error) {
      console.log('⚠️  Embedded PostgreSQL startup failed:', error.message);
      return false;
    }
  }

  /**
   * URL
   */
  generateDatabaseUrl(type) {
    const config = this.configs[type];
    const encodedUsername = encodeURIComponent(config.username);
    const encodedPassword = encodeURIComponent(config.password);
    return `postgresql://${encodedUsername}:${encodedPassword}@${config.host}:${config.port}/${config.database}`;
  }

  /**
   * 
   */
  updateEnvironmentFile(databaseUrl) {
    console.log('📝 Updating environment configuration...');
    
    let envContent = '';
    
    // 
    if (fs.existsSync(this.envPath)) {
      envContent = fs.readFileSync(this.envPath, 'utf8');
    }
    
    // DATABASE_URL
    const lines = envContent.split('\n');
    let found = false;
    
    for (let i = 0; i < lines.length; i++) {
      if (lines[i].startsWith('DATABASE_URL=')) {
        lines[i] = `DATABASE_URL=${databaseUrl}`;
        found = true;
        break;
      }
    }
    
    if (!found) {
      lines.push(`DATABASE_URL=${databaseUrl}`);
    }
    
    // 
    const requiredVars = {
      'NODE_ENV': 'production'
    };
    
    for (const [key, defaultValue] of Object.entries(requiredVars)) {
      const exists = lines.some(line => line.startsWith(`${key}=`));
      if (!exists) {
        lines.push(`${key}=${defaultValue}`);
      }
    }
    
    // 
    fs.writeFileSync(this.envPath, lines.filter(line => line.trim()).join('\n') + '\n');
    
    console.log('✅ Environment configuration updated');
  }

  /**
   * 
   */
  async runMigrations(databaseUrl) {
    console.log('🗄️  Running database migrations...');
    
    try {
      // DATABASE_URL
      process.env.DATABASE_URL = databaseUrl;
      
      //  Prisma 
      const nodeModulesPath = path.join(this.rootDir, 'node_modules');
      const prismaPath = path.join(nodeModulesPath, '.bin', 'prisma');
      
      if (fs.existsSync(prismaPath)) {
        // Prisma
        execSync(`"${prismaPath}" generate`, { stdio: 'inherit' });
        
        // 
        execSync(`"${prismaPath}" migrate deploy`, { stdio: 'inherit' });
      } else {
        // ：node
        const prismaClientGenerator = path.join(nodeModulesPath, 'prisma', 'build', 'index.js');
        if (fs.existsSync(prismaClientGenerator)) {
          execSync(`node "${prismaClientGenerator}" generate`, { stdio: 'inherit' });
        } else {
          console.log('⚠️  Prisma client not found, skipping generation');
        }
      }
      
      console.log('✅ Database migrations completed');
      return true;
    } catch (error) {
      console.error('❌ Database migration failed:', error.message);
      return false;
    }
  }

  /**
   * 
   */
  printSetupInstructions() {
    console.log('\n📖 PostgreSQL Setup Instructions:\n');
    
    console.log('🐳 Option 1: Docker (Recommended)');
    console.log('docker run --name kimbap-postgres \\');
    console.log('  -e POSTGRES_USER=kimbap \\');
    console.log('  -e POSTGRES_PASSWORD=kimbap123 \\');
    console.log('  -e POSTGRES_DB=kimbap_db \\');
    console.log('  -p 5432:5432 -d postgres:16\n');
    
    console.log('🏠 Option 2: Local Installation');
    if (process.platform === 'darwin') {
      console.log('brew install postgresql@16');
      console.log('brew services start postgresql@16');
      console.log('createdb kimbap_db');
    } else if (process.platform === 'linux') {
      console.log('sudo apt update');
      console.log('sudo apt install postgresql-16');
      console.log('sudo systemctl start postgresql');
      console.log('sudo -u postgres createdb kimbap_db');
    } else {
      console.log('Download PostgreSQL 16 from: https://www.postgresql.org/download/windows/');
    }
    
    console.log('\n☁️  Option 3: Cloud Database');
    console.log('Set these environment variables:');
    console.log('CLOUD_DB_HOST=your-cloud-host');
    console.log('CLOUD_DB_USER=your-username');
    console.log('CLOUD_DB_PASSWORD=your-password');
    console.log('CLOUD_DB_NAME=your-database\n');
  }

  /**
   * 
   */
  sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  /**
   * 
   */
  async validateConfig() {
    try {
      const databaseUrl = await this.detectBestConfig();
      this.updateEnvironmentFile(databaseUrl);
      
      if (await this.runMigrations(databaseUrl)) {
        console.log('🎉 Database configuration completed successfully!');
        console.log(`📊 Using: ${databaseUrl.replace(/:[^:]*@/, ':***@')}`);
        return true;
      }
      
      return false;
    } catch (error) {
      console.error('❌ Database configuration failed:', error.message);
      return false;
    }
  }
}

// CLI 
if (require.main === module) {
  const config = new DatabaseConfig();
  
  const action = process.argv[2] || 'validate';
  
  switch (action) {
    case 'validate':
    case 'setup':
      config.validateConfig().then(success => {
        process.exit(success ? 0 : 1);
      });
      break;
      
    case 'test':
      config.detectBestConfig().then(url => {
        console.log('Database URL:', url.replace(/:[^:]*@/, ':***@'));
      }).catch(error => {
        console.error('Error:', error.message);
        process.exit(1);
      });
      break;
      
    case 'help':
      console.log('Usage: node database-config.js [action]');
      console.log('Actions:');
      console.log('  validate - Auto-detect and configure database (default)');
      console.log('  test     - Test database configuration');
      console.log('  help     - Show this help');
      break;
      
    default:
      console.error('Unknown action:', action);
      process.exit(1);
  }
}

module.exports = DatabaseConfig;
