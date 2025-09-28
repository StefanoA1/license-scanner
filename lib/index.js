const { spawn } = require('child_process');
const path = require('path');
const { getBinaryPath } = require('./binary');

class LicenseScanner {
  constructor(options = {}) {
    this.options = options;
  }

  async scan(projectPath = process.cwd()) {
    return new Promise((resolve, reject) => {
      const binaryPath = getBinaryPath();
      const args = [];

      // Add flags first, then project path
      if (this.options.verbose) {
        args.push('-verbose');
      }

      if (this.options.format) {
        args.push('-format', this.options.format);
      }

      if (this.options.prodOnly) {
        args.push('-prod-only');
      }

      if (this.options.noSummary) {
        args.push('-no-summary');
      }

      // Add project path at the end
      args.push(projectPath);


      const child = spawn(binaryPath, args);
      let stdout = '';
      let stderr = '';

      child.stdout.on('data', (data) => {
        stdout += data.toString();
      });

      child.stderr.on('data', (data) => {
        stderr += data.toString();
      });

      child.on('close', (code) => {
        if (code !== 0) {
          reject(new Error(`Scanner failed with code ${code}: ${stderr}`));
          return;
        }

        // For HTML format, return raw output, for JSON format parse it
        if (this.options.format === 'html') {
          resolve(stdout);
        } else {
          try {
            const result = JSON.parse(stdout);
            resolve(result);
          } catch (error) {
            reject(new Error(`Failed to parse scanner output: ${error.message}`));
          }
        }
      });

      child.on('error', (error) => {
        reject(new Error(`Failed to start scanner: ${error.message}`));
      });
    });
  }
}

module.exports = { LicenseScanner };