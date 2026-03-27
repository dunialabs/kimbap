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
    
    // （）
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
    
    console.log('🔨 Building application...');
    const buildResult = spawnSync('npm', ['run', 'build'], {
      stdio: 'inherit',
      cwd: process.cwd()
    });
    
    if (buildResult.status !== 0) {
      console.error('❌ Failed to build application');
      process.exit(1);
    }
    
    console.log('✅ Application built successfully\n');
    
    // 
    const { frontendPort, backendPort } = await allocatePorts();
    
    console.log('\n📦 Starting services...\n');
    
    // 
    const env = {
      ...process.env,
      FRONTEND_PORT: frontendPort,
      BACKEND_PORT: backendPort,
      KIMBAP_CORE_URL: process.env.KIMBAP_CORE_URL || 'http://localhost:3002',
      PORT: frontendPort  // Next.js 
    };

    // （HTTPS）
    const useCustomServer = process.env.ENABLE_HTTPS === 'true';
    const frontendCommand = useCustomServer ? 'npm run dev:custom' : 'npm run dev:next';

    //  concurrently 
    const concurrentlyArgs = [
      '--kill-others-on-fail',
      '-n', 'DB,Frontend',
      '-c', 'yellow,blue',
      'npm run db:start',
      `PORT=${frontendPort} FRONTEND_PORT=${frontendPort} FRONTEND_HTTPS_PORT=${frontendPort} ${frontendCommand}`
    ];
    
    // 
    const child = spawn('npx', ['concurrently', ...concurrentlyArgs], {
      stdio: 'inherit',
      env: env,
      cwd: process.cwd()
    });
    
    
    // 
    process.on('SIGINT', async () => {
      console.log('\n🛑 Shutting down services...');
      
      // 
      child.kill('SIGINT');
    });
    
    process.on('SIGTERM', async () => {
      child.kill('SIGTERM');
    });
    
    // 
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

// 
startWithPortAllocation();
