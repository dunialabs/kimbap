export type Dialect = 'sqlite';

export function detectDialect(): Dialect {
  return 'sqlite';
}

export function getDialect(): Dialect {
  return 'sqlite';
}

export function isPostgres(): boolean {
  return false;
}

export function isSQLite(): boolean {
  return true;
}
