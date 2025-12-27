const fs = require('fs');
const path = require('path');
const http = require('http');

/**
 * Global setup for E2E tests
 * Runs once before all tests
 *
 * Now supports two modes:
 * 1. Dedicated test database (test.db) - creates fresh test data
 * 2. Existing server - uses running server with existing data
 */
module.exports = async (config) => {
  console.log('');
  console.log('═══════════════════════════════════════════════════');
  console.log('🚀 Global Setup: Preparing E2E Test Environment');
  console.log('═══════════════════════════════════════════════════');
  console.log('');

  try {
    // Check if server is already running
    const serverRunning = await checkServerHealth('http://localhost:8080');

    if (serverRunning) {
      console.log('   ✅ Server is running at localhost:8080');
      console.log('   ℹ️  Using existing database and test data');
      console.log('');
      console.log('✅ Global setup complete!');
      console.log('═══════════════════════════════════════════════════');
      console.log('');
      return;
    }

    // If server not running, check for test database
    const testDbPath = path.resolve(__dirname, 'test.db');
    const parentDbPath = path.resolve(__dirname, '..', 'gassigeher.db');

    console.log('⏳ Waiting for server to start...');
    let waitCount = 0;
    while (waitCount < 15) {
      if (await checkServerHealth('http://localhost:8080')) {
        console.log('   ✅ Server started successfully');
        break;
      }
      await new Promise(resolve => setTimeout(resolve, 1000));
      waitCount++;
    }

    if (waitCount >= 15) {
      throw new Error('Server did not start after 15 seconds. Please start the server manually.');
    }

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

/**
 * Check if server is running by calling health endpoint
 */
function checkServerHealth(baseUrl) {
  return new Promise((resolve) => {
    const url = new URL('/api/health', baseUrl);
    const req = http.get(url, (res) => {
      resolve(res.statusCode === 200);
    });
    req.on('error', () => resolve(false));
    req.setTimeout(2000, () => {
      req.destroy();
      resolve(false);
    });
  });
}

// DONE: Global setup runs once before all tests
