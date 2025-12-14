// Setup file for Jest with jsdom
// Polyfill TextEncoder/TextDecoder for older Node versions
const { TextEncoder, TextDecoder } = require('util');
global.TextEncoder = TextEncoder;
global.TextDecoder = TextDecoder;
