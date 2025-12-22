// Setup file for Jest with jsdom
// Polyfill TextEncoder/TextDecoder for older Node versions
const { TextEncoder, TextDecoder } = require('util');
global.TextEncoder = TextEncoder;
global.TextDecoder = TextDecoder;

// Load source files for testing
// This avoids duplicating class definitions in test files
const fs = require('fs');
const path = require('path');

/**
 * Load a source file and make its exports available globally
 * @param {string} relativePath - Path relative to project root
 */
global.loadSourceFile = function(relativePath) {
    const fullPath = path.join(__dirname, '..', relativePath);
    const code = fs.readFileSync(fullPath, 'utf-8');
    // Execute the code in global context
    eval(code);
};
