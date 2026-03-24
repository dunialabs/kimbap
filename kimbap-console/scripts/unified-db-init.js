#!/usr/bin/env node

/**
 * Unified Database Initialization and Migration Script
 * This script handles both new installations and upgrades using Prisma Migrate
 * 
 * Features:
 * - Automatic detection of new vs existing database
 * - Applies only necessary migrations
 * - Works in all environments (Docker, local, production)
 * - Silent operation for better user experience
 */

const { execSync } = require('child_process');
const path = require('path');
const fs = require('fs');

// Configuration
const MAX_RETRIES = 30;
const RETRY_DELAY = 1000; // 1 second
const SILENT_MODE = process.env.DB_INIT_VERBOSE !== 'true'; // Silent by default

// Colors for console output (only used in verbose mode)
const colors = {
  reset: '\x1b[0m',
  bright: '\x1b[1m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  red: '\x1b[31m',
  gray: '\x1b[90m'
};

/**
 * Log helper that respects silent mode
 */
function log(message, color = colors.reset, forceShow = false) {
  if (!SILENT_MODE || forceShow) {
    console.log(`${color}${message}${colors.reset}`);
  }
}

/**
 * Execute command with error handling
 */
function exec(command, options = {}) {
  const defaultOptions = {
    stdio: SILENT_MODE ? 'pipe' : 'inherit',
    encoding: 'utf8',
    ...options
  };
  
  try {
    log(`Running: ${command}`, colors.gray);
    const result = execSync(command, defaultOptions);
    return result ? result.toString().trim() : '';
  } catch (error) {
    if (options.ignoreError) {
      return null;
    }
    throw error;
  }
}

/**
 * Wait for database to be ready
 */
async function waitForDatabase() {
  log('Waiting for database connection...', colors.yellow);
  
  for (let i = 0; i < MAX_RETRIES; i++) {
    try {
      // Try a simple query to check if database is ready
      exec('npx prisma db execute --stdin --schema=./prisma/schema.prisma', {
        input: 'SELECT 1',
        stdio: 'pipe'
      });
      log('✅ Database is ready', colors.green);
      return true;
    } catch (error) {
      if (i === MAX_RETRIES - 1) {
        log('❌ Database connection timeout', colors.red, true);
        throw new Error('Could not connect to database after ' + MAX_RETRIES + ' attempts');
      }
      // Wait before retry
      await new Promise(resolve => setTimeout(resolve, RETRY_DELAY));
    }
  }
  return false;
}

/**
 * Check if this is a new database installation
 */
function isNewDatabase() {
  try {
    // Check if _prisma_migrations table exists and has records
    const result = exec(
      'npx prisma migrate status --schema=./prisma/schema.prisma',
      { stdio: 'pipe', ignoreError: true }
    );
    
    if (!result) {
      // Command failed, likely new database
      return true;
    }
    
    // Parse the output to determine status
    if (result.includes('No migration found') || 
        result.includes('The database is empty') ||
        result.includes('No schema found')) {
      return true;
    }
    
    return false;
  } catch (error) {
    // Error likely means new database
    return true;
  }
}

/**
 * Apply database migrations
 */
function applyMigrations() {
  const isNew = isNewDatabase();
  
  if (isNew) {
    log('📦 Detected new database installation', colors.blue);
    log('Applying all migrations...', colors.yellow);
  } else {
    log('📦 Detected existing database', colors.blue);
    log('Checking for pending migrations...', colors.yellow);
  }
  
  try {
    // Use migrate deploy for production-safe migration
    // This applies all pending migrations without creating new ones
    const output = exec('npx prisma migrate deploy --schema=./prisma/schema.prisma', {
      stdio: 'pipe'
    });
    
    if (output && output.includes('No pending migrations')) {
      log('✅ Database is already up to date', colors.green);
    } else {
      log('✅ Migrations applied successfully', colors.green);
    }
    
    return true;
  } catch (error) {
    // Check if error is because database is already up to date
    const errorMsg = error.message || error.toString();
    if (errorMsg.includes('No pending migrations') || 
        errorMsg.includes('already in sync')) {
      log('✅ Database is already up to date', colors.green);
      return true;
    }
    
    // Handle P3009 error (failed migrations)
    if (errorMsg.includes('P3009') || errorMsg.includes('failed migrations')) {
      log('⚠️  Found failed migrations, attempting to resolve...', colors.yellow);
      
      try {
        // Try to create the missing table manually
        exec('echo "CREATE TABLE IF NOT EXISTS \\"public\\".\\"ip_whitelist\\" (\\"id\\" SERIAL NOT NULL, \\"ip\\" VARCHAR(128) NOT NULL DEFAULT \'\', \\"addtime\\" INTEGER NOT NULL DEFAULT 0, CONSTRAINT \\"ip_whitelist_pkey\\" PRIMARY KEY (\\"id\\"));" | npx prisma db execute --stdin --schema=./prisma/schema.prisma', {
          stdio: 'pipe'
        });
        
        // Mark the failed migration as applied
        if (errorMsg.includes('20241201140000_add_ip_whitelist_table')) {
          exec('npx prisma migrate resolve --applied "20241201140000_add_ip_whitelist_table"', {
            stdio: 'pipe'
          });
          log('✅ Failed migration resolved and marked as applied', colors.green);
          return true;
        }
      } catch (resolveError) {
        log('❌ Failed to resolve migration automatically', colors.red, true);
      }
    }
    
    log('❌ Migration failed', colors.red, true);
    console.error(error.message);
    throw error;
  }
}

/**
 * Generate Prisma Client
 */
function generatePrismaClient() {
  log('Generating Prisma Client...', colors.yellow);
  try {
    exec('npx prisma generate --schema=./prisma/schema.prisma', {
      stdio: SILENT_MODE ? 'pipe' : 'inherit'
    });
    log('✅ Prisma Client generated', colors.green);
    return true;
  } catch (error) {
    log('❌ Failed to generate Prisma Client', colors.red, true);
    throw error;
  }
}

/**
 * Main initialization function
 */
async function initialize() {
  const startTime = Date.now();
  
  // Show banner only in verbose mode
  if (!SILENT_MODE) {
    log('\n================================', colors.bright);
    log('  Database Migration System     ', colors.bright);
    log('================================\n', colors.bright);
  }
  
  try {
    // Step 1: Wait for database to be ready
    await waitForDatabase();
    
    // Step 2: Apply migrations (handles both new and existing databases)
    applyMigrations();
    
    // Step 3: Generate Prisma Client (skip in Docker as it's already generated)
    if (process.env.SKIP_PRISMA_GENERATE !== 'true') {
      generatePrismaClient();
    } else {
      log('✅ Skipping Prisma Client generation (already exists)', colors.green);
    }
    
    const duration = ((Date.now() - startTime) / 1000).toFixed(2);
    log(`\n✅ Database initialization completed in ${duration}s`, colors.green);
    
    // Exit with success
    process.exit(0);
  } catch (error) {
    const duration = ((Date.now() - startTime) / 1000).toFixed(2);
    log(`\n❌ Database initialization failed after ${duration}s`, colors.red, true);
    console.error(error);
    process.exit(1);
  }
}

/**
 * Handle process signals
 */
process.on('SIGINT', () => {
  log('\n⚠️  Initialization interrupted', colors.yellow, true);
  process.exit(130);
});

process.on('SIGTERM', () => {
  log('\n⚠️  Initialization terminated', colors.yellow, true);
  process.exit(143);
});

// Handle uncaught errors
process.on('unhandledRejection', (error) => {
  log('\n❌ Unexpected error:', colors.red, true);
  console.error(error);
  process.exit(1);
});

// Run the initialization
initialize();