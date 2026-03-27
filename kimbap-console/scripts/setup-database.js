#!/usr/bin/env node

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

function run(command) {
  execSync(command, { stdio: 'inherit' });
}

function resolveSqlitePath(databaseUrl = process.env.DATABASE_URL || '') {
  if (!databaseUrl) {
    return './data/kimbap-console.db';
  }

  if (databaseUrl.startsWith('file:')) {
    return databaseUrl.replace(/^file:/, '');
  }

  return databaseUrl;
}

async function setupDatabase() {
  try {
    console.log('🔧 Setting up SQLite database...');

    const dbPath = resolveSqlitePath();
    const dir = path.dirname(path.resolve(dbPath));
    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir, { recursive: true });
    }

    console.log('📦 Syncing schema...');
    run('npx prisma db push --schema=./prisma/schema.prisma --accept-data-loss');

    console.log('✅ SQLite database is ready');
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    console.error('❌ Database setup failed:', message);
    process.exit(1);
  }
}

if (require.main === module) {
  setupDatabase();
}

module.exports = { setupDatabase };
