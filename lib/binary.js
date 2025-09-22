const os = require('os');
const path = require('path');
const fs = require('fs');

function getBinaryPath() {
  const platform = os.platform();
  const arch = os.arch();

  // Map Node.js platform/arch to Go platform/arch
  const goPlatform = platform === 'win32' ? 'windows' : platform;
  const goArch = arch === 'x64' ? 'amd64' : arch;

  let binaryName = `license-scanner-${goPlatform}-${goArch}`;
  if (platform === 'win32') {
    binaryName += '.exe';
  }

  // For development, use the simple binary in the bin directory
  const devBinaryPath = path.join(__dirname, '..', 'bin', 'license-scanner' + (platform === 'win32' ? '.exe' : ''));
  if (fs.existsSync(devBinaryPath)) {
    return devBinaryPath;
  }

  // In production, use platform-specific binary names
  const platformBinaryPath = path.join(__dirname, '..', 'bin', binaryName);
  if (fs.existsSync(platformBinaryPath)) {
    return platformBinaryPath;
  }

  throw new Error(`Binary not found for platform ${platform}-${arch} (looking for ${binaryName})`);
}

module.exports = { getBinaryPath };