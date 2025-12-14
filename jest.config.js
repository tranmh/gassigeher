/** @type {import('jest').Config} */
module.exports = {
  testEnvironment: 'jsdom',
  testMatch: ['**/frontend-tests/**/*.test.js'],
  moduleFileExtensions: ['js'],
  verbose: true,
  testPathIgnorePatterns: ['/node_modules/', '/e2e-tests/'],
  setupFiles: ['./frontend-tests/setup.js'],
};
