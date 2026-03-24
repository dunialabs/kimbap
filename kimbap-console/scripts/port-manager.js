#!/usr/bin/env node

const net = require('net');
const fs = require('fs');
const path = require('path');

/**
 * 检查端口是否可用
 */
function isPortAvailable(port) {
  return new Promise((resolve) => {
    const server = net.createServer();
    
    server.listen(port, () => {
      server.once('close', () => {
        resolve(true);
      });
      server.close();
    });
    
    server.on('error', () => {
      resolve(false);
    });
  });
}

/**
 * 查找可用端口
 */
async function findAvailablePort(startPort = 3000, maxPort = 4000, excludePorts = []) {
  for (let port = startPort; port <= maxPort; port++) {
    if (!excludePorts.includes(port) && await isPortAvailable(port)) {
      return port;
    }
  }
  throw new Error(`No available ports found between ${startPort}-${maxPort} excluding ${excludePorts.join(', ')}`);
}

/**
 * 为前端和后端分配可用端口
 */
async function allocatePorts() {
  console.log('🔍 Checking for available ports...');
  
  try {
    // 检查前端默认端口 3000
    let frontendPort = 3000;
    if (!(await isPortAvailable(frontendPort))) {
      console.log('⚠️  Port 3000 is in use, finding alternative...');
      frontendPort = await findAvailablePort(3000, 3010);
      console.log(`📱 Frontend will use port ${frontendPort}`);
    } else {
      console.log('✅ Frontend will use default port 3000');
    }
    
    // 为后端查找可用端口（避开前端端口）
    let backendPort = 3002;
    
    if (!(await isPortAvailable(backendPort)) || backendPort === frontendPort) {
      console.log(`⚠️  Port ${backendPort} is in use or conflicts, finding alternative...`);
      backendPort = await findAvailablePort(3002, 3020, [frontendPort]);
      console.log(`🚀 Backend will use port ${backendPort}`);
    } else {
      console.log('✅ Backend will use default port 3002');
    }
    
    // 保存端口信息到临时文件
    const portConfig = {
      frontendPort,
      backendPort,
      timestamp: Date.now()
    };
    
    const configPath = path.join(__dirname, '../.port-config.json');
    fs.writeFileSync(configPath, JSON.stringify(portConfig, null, 2));
    
    // 设置环境变量
    process.env.FRONTEND_PORT = frontendPort;
    process.env.BACKEND_PORT = backendPort;
    
    console.log(`📋 Port allocation complete:`);
    console.log(`   Frontend: http://localhost:${frontendPort}`);
    console.log(`   Backend:  http://localhost:${backendPort}`);
    
    return portConfig;
    
  } catch (error) {
    console.error('❌ Failed to allocate ports:', error.message);
    process.exit(1);
  }
}

/**
 * 获取已分配的端口
 */
function getAllocatedPorts() {
  const configPath = path.join(__dirname, '../.port-config.json');
  
  if (fs.existsSync(configPath)) {
    try {
      const config = JSON.parse(fs.readFileSync(configPath, 'utf8'));
      // 检查配置是否太旧（超过1小时）
      if (Date.now() - config.timestamp < 3600000) {
        return config;
      }
    } catch (error) {
      console.log('⚠️  Invalid port config, will reallocate');
    }
  }
  
  return null;
}

// 如果直接执行此脚本
if (require.main === module) {
  const command = process.argv[2];
  
  if (command === 'allocate') {
    allocatePorts();
  } else if (command === 'get') {
    const ports = getAllocatedPorts();
    if (ports) {
      console.log(JSON.stringify(ports));
    } else {
      console.log('{}');
    }
  } else {
    console.log('Usage: node port-manager.js [allocate|get]');
  }
}

module.exports = {
  isPortAvailable,
  findAvailablePort,
  allocatePorts,
  getAllocatedPorts
};