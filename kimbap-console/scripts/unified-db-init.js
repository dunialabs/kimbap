#!/usr/bin/env node

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

function run(command, stdio = 'inherit') {
  execSync(command, { stdio });
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

function ensureSqliteDirectory() {
  const dbPath = resolveSqlitePath();
  const dir = path.dirname(path.resolve(dbPath));
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
  }
}

function initialize() {
  try {
    ensureSqliteDirectory();

    run('npx prisma db push --schema=./prisma/schema.prisma', 'pipe');

    if (process.env.SKIP_PRISMA_GENERATE !== 'true') {
      run('npx prisma generate --schema=./prisma/schema.prisma', process.env.DB_INIT_VERBOSE === 'true' ? 'inherit' : 'pipe');
    }

    process.exit(0);
  } catch (error) {
    console.error('Database initialization failed:', error);
    process.exit(1);
  }
}

process.on('SIGINT', () => process.exit(130));
process.on('SIGTERM', () => process.exit(143));
process.on('unhandledRejection', (error) => {
  console.error(error);
  process.exit(1);
});

initialize();
