/** @type {import('jest').Config} */
module.exports = {
  preset: 'ts-jest',
  testEnvironment: 'node',
  roots: ['<rootDir>/app', '<rootDir>/components', '<rootDir>/lib'],
  modulePathIgnorePatterns: ['<rootDir>/.next/'],
  testPathIgnorePatterns: [
    '/node_modules/',
    '/.next/',
    '/app/api/external/__tests__/api.test.ts',
  ],
};
