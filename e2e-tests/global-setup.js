const fs = require('fs');
const path = require('path');
const { setupAdminAuth } = require('./fixtures/auth');
const { setupTestData } = require('./setup-test-data');

/**
 * Global setup for E2E tests
 * Runs once before all tests
 * Uses setup-test-data.js for test data
 */
module.exports = async (config) => {
  console.log('');
  console.log('═══════════════════════════════════════════════════');
  console.log('🚀 Global Setup: Preparing E2E Test Environment');
  console.log('═══════════════════════════════════════════════════');
  console.log('');

  try {
    const testDbPath = path.resolve(__dirname, 'test.db');

    // Step 2: Wait for server to create database
    console.log('⏳ Waiting for server to create database...');
    let waitCount = 0;
    while (!fs.existsSync(testDbPath) && waitCount < 15) {
      await new Promise(resolve => setTimeout(resolve, 1000));
      waitCount++;
    }

    if (!fs.existsSync(testDbPath)) {
      throw new Error('Server did not create test database after 15 seconds');
    }
    console.log('   ✅ Database created by server');

    // Step 3: Generate test data
    console.log('🌱 Generating test data...');
    console.log('');

    setupTestData(testDbPath)

    // Step 4: Pre-authenticate admin user
    console.log('🔐 Pre-authenticating admin user...');
    await setupAdminAuth();

    console.log('');
    console.log('✅ Global setup complete!');
    console.log('═══════════════════════════════════════════════════');
    console.log('');

  } catch (error) {
    console.error('');
    console.error('❌ Global setup failed:', error.message);
    console.error('═══════════════════════════════════════════════════');
    console.error('');
    throw error;
  }
};

// DONE: Global setup runs once before all tests
