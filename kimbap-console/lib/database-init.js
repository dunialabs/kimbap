/**
 * Database initialization and migration
 * Ensures database schema is up to date on server startup
 */

const { execSync } = require('child_process');

async function initializeDatabase() {
  // Skip migration in production to avoid issues
  if (process.env.NODE_ENV === 'production') {
    console.log('[Database] Production mode - skipping automatic migrations');
    return true;
  }
  
  try {
    console.log('[Database] Checking database migrations...');
    
    // First, try to apply migrations directly
    // This handles both new databases and existing ones with pending migrations
    try {
      const result = execSync('npx prisma migrate deploy', { 
        encoding: 'utf8',
        stdio: 'pipe',
        env: { ...process.env }
      });
      
      if (result.includes('No pending migrations') || result.includes('All migrations have been successfully applied')) {
        console.log('[Database] ✅ Database schema is up to date');
        return true;
      }
      
      console.log('[Database] ✅ Migrations applied successfully');
      return true;
      
    } catch (deployError) {
      // Check if error is because of drift or other issues
      const errorOutput = deployError.stderr ? deployError.stderr.toString() : deployError.toString();
      
      if (errorOutput.includes('Drift detected')) {
        console.log('[Database] ⚠️ Schema drift detected, but continuing...');
        return true;  // Don't block server startup for drift
      }
      
      if (errorOutput.includes('P3018')) {
        console.log('[Database] ⚠️ Shadow database issue (P3018), but continuing...');
        return true;  // Don't block server startup for shadow DB issues
      }
      
      if (errorOutput.includes('does not exist')) {
        console.error('[Database] ❌ Database connection failed. Please ensure PostgreSQL is running.');
        return false;
      }
      
      if (errorOutput.includes('No pending migrations')) {
        console.log('[Database] ✅ Database schema is up to date');
        return true;
      }
      
      // For other errors, log but don't block startup
      console.log('[Database] ⚠️ Migration warning:', errorOutput.substring(0, 300));
      return true;
    }
    
  } catch (error) {
    console.error('[Database] ⚠️ Migration check error:', error.message ? error.message.substring(0, 200) : 'Unknown error');
    // Don't block server startup for migration errors
    return true;
  }
}

module.exports = { initializeDatabase };