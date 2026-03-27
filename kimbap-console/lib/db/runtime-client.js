const databaseUrl = process.env.DATABASE_URL || '';
const isPostgres = databaseUrl.startsWith('postgresql://') || databaseUrl.startsWith('postgres://');

module.exports = isPostgres
  ? require('@kimbap/prisma-postgres-client')
  : require('@prisma/client');
