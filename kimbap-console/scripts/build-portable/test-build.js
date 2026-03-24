#!/usr/bin/env node

/**
 * 便携包构建测试脚本
 * 执行基本的构建测试，无需实际下载大文件
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

class BuildTester {
  constructor() {
    this.rootDir = path.resolve(__dirname, '../..');
    this.platform = process.platform;
    this.outputDir = path.join(this.rootDir, 'dist', `kimbap-console-${this.platform}-test`);
  }

  async test() {
    console.log('🧪 Testing portable build process...');
    console.log(`Platform: ${this.platform}`);
    
    try {
      // 1. 测试目录创建
      await this.testDirectoryCreation();
      
      // 2. 测试应用构建
      await this.testAppBuild();
      
      // 3. 测试启动脚本生成
      await this.testStartupScripts();
      
      // 4. 测试配置文件生成
      await this.testConfigGeneration();
      
      // 5. 清理测试文件
      await this.cleanup();
      
      console.log('✅ All tests passed!');
      console.log('🚀 Ready to run full build with: npm run build:portable');
      
    } catch (error) {
      console.error('❌ Test failed:', error);
      await this.cleanup();
      process.exit(1);
    }
  }

  async testDirectoryCreation() {
    console.log('🔧 Testing directory creation...');
    
    // 创建输出目录
    if (fs.existsSync(this.outputDir)) {
      fs.rmSync(this.outputDir, { recursive: true, force: true });
    }
    
    const dirs = [
      'app',
      'node',
      'postgresql',
      'scripts',
      'config',
      'logs'
    ];
    
    dirs.forEach(dir => {
      const dirPath = path.join(this.outputDir, dir);
      fs.mkdirSync(dirPath, { recursive: true });
      
      if (!fs.existsSync(dirPath)) {
        throw new Error(`Failed to create directory: ${dir}`);
      }
    });
    
    console.log('✅ Directory creation test passed');
  }

  async testAppBuild() {
    console.log('🏗️  Testing app build...');
    
    try {
      process.chdir(this.rootDir);
      
      // 检查 package.json
      if (!fs.existsSync('package.json')) {
        throw new Error('package.json not found');
      }
      
      // 检查 Next.js 配置
      if (!fs.existsSync('next.config.mjs')) {
        throw new Error('next.config.mjs not found');
      }
      
      // 模拟文件复制（不实际复制大文件）
      const appDir = path.join(this.outputDir, 'app');
      const testFiles = [
        'package.json',
        'next.config.mjs'
      ];
      
      testFiles.forEach(file => {
        const src = path.join(this.rootDir, file);
        const dest = path.join(appDir, file);
        
        if (fs.existsSync(src)) {
          fs.copyFileSync(src, dest);
        }
      });
      
      // 创建环境变量文件
      const prodEnv = `NODE_ENV=production
DATABASE_URL=postgresql://kimbap:kimbap123@localhost:5432/kimbap_db`;
      
      fs.writeFileSync(path.join(appDir, '.env.local'), prodEnv);
      
      console.log('✅ App build test passed');
      
    } catch (error) {
      throw new Error(`App build test failed: ${error.message}`);
    }
  }

  async testStartupScripts() {
    console.log('📝 Testing startup script generation...');
    
    const scriptsDir = path.join(this.outputDir, 'scripts');
    
    if (this.platform === 'win32') {
      await this.createTestWindowsScript(scriptsDir);
    } else {
      await this.createTestUnixScript(scriptsDir);
    }
    
    console.log('✅ Startup script test passed');
  }

  async createTestWindowsScript(scriptsDir) {
    const script = `@echo off
echo Kimbap Console Test Script
echo Platform: Windows
echo.
echo This is a test version. Run 'npm run build:portable' for full build.
pause`;

    const scriptPath = path.join(scriptsDir, 'start-test.bat');
    fs.writeFileSync(scriptPath, script);
    
    if (!fs.existsSync(scriptPath)) {
      throw new Error('Failed to create Windows test script');
    }
  }

  async createTestUnixScript(scriptsDir) {
    const script = `#!/bin/bash
echo "Kimbap Console Test Script"
echo "Platform: ${this.platform}"
echo
echo "This is a test version. Run 'npm run build:portable' for full build."
read -p "Press Enter to continue..."`;

    const scriptPath = path.join(scriptsDir, 'start-test.sh');
    fs.writeFileSync(scriptPath, script);
    fs.chmodSync(scriptPath, 0o755);
    
    if (!fs.existsSync(scriptPath)) {
      throw new Error('Failed to create Unix test script');
    }
  }

  async testConfigGeneration() {
    console.log('📄 Testing config generation...');
    
    const config = {
      app: {
        name: 'Kimbap Console',
        version: '1.0.0-test',
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
      test: true
    };
    
    const configPath = path.join(this.outputDir, 'config', 'config.json');
    fs.writeFileSync(configPath, JSON.stringify(config, null, 2));
    
    if (!fs.existsSync(configPath)) {
      throw new Error('Failed to create config file');
    }
    
    // 验证配置文件内容
    const savedConfig = JSON.parse(fs.readFileSync(configPath, 'utf8'));
    if (savedConfig.app.name !== 'Kimbap Console') {
      throw new Error('Config file content validation failed');
    }
    
    console.log('✅ Config generation test passed');
  }

  async cleanup() {
    console.log('🧹 Cleaning up test files...');
    
    if (fs.existsSync(this.outputDir)) {
      fs.rmSync(this.outputDir, { recursive: true, force: true });
    }
    
    console.log('✅ Cleanup completed');
  }

  // 静态方法：检查系统要求
  static checkSystemRequirements() {
    console.log('🔍 Checking system requirements...');
    
    const requirements = {
      node: process.version,
      platform: process.platform,
      arch: process.arch,
      memory: Math.round(require('os').totalmem() / 1024 / 1024 / 1024) + 'GB'
    };
    
    console.log('System Info:');
    console.log(`- Node.js: ${requirements.node}`);
    console.log(`- Platform: ${requirements.platform}`);
    console.log(`- Architecture: ${requirements.arch}`);
    console.log(`- Memory: ${requirements.memory}`);
    
    // 检查 Node.js 版本
    const nodeVersion = parseInt(process.version.replace('v', '').split('.')[0]);
    if (nodeVersion < 18) {
      throw new Error('Node.js 18+ is required');
    }
    
    console.log('✅ System requirements check passed');
  }

  // 静态方法：检查依赖
  static checkDependencies() {
    console.log('📋 Checking dependencies...');
    
    const requiredFiles = [
      'package.json',
      'next.config.mjs',
      'prisma/schema.prisma'
    ];
    
    const rootDir = path.resolve(__dirname, '../..');
    
    requiredFiles.forEach(file => {
      const filePath = path.join(rootDir, file);
      if (!fs.existsSync(filePath)) {
        throw new Error(`Required file not found: ${file}`);
      }
    });
    
    console.log('✅ Dependencies check passed');
  }
}

// CLI 执行
if (require.main === module) {
  const args = process.argv.slice(2);
  
  if (args.includes('--check')) {
    try {
      BuildTester.checkSystemRequirements();
      BuildTester.checkDependencies();
      console.log('🎉 Ready for portable build!');
    } catch (error) {
      console.error('❌ Requirements check failed:', error.message);
      process.exit(1);
    }
  } else {
    const tester = new BuildTester();
    tester.test();
  }
}

module.exports = BuildTester;