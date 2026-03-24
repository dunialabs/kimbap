#!/usr/bin/env node

const net = require('net');
const fs = require('fs');
const path = require('path');

/**
 * Check if a port is available
 * @param {number} port - Port number to check
 * @returns {Promise<boolean>} - True if port is available
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
 * Find the next available port starting from a base port
 * @param {number} basePort - Starting port number
 * @param {number} maxTries - Maximum number of ports to try
 * @returns {Promise<number>} - Next available port
 */
async function findNextAvailablePort(basePort, maxTries = 100) {
  for (let i = 0; i < maxTries; i++) {
    const port = basePort + i;
    if (await isPortAvailable(port)) {
      return port;
    }
  }
  throw new Error(`No available port found starting from ${basePort}`);
}

/**
 * Find available ports for all Kimbap Console services
 * @returns {Promise<Object>} - Object containing available ports
 */
async function findAvailablePorts() {
  const basePorts = {
    frontend: 3000,
    backend: 3002,
    postgres: 5432,
    adminer: 8080
  };

  const availablePorts = {};
  
  console.log('🔍 Checking for available ports...');
  
  for (const [service, basePort] of Object.entries(basePorts)) {
    try {
      const availablePort = await findNextAvailablePort(basePort);
      availablePorts[service] = availablePort;
      
      if (availablePort === basePort) {
        console.log(`✅ ${service}: ${availablePort} (default)`);
      } else {
        console.log(`🔄 ${service}: ${availablePort} (auto-incremented from ${basePort})`);
      }
    } catch (error) {
      console.error(`❌ ${service}: Failed to find available port starting from ${basePort}`);
      availablePorts[service] = basePort; // fallback to base port
    }
  }
  
  return availablePorts;
}

/**
 * Create environment file with port configuration
 * @param {Object} ports - Port configuration object
 */
function createPortConfig(ports) {
  const configPath = path.join(__dirname, '..', '.port-config.json');
  const envPath = path.join(__dirname, '..', '.env.ports');
  
  // Write JSON config for Docker Compose
  fs.writeFileSync(configPath, JSON.stringify(ports, null, 2));
  
  // Write environment variables file
  const envContent = `# Auto-generated port configuration
# Do not edit manually - run npm run docker:find-ports to regenerate

FRONTEND_PORT=${ports.frontend}
BACKEND_PORT=${ports.backend}
POSTGRES_PORT=${ports.postgres}
ADMINER_PORT=${ports.adminer}

# For Docker Compose
KIMBAP_FRONTEND_PORT=${ports.frontend}
KIMBAP_BACKEND_PORT=${ports.backend}
KIMBAP_POSTGRES_PORT=${ports.postgres}
KIMBAP_ADMINER_PORT=${ports.adminer}
`;
  
  fs.writeFileSync(envPath, envContent);
  
  console.log('');
  console.log('📝 Port configuration saved:');
  console.log(`   Config file: ${configPath}`);
  console.log(`   Env file: ${envPath}`);
  console.log('');
  console.log('🚀 Service URLs:');
  console.log(`   Frontend: http://localhost:${ports.frontend}`);
  console.log(`   Backend:  http://localhost:${ports.backend}`);
  console.log(`   Adminer:  http://localhost:${ports.adminer}`);
  console.log('');
}

/**
 * Main function
 */
async function main() {
  try {
    console.log('🐳 Kimbap Console - Port Allocation Tool');
    console.log('=====================================');
    console.log('');
    
    const ports = await findAvailablePorts();
    createPortConfig(ports);
    
    console.log('✅ Port allocation completed successfully!');
    console.log('');
    console.log('💡 Next steps:');
    console.log('   1. Run: docker compose --env-file .env.ports up -d');
    console.log('   2. Or use: npm run docker:up:auto-ports');
    
  } catch (error) {
    console.error('❌ Error finding available ports:', error.message);
    process.exit(1);
  }
}

// Run if called directly
if (require.main === module) {
  main();
}

module.exports = {
  isPortAvailable,
  findNextAvailablePort,
  findAvailablePorts,
  createPortConfig
};