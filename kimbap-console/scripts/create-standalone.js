#!/usr/bin/env node

/**
 * 创建独立部署包 - 支持跨平台本地部署
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

class StandaloneBuilder {
  constructor() {
    this.rootDir = process.cwd();
    this.platform = process.platform;
    this.arch = process.arch;
    this.outputDir = path.join(this.rootDir, 'dist', `kimbap-console-standalone-${this.platform}-${this.arch}`);
    
    console.log(`🚀 Creating standalone package for ${this.platform}-${this.arch}`);
  }

  async build() {
    try {
      // 1. 准备目录
      await this.prepare();
      
      // 2. 下载Node.js运行时
      await this.setupNodeRuntime();
      
      // 3. 复制应用文件
      await this.copyAppFiles();
      
      // 4. 创建启动脚本
      await this.createStartupScripts();
      
      // 5. 创建配置文件
      await this.createConfig();
      
      // 6. 创建说明文档
      await this.createDocs();
      
      console.log(`✅ Standalone package created successfully!`);
      console.log(`📁 Location: ${this.outputDir}`);
      console.log(`📊 Size: ${this.getDirectorySize(this.outputDir)}MB`);
      
    } catch (error) {
      console.error('❌ Build failed:', error.message);
      process.exit(1);
    }
  }

  async prepare() {
    console.log('🔧 Preparing directories...');
    
    if (fs.existsSync(this.outputDir)) {
      fs.rmSync(this.outputDir, { recursive: true, force: true });
    }
    
    fs.mkdirSync(this.outputDir, { recursive: true });
    fs.mkdirSync(path.join(this.outputDir, 'app'), { recursive: true });
    fs.mkdirSync(path.join(this.outputDir, 'scripts'), { recursive: true });
    fs.mkdirSync(path.join(this.outputDir, 'config'), { recursive: true });
  }

  async setupNodeRuntime() {
    console.log('📥 Setting up Node.js runtime...');
    
    // 检查是否已经下载了Node.js
    const nodeUrl = this.getNodeDownloadUrl();
    const fileName = path.basename(nodeUrl);
    const downloadPath = path.join(this.rootDir, 'temp', fileName);
    
    if (fs.existsSync(downloadPath)) {
      console.log('✅ Using cached Node.js runtime');
    } else {
      console.log('📥 Downloading Node.js runtime...');
      fs.mkdirSync(path.dirname(downloadPath), { recursive: true });
      await this.downloadFile(nodeUrl, downloadPath);
    }
    
    // 解压到输出目录
    const nodeDir = path.join(this.outputDir, 'node');
    await this.extractNode(downloadPath, nodeDir);
    
    console.log('✅ Node.js runtime ready');
  }

  getNodeDownloadUrl() {
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
    
    return `${baseUrl}/v${version}/${fileName}`;
  }

  async downloadFile(url, destination) {
    return new Promise((resolve, reject) => {
      const https = require('https');
      const file = fs.createWriteStream(destination);
      
      https.get(url, (response) => {
        if (response.statusCode === 200) {
          response.pipe(file);
          file.on('finish', () => {
            file.close();
            resolve();
          });
        } else if (response.statusCode === 302 || response.statusCode === 301) {
          file.close();
          fs.unlinkSync(destination);
          this.downloadFile(response.headers.location, destination)
            .then(resolve)
            .catch(reject);
        } else {
          file.close();
          fs.unlinkSync(destination);
          reject(new Error(`HTTP ${response.statusCode}`));
        }
      }).on('error', (err) => {
        file.close();
        if (fs.existsSync(destination)) fs.unlinkSync(destination);
        reject(err);
      });
    });
  }

  async extractNode(archivePath, nodeDir) {
    console.log('📦 Extracting Node.js...');
    
    const tempExtractDir = path.join(path.dirname(archivePath), 'node-extract');
    fs.mkdirSync(tempExtractDir, { recursive: true });
    
    try {
      if (this.platform === 'win32') {
        // 使用PowerShell解压ZIP
        const extractCmd = `powershell -command "Expand-Archive -Path '${archivePath}' -DestinationPath '${tempExtractDir}' -Force"`;
        execSync(extractCmd, { stdio: 'pipe' });
      } else {
        // 使用tar解压
        const extractCmd = `tar -xf "${archivePath}" -C "${tempExtractDir}"`;
        execSync(extractCmd, { stdio: 'pipe' });
      }
      
      // 找到解压后的Node.js目录
      const extractedDirs = fs.readdirSync(tempExtractDir);
      const nodeSourceDir = path.join(tempExtractDir, extractedDirs[0]);
      
      // 复制到最终位置
      this.copyDir(nodeSourceDir, nodeDir);
      
      // 清理临时目录
      fs.rmSync(tempExtractDir, { recursive: true, force: true });
      
    } catch (error) {
      throw new Error(`Node.js extraction failed: ${error.message}`);
    }
  }

  async copyAppFiles() {
    console.log('📋 Copying application files...');
    
    const appDir = path.join(this.outputDir, 'app');
    
    // 检查构建文件是否存在
    const nextDir = path.join(this.rootDir, '.next');
    if (!fs.existsSync(nextDir)) {
      throw new Error('Next.js build not found. Please run "npm run build" first.');
    }
    
    // 复制核心文件
    const standaloneNextDir = path.join(this.rootDir, '.next/standalone/.next');
    if (fs.existsSync(standaloneNextDir)) {
      this.copyDir(standaloneNextDir, path.join(appDir, '.next'));
    } else {
      this.copyDir(nextDir, path.join(appDir, '.next'));
    }
    this.copyDir(path.join(this.rootDir, 'public'), path.join(appDir, 'public'));
    this.copyDir(path.join(this.rootDir, 'prisma'), path.join(appDir, 'prisma'));
    
    // 复制后端服务器（已编译的proxy-server）
    const proxyServerDir = path.join(this.rootDir, 'proxy-server');
    if (fs.existsSync(proxyServerDir)) {
      this.copyDir(proxyServerDir, path.join(appDir, 'proxy-server'));
      // 创建proxy-server的package.json以支持ES模块
      fs.writeFileSync(path.join(appDir, 'proxy-server/package.json'), JSON.stringify({ type: 'module' }, null, 2));
      console.log('✅ Backend proxy-server copied');
    } else {
      console.warn('⚠️  proxy-server not found, make sure to run "npm run build:backend" first');
    }
    
    // 复制脚本和配置
    fs.mkdirSync(path.join(appDir, 'scripts'), { recursive: true });
    fs.copyFileSync(path.join(this.rootDir, 'scripts/database-config.js'), path.join(appDir, 'scripts/database-config.js'));
    
    // 创建简化的package.json（仅生产依赖）
    const originalPackage = JSON.parse(fs.readFileSync(path.join(this.rootDir, 'package.json'), 'utf8'));
    const prodPackage = {
      name: originalPackage.name,
      version: originalPackage.version,
      private: true,
      scripts: {
        start: "node node_modules/next/dist/bin/next start -p 3000",
        "db:migrate": "prisma migrate deploy",
        "db:generate": "prisma generate"
      },
      dependencies: originalPackage.dependencies
    };
    
    fs.writeFileSync(path.join(appDir, 'package.json'), JSON.stringify(prodPackage, null, 2));
    fs.copyFileSync(path.join(this.rootDir, 'next.config.mjs'), path.join(appDir, 'next.config.mjs'));
    
    // 复制node_modules（从.next/standalone或主目录）
    console.log('📦 Copying dependencies...');
    const standaloneNodeModules = path.join(this.rootDir, '.next/standalone/node_modules');
    if (fs.existsSync(standaloneNodeModules)) {
      this.copyDir(standaloneNodeModules, path.join(appDir, 'node_modules'));
    } else {
      // 复制必要的生产依赖
      this.copyDir(path.join(this.rootDir, 'node_modules'), path.join(appDir, 'node_modules'));
    }
    
    // 修复proxy-server的prisma导入路径
    const prismaConfigPath = path.join(appDir, 'proxy-server/config/prisma.js');
    if (fs.existsSync(prismaConfigPath)) {
      let content = fs.readFileSync(prismaConfigPath, 'utf8');
      content = content.replace('../../../node_modules/@prisma/client/index.js', '../../node_modules/@prisma/client/index.js');
      fs.writeFileSync(prismaConfigPath, content);
    }
    
    // 创建环境配置
    const envConfig = `NODE_ENV=production
# 数据库URL将在启动时自动检测
DATABASE_URL=postgresql://kimbap:kimbap123@localhost:5432/kimbap_db

# 云端数据库配置（可选，如果需要连接云数据库请取消注释并填写）
# CLOUD_DB_HOST=your-cloud-host.com
# CLOUD_DB_USER=your-username
# CLOUD_DB_PASSWORD=your-password
# CLOUD_DB_NAME=kimbap_db
# CLOUD_DB_PORT=5432`;
    
    fs.writeFileSync(path.join(appDir, '.env.local'), envConfig);
    
    process.chdir(this.rootDir);
    console.log('✅ Application files ready');
  }

  async createStartupScripts() {
    console.log('📝 Creating startup scripts...');
    
    const scriptsDir = path.join(this.outputDir, 'scripts');
    
    // 复制便携版启动脚本
    fs.copyFileSync(
      path.join(this.rootDir, 'scripts/start-portable.sh'),
      path.join(scriptsDir, 'start.sh')
    );
    fs.copyFileSync(
      path.join(this.rootDir, 'scripts/start-portable.bat'),
      path.join(scriptsDir, 'start.bat')
    );
    
    // 设置执行权限
    if (this.platform !== 'win32') {
      fs.chmodSync(path.join(scriptsDir, 'start.sh'), 0o755);
    }
    
    console.log('✅ Startup scripts created');
  }

  async createConfig() {
    console.log('📄 Creating configuration files...');
    
    const config = {
      app: {
        name: 'Kimbap Console',
        version: '1.0.0',
        port: 3000,
        host: 'localhost'
      },
      database: {
        autoDetect: true,
        fallback: {
          host: 'localhost',
          port: 5432,
          database: 'kimbap_db',
          username: 'kimbap',
          password: 'kimbap123'
        }
      },
      features: {
        autoMigrate: true,
        autoStart: true
      }
    };
    
    fs.writeFileSync(
      path.join(this.outputDir, 'config', 'app.json'),
      JSON.stringify(config, null, 2)
    );
    
    console.log('✅ Configuration files created');
  }

  async createDocs() {
    const readme = `# Kimbap Console 独立部署包

## 快速开始

### Windows
1. 双击 \`scripts/start.bat\` 启动
2. 在浏览器中打开 http://localhost:3000

### Mac/Linux  
1. 打开终端，进入应用目录
2. 运行 \`./scripts/start.sh\`
3. 在浏览器中打开 http://localhost:3000

## 数据库配置

此版本支持多种数据库配置：

### 🐳 Docker PostgreSQL (推荐)
\`\`\`bash
docker run --name kimbap-postgres \\
  -e POSTGRES_USER=kimbap \\
  -e POSTGRES_PASSWORD=kimbap123 \\
  -e POSTGRES_DB=kimbap_db \\
  -p 5432:5432 -d postgres:16
\`\`\`

### 🏠 本地 PostgreSQL
- macOS: \`brew install postgresql@16\`
- Linux: \`apt install postgresql-16\`
- Windows: 从官网下载安装

### ☁️ 云端数据库
编辑 \`app/.env.local\` 文件，设置 CLOUD_DB_* 变量

## 系统要求

- **内存**: 最少 1GB RAM
- **存储**: 最少 200MB 可用空间
- **网络**: 需要联网下载依赖（首次运行）

## 故障排除

### 端口占用
修改 \`config/app.json\` 中的端口设置

### 数据库连接
查看 \`logs/database.log\` 获取详细错误信息

### 权限问题
确保脚本有执行权限：\`chmod +x scripts/start.sh\`

---
Kimbap Console v1.0.0
构建时间: ${new Date().toISOString()}
平台: ${this.platform}-${this.arch}`;

    fs.writeFileSync(path.join(this.outputDir, 'README.md'), readme);
    
    console.log('✅ Documentation created');
  }

  // 辅助方法
  copyDir(src, dest) {
    if (!fs.existsSync(src)) {
      console.warn(`⚠️  Source not found: ${src}`);
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
      try {
        const entries = fs.readdirSync(currentPath, { withFileTypes: true });
        
        for (const entry of entries) {
          const fullPath = path.join(currentPath, entry.name);
          
          if (entry.isDirectory()) {
            walk(fullPath);
          } else {
            size += fs.statSync(fullPath).size;
          }
        }
      } catch (error) {
        // 忽略权限错误
      }
    };
    
    walk(dir);
    return Math.round(size / 1024 / 1024);
  }
}

if (require.main === module) {
  const builder = new StandaloneBuilder();
  builder.build();
}

module.exports = StandaloneBuilder;