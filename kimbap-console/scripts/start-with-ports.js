#!/usr/bin/env node

const { spawn, spawnSync, exec } = require('child_process');
const { allocatePorts } = require('./port-manager');
const { promisify } = require('util');
const fs = require('fs');
const path = require('path');

// Load environment variables from .env file
const dotenvPath = path.join(process.cwd(), '.env');
if (fs.existsSync(dotenvPath)) {
  require('dotenv').config({ path: dotenvPath });
}

const execAsync = promisify(exec);

async function startWithPortAllocation() {
  console.log('🚀 Starting Kimbap Console with automatic port allocation...\n');
  
  try {
    console.log('');
    
    // 执行统一的数据库初始化（包含迁移和客户端生成）
    console.log('🔄 Initializing database...');
    const dbInitResult = spawnSync('npm', ['run', 'db:init'], {
      stdio: 'inherit',
      cwd: process.cwd()
    });
    
    if (dbInitResult.status !== 0) {
      console.error('❌ Failed to initialize database');
      process.exit(1);
    }
    
    console.log('✅ Database initialized\n');
    
    console.log('🔨 Building backend...');
    const buildResult = spawnSync('npm', ['run', 'build:backend'], {
      stdio: 'inherit',
      cwd: process.cwd()
    });
    
    if (buildResult.status !== 0) {
      console.error('❌ Failed to build backend');
      process.exit(1);
    }
    
    console.log('✅ Backend built successfully\n');
    
    // 分配可用端口
    const { frontendPort, backendPort } = await allocatePorts();
    
    console.log('\n📦 Starting services...\n');
    
    // 设置环境变量
    const env = {
      ...process.env,
      FRONTEND_PORT: frontendPort,
      BACKEND_PORT: backendPort,
      PORT: frontendPort  // Next.js 使用的环境变量
    };

    // 判断是否使用自定义服务器（支持HTTPS）
    const useCustomServer = process.env.ENABLE_HTTPS === 'true';
    const frontendCommand = useCustomServer ? 'npm run dev:next:custom' : 'npm run dev:next';

    // 构建 concurrently 命令
    const concurrentlyArgs = [
      '--kill-others-on-fail',
      '-n', 'DB,Backend,Frontend',
      '-c', 'yellow,green,blue',
      'npm run db:start',
      `BACKEND_PORT=${backendPort} BACKEND_HTTPS_PORT=${backendPort} npm run dev:backend`,
      `PORT=${frontendPort} FRONTEND_PORT=${frontendPort} FRONTEND_HTTPS_PORT=${frontendPort} ${frontendCommand}`
    ];
    
    // 启动服务
    const child = spawn('npx', ['concurrently', ...concurrentlyArgs], {
      stdio: 'inherit',
      env: env,
      cwd: process.cwd()
    });
    
    
    // 处理进程信号
    process.on('SIGINT', async () => {
      console.log('\n🛑 Shutting down services...');
      
      // 停止其他服务
      child.kill('SIGINT');
    });
    
    process.on('SIGTERM', async () => {
      child.kill('SIGTERM');
    });
    
    // 等待子进程结束
    child.on('close', (code) => {
      console.log(`\n✨ Services stopped with code ${code}`);
      process.exit(code);
    });
    
    child.on('error', (error) => {
      console.error('❌ Failed to start services:', error);
      process.exit(1);
    });
    
  } catch (error) {
    console.error('❌ Startup failed:', error.message);
    process.exit(1);
  }
}

// 执行启动
startWithPortAllocation();