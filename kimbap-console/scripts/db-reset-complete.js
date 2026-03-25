#!/usr/bin/env node

/**
 * Complete Database Reset Script
 * This script performs a complete database reset by:
 * 1. Dropping and recreating the database (if using Docker)
 * 2. Or dropping all tables and re-running migrations
 * 
 * WARNING: This will delete ALL data in the database!
 */

const { execSync } = require('child_process');
const { PrismaClient } = require('@prisma/client');

const prisma = new PrismaClient();

/**
 * Parse DATABASE_URL to extract connection info
 */
function parseDatabaseUrl() {
  const dbUrl = process.env.DATABASE_URL;
  if (!dbUrl) {
    throw new Error('DATABASE_URL environment variable is not set');
  }

  let parsed;
  try {
    parsed = new URL(dbUrl);
  } catch {
    throw new Error('Invalid DATABASE_URL format');
  }

  if (!['postgresql:', 'postgres:'].includes(parsed.protocol)) {
    throw new Error('Invalid DATABASE_URL protocol');
  }

  const database = parsed.pathname.replace(/^\/+/, '');
  if (!database) {
    throw new Error('Invalid DATABASE_URL database name');
  }

  return {
    user: decodeURIComponent(parsed.username),
    password: decodeURIComponent(parsed.password),
    host: parsed.hostname,
    port: parsed.port || '5432',
    database
  };
}

/**
 * Check if we're using Docker and get container name
 */
function getDockerContainer() {
  try {
    const dbInfo = parseDatabaseUrl();
    
    // Check for common container names
    const containerNames = ['kimbap-postgres', 'postgres', 'postgres-console'];
    
    for (const name of containerNames) {
      try {
        const result = execSync(`docker ps --filter name=${name} --format "{{.Names}}"`, { 
          stdio: 'pipe',
          encoding: 'utf8'
        }).trim();
        if (result) {
          return result;
        }
      } catch {
        continue;
      }
    }
    
    // If host is Docker service name, try to find container
    if (dbInfo.host === 'postgres' || dbInfo.host === 'postgres-console') {
      return dbInfo.host;
    }
    
    return null;
  } catch {
    return null;
  }
}

/**
 * Reset database using Docker (drop and recreate)
 */
async function resetUsingDocker(containerName) {
  console.log('🐳 Detected Docker environment');
  console.log(`Using container: ${containerName}`);
  console.log('Dropping and recreating database...');
  
  const dbInfo = parseDatabaseUrl();
  
  try {
    // Drop database using docker exec
    execSync(
      `docker exec ${containerName} psql -U ${dbInfo.user} -d postgres -c "DROP DATABASE IF EXISTS ${dbInfo.database};"`,
      {
        stdio: 'inherit',
        env: { ...process.env, PGPASSWORD: dbInfo.password }
      }
    );
    
    console.log('✅ Database dropped');
    
    // Recreate database using docker exec
    execSync(
      `docker exec ${containerName} psql -U ${dbInfo.user} -d postgres -c "CREATE DATABASE ${dbInfo.database};"`,
      {
        stdio: 'inherit',
        env: { ...process.env, PGPASSWORD: dbInfo.password }
      }
    );
    
    console.log('✅ Database recreated');
    
    return true;
  } catch (error) {
    console.error('❌ Failed to reset database using Docker:', error.message);
    // Fallback to table dropping method
    console.log('⚠️  Falling back to table dropping method...');
    return false;
  }
}

/**
 * Reset database by dropping all tables
 */
async function resetByDroppingTables(prismaClient = prisma) {
  console.log('🗑️  Dropping all tables...');
  
  try {
    // Get all table names
    const tables = await prismaClient.$queryRaw`
      SELECT tablename 
      FROM pg_tables 
      WHERE schemaname = 'public'
    `;
    
    if (tables.length === 0) {
      console.log('✅ No tables to drop');
      return true;
    }
    
    // Drop all tables one by one to avoid issues
    for (const table of tables) {
      try {
        await prismaClient.$executeRawUnsafe(`DROP TABLE IF EXISTS "${table.tablename}" CASCADE;`);
      } catch (error) {
        console.warn(`⚠️  Failed to drop table ${table.tablename}:`, error.message);
      }
    }
    
    console.log(`✅ Dropped ${tables.length} tables`);
    return true;
  } catch (error) {
    console.error('❌ Failed to drop tables:', error.message);
    return false;
  }
}

/**
 * Run Prisma migrations
 */
function runMigrations() {
  console.log('📦 Running migrations...');
  
  try {
    execSync('npx prisma migrate deploy', {
      stdio: 'inherit',
      env: process.env
    });
    
    console.log('✅ Migrations applied');
    return true;
  } catch (error) {
    console.error('❌ Failed to run migrations:', error.message);
    return false;
  }
}

/**
 * Generate Prisma Client
 */
function generateClient() {
  console.log('🔧 Generating Prisma Client...');
  
  try {
    execSync('npx prisma generate', {
      stdio: 'inherit',
      env: process.env
    });
    
    console.log('✅ Prisma Client generated');
    return true;
  } catch (error) {
    console.error('❌ Failed to generate Prisma Client:', error.message);
    return false;
  }
}

/**
 * Main reset function
 */
async function main() {
  console.log('\n================================');
  console.log('  Complete Database Reset');
  console.log('================================\n');
  console.log('⚠️  WARNING: This will delete ALL data!\n');
  
  try {
    // Check if using Docker
    const containerName = getDockerContainer();
    
    let resetSuccess = false;
    if (containerName) {
      resetSuccess = await resetUsingDocker(containerName);
      // If Docker method failed, fall back to table dropping
      if (!resetSuccess) {
        resetSuccess = await resetByDroppingTables();
      }
    } else {
      resetSuccess = await resetByDroppingTables();
    }
    
    if (!resetSuccess) {
      console.error('\n❌ Database reset failed');
      process.exit(1);
    }
    
    // Disconnect Prisma after reset (will reconnect for migrations)
    await prisma.$disconnect();
    
    // Run migrations
    if (!runMigrations()) {
      console.error('\n❌ Migration failed');
      process.exit(1);
    }
    
    // Generate Prisma Client
    if (!generateClient()) {
      console.warn('\n⚠️  Failed to generate Prisma Client (may already be generated)');
    }
    
    console.log('\n✅ Database reset completed successfully!');
    process.exit(0);
  } catch (error) {
    console.error('\n❌ Unexpected error:', error);
    process.exit(1);
  } finally {
    try {
      await prisma.$disconnect();
    } catch {
      // Ignore disconnect errors
    }
  }
}

// Run the reset
main();
