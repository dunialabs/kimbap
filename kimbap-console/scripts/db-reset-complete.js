#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

function resolveSqlitePath(databaseUrl = process.env.DATABASE_URL || '') {
  if (!databaseUrl) {
    return './data/kimbap-console.db';
  }

  if (databaseUrl.startsWith('file:')) {
    return databaseUrl.replace(/^file:/, '');
  }

  return databaseUrl;
}

function resetDatabase() {
  try {
    const dbPath = path.resolve(resolveSqlitePath());
    const dbDir = path.dirname(dbPath);

    if (!fs.existsSync(dbDir)) {
      fs.mkdirSync(dbDir, { recursive: true });
    }

    if (fs.existsSync(dbPath)) {
      fs.rmSync(dbPath, { force: true });
    }

    execSync('npx prisma db push --schema=./prisma/schema.prisma --accept-data-loss', { stdio: 'inherit' });
    execSync('npx prisma generate --schema=./prisma/schema.prisma', { stdio: 'inherit' });

    console.log('✅ SQLite database reset completed');
    process.exit(0);
  } catch (error) {
    console.error('❌ SQLite database reset failed:', error);
    process.exit(1);
  }
}

resetDatabase();
