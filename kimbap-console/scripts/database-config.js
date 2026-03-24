#!/usr/bin/env node

/**
 * 数据库配置管理器 - 支持本地和云端PostgreSQL自动切换
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

class DatabaseConfig {
  constructor() {
    this.rootDir = process.cwd();
    this.envPath = path.join(this.rootDir, '.env.local');
    
    // 默认配置
    this.configs = {
      local: {
        host: 'localhost',
        port: 5432,
        database: 'kimbap_db',
        username: 'kimbap',
        password: 'kimbap123'
      },
      cloud: {
        // 云端配置将从环境变量或配置文件读取
        host: process.env.CLOUD_DB_HOST || '',
        port: parseInt(process.env.CLOUD_DB_PORT) || 5432,
        database: process.env.CLOUD_DB_NAME || 'kimbap_db',
        username: process.env.CLOUD_DB_USER || '',
        password: process.env.CLOUD_DB_PASSWORD || ''
      }
    };
  }

  /**
   * 查找可用的 psql 路径
   */
  findPsqlPath() {
    // 1. 优先使用内嵌PostgreSQL的psql
    const embeddedPsql = path.join(this.rootDir, 'postgresql', 'bin', 
      process.platform === 'win32' ? 'psql.exe' : 'psql');
    if (fs.existsSync(embeddedPsql)) {
      return embeddedPsql;
    }
    
    // 2. 尝试常见的PostgreSQL安装路径
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
    
    // 3. 检查系统 PATH 中是否有psql
    try {
      execSync('psql --version', { stdio: 'pipe' });
      return 'psql';
    } catch (error) {
      return null; // 没有找到任何可用的psql
    }
  }

  /**
   * 自动检测最佳数据库配置
   */
  async detectBestConfig() {
    console.log('🔍 Detecting optimal database configuration...');
    
    // 1. 检查是否有云端数据库配置
    if (await this.testCloudConnection()) {
      console.log('✅ Using cloud PostgreSQL database');
      return this.generateDatabaseUrl('cloud');
    }
    
    // 2. 优先尝试启动内嵌PostgreSQL（对于打包版本）
    if (await this.startEmbeddedPostgreSQL()) {
      console.log('✅ Using embedded PostgreSQL database');
      return this.generateDatabaseUrl('local');
    }
    
    // 3. 检查本地PostgreSQL是否可用
    if (await this.testLocalConnection()) {
      console.log('✅ Using local PostgreSQL database');
      return this.generateDatabaseUrl('local');
    }
    
    // 4. 提供设置指导
    this.printSetupInstructions();
    throw new Error('No PostgreSQL database available. Please set up a database first.');
  }

  /**
   * 测试云端数据库连接
   */
  async testCloudConnection() {
    if (!this.configs.cloud.host || !this.configs.cloud.username) {
      return false;
    }
    
    try {
      const testUrl = this.generateDatabaseUrl('cloud');
      console.log('🔗 Testing cloud database connection...');
      
      // 优先使用内嵌psql，否则使用系统psql
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
   * 测试本地数据库连接
   */
  async testLocalConnection() {
    try {
      const testUrl = this.generateDatabaseUrl('local');
      console.log('🔗 Testing local database connection...');
      
      // 检查是否有可用的psql
      const psqlPath = this.findPsqlPath();
      if (!psqlPath) {
        console.log('⚠️  No psql client found');
        return false;
      }
      
      // 增加重试逻辑，特别是针对 Docker 容器启动
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
            await new Promise(resolve => setTimeout(resolve, 2000)); // 等待2秒
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
   * 尝试启动内嵌PostgreSQL
   */
  async startEmbeddedPostgreSQL() {
    const pgDir = path.join(this.rootDir, 'postgresql');
    const pgBin = path.join(pgDir, 'bin');
    const pgData = path.join(pgDir, 'data');
    
    // 检查是否有内嵌PostgreSQL
    const pgExecutable = path.join(pgBin, process.platform === 'win32' ? 'postgres.exe' : 'postgres');
    if (!fs.existsSync(pgExecutable)) {
      return false;
    }
    
    try {
      console.log('🚀 Starting embedded PostgreSQL...');
      
      // 如果数据目录不存在，初始化数据库
      if (!fs.existsSync(pgData)) {
        console.log('🔧 Initializing embedded database...');
        
        const initdbPath = path.join(pgBin, process.platform === 'win32' ? 'initdb.exe' : 'initdb');
        const initCommand = `"${initdbPath}" -D "${pgData}" -U kimbap --auth-local=trust --auth-host=md5`;
        execSync(initCommand, { stdio: 'inherit' });
      }
      
      // 启动PostgreSQL
      const pgCtlPath = path.join(pgBin, process.platform === 'win32' ? 'pg_ctl.exe' : 'pg_ctl');
      const logPath = path.join(pgDir, 'postgresql.log');
      const startCommand = `"${pgCtlPath}" -D "${pgData}" -l "${logPath}" start`;
      execSync(startCommand, { stdio: 'pipe' });
      
      // 等待启动
      await this.sleep(3000);
      
      // 检查数据库是否存在，不存在则创建
      try {
        const testUrl = this.generateDatabaseUrl('local');
        const psqlPath = path.join(pgBin, process.platform === 'win32' ? 'psql.exe' : 'psql');
        execSync(`"${psqlPath}" "${testUrl}" -c "SELECT 1;" --quiet`, { stdio: 'pipe' });
      } catch (error) {
        // 数据库不存在，创建它
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
   * 生成数据库连接URL
   */
  generateDatabaseUrl(type) {
    const config = this.configs[type];
    return `postgresql://${config.username}:${config.password}@${config.host}:${config.port}/${config.database}`;
  }

  /**
   * 更新环境变量文件
   */
  updateEnvironmentFile(databaseUrl) {
    console.log('📝 Updating environment configuration...');
    
    let envContent = '';
    
    // 读取现有环境变量
    if (fs.existsSync(this.envPath)) {
      envContent = fs.readFileSync(this.envPath, 'utf8');
    }
    
    // 更新或添加DATABASE_URL
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
    
    // 确保其他必要的环境变量存在
    const requiredVars = {
      'NODE_ENV': 'production'
    };
    
    for (const [key, defaultValue] of Object.entries(requiredVars)) {
      const exists = lines.some(line => line.startsWith(`${key}=`));
      if (!exists) {
        lines.push(`${key}=${defaultValue}`);
      }
    }
    
    // 写回文件
    fs.writeFileSync(this.envPath, lines.filter(line => line.trim()).join('\n') + '\n');
    
    console.log('✅ Environment configuration updated');
  }

  /**
   * 运行数据库迁移
   */
  async runMigrations(databaseUrl) {
    console.log('🗄️  Running database migrations...');
    
    try {
      // 确保DATABASE_URL环境变量设置
      process.env.DATABASE_URL = databaseUrl;
      
      // 查找 Prisma 客户端生成器
      const nodeModulesPath = path.join(this.rootDir, 'node_modules');
      const prismaPath = path.join(nodeModulesPath, '.bin', 'prisma');
      
      if (fs.existsSync(prismaPath)) {
        // 生成Prisma客户端
        execSync(`"${prismaPath}" generate`, { stdio: 'inherit' });
        
        // 运行迁移
        execSync(`"${prismaPath}" migrate deploy`, { stdio: 'inherit' });
      } else {
        // 备用方案：直接使用node模块
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
   * 打印设置指导
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
   * 辅助方法
   */
  sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  /**
   * 验证配置
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

// CLI 接口
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