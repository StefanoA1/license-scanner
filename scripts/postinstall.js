#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { getBinaryPath } = require('../lib/binary');

try {
  const binaryPath = getBinaryPath();

  // Make the binary executable
  fs.chmodSync(binaryPath, 0o755);

  console.log(`Made ${path.basename(binaryPath)} executable`);
} catch (error) {
  console.warn(`Warning: Could not set execute permissions: ${error.message}`);
  // Don't fail the install if we can't set permissions
}