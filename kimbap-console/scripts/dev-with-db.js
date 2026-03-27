#!/usr/bin/env node
/**
 * Development environment startup script with automatic database management
 * This script ensures the database is properly initialized and migrated before starting the app
 */

const { spawn, execSync } = require('child_process');
const path = require('path');
const fs = require('fs');

// Colors for console output
const colors = {
  reset: '\x1b[0m',
  bright: '\x1b[1m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  red: '\x1b[31m',
  cyan: '\x1b[36m'
};

function log(message, color = colors.reset) {
  console.log(`${color}${message}${colors.reset}`);
}

// Check if a port is available
function isPortAvailable(port) {
  try {
    execSync(`lsof -i:${port}`, { stdio: 'ignore' });
    return false;
  } catch {
    return true;
  }
}

// Find an available port starting from the given port
function findAvailablePort(startPort) {
  let port = startPort;
  while (!isPortAvailable(port)) {
    port++;
  }
  return port;
}

// Save port configuration
function savePortConfig(frontendPort, backendPort) {
  const config = {
    frontend: frontendPort,
    backend: backendPort,
    timestamp: new Date().toISOString()
  };
  fs.writeFileSync('.port-config.json', JSON.stringify(config, null, 2));
}

async function initializeDatabase() {
  log('\n📊 Initializing database...', colors.bright);
  
  try {
    // Run the database initialization script
    execSync('node scripts/unified-db-init.js', { stdio: 'inherit' });
    return true;
  } catch (error) {
    log('❌ Failed to initialize database', colors.red);
    console.error(error);
    return false;
  }
}

async function startDevelopmentEnvironment() {
  log('\n================================', colors.bright);
  log('  Starting Development Environment', colors.bright);
  log('================================\n', colors.bright);
  
  // Initialize database first
  const dbInitialized = await initializeDatabase();
  if (!dbInitialized) {
    log('\n❌ Cannot start without database', colors.red);
    process.exit(1);
  }
  
  // Find available ports
  const frontendPort = findAvailablePort(3000);
  const backendPort = findAvailablePort(3002);
  
  log(`\n🔌 Port Configuration:`, colors.bright);
  log(`   Frontend: ${frontendPort}`, colors.cyan);
  log(`   Backend:  ${backendPort}`, colors.cyan);
  
  // Save port configuration
  savePortConfig(frontendPort, backendPort);
  
  // Set environment variables
  const env = {
    ...process.env,
    PORT: frontendPort,
    FRONTEND_PORT: frontendPort,
    BACKEND_PORT: backendPort,
    NEXT_PUBLIC_BACKEND_PORT: backendPort,
    KIMBAP_CORE_URL: process.env.KIMBAP_CORE_URL || `http://localhost:${backendPort}`
  };
  
  log('\n🚀 Starting services...', colors.bright);
  
  log('\n📦 Starting application service...', colors.yellow);
  const app = spawn('npm', ['run', 'dev:next'], {
    env,
    stdio: 'inherit'
  });
  
  // Handle process termination
  const cleanup = () => {
    log('\n🛑 Shutting down services...', colors.yellow);
    app.kill();
    process.exit(0);
  };
  
  process.on('SIGINT', cleanup);
  process.on('SIGTERM', cleanup);
  
  // Log access URLs after a delay
  setTimeout(() => {
    log('\n✅ Development environment is ready!', colors.green);
    log('\n📱 Access your application at:', colors.bright);
    log(`   Frontend: http://localhost:${frontendPort}`, colors.cyan);
    log(`   Backend:  http://localhost:${backendPort}`, colors.cyan);
    log(`   Database: postgresql://kimbap:kimbap123@localhost:5432/kimbap_db`, colors.cyan);
    log('\n📚 Useful commands:', colors.bright);
    log('   npm run db:studio    - Open Prisma Studio', colors.blue);
    log('   npm run db:migrate   - Create new migration', colors.blue);
    log('   npm run db:push      - Push schema changes (dev only)', colors.blue);
  }, 5000);
}

// Handle errors
process.on('unhandledRejection', (error) => {
  log('\n❌ Unexpected error:', colors.red);
  console.error(error);
  process.exit(1);
});

// Start the application
startDevelopmentEnvironment().catch((error) => {
  log('\n❌ Failed to start development environment:', colors.red);
  console.error(error);
  process.exit(1);
});
