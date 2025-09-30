#!/usr/bin/env node

const { LicenseScanner } = require('./index');
const fs = require('fs');
const path = require('path');

function parseArgs() {
  const args = process.argv.slice(2);
  const options = {
    prodOnly: false,
    format: 'json',
    output: null,
    noSummary: false,
    clearCache: false,
    verbose: false
  };
  let projectPath = process.cwd();

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];

    switch (arg) {
      case '--prod-only':
        options.prodOnly = true;
        break;
      case '--format':
        options.format = args[++i];
        break;
      case '--output':
        options.output = args[++i];
        break;
      case '--no-summary':
        options.noSummary = true;
        break;
      case '--clear-cache':
        options.clearCache = true;
        break;
      case '--verbose':
      case '-v':
        options.verbose = true;
        break;
      case '--help':
      case '-h':
        showHelp();
        process.exit(0);
        break;
      default:
        if (!arg.startsWith('-')) {
          projectPath = path.resolve(arg);
        }
        break;
    }
  }

  return { options, projectPath };
}

function showHelp() {
  console.log(`
License Scanner

Usage: license-scanner [options] [path]

Options:
  --prod-only          Scan production dependencies only
  --format <format>    Output format (json, html) [default: json]
  --output <file>      Output file path
  --no-summary         Skip license summary
  --clear-cache        Clear the license cache
  -v, --verbose        Enable verbose logging
  -h, --help          Show this help message

Examples:
  license-scanner                           # Scan current directory
  license-scanner /path/to/project          # Scan specific directory
  license-scanner --prod-only               # Production dependencies only
  license-scanner --format html --output report.html  # Generate HTML report
`);
}

async function main() {
  try {
    const { options, projectPath } = parseArgs();

    if (options.clearCache) {
      console.log('Cache clearing not implemented yet');
      return;
    }

    const scanner = new LicenseScanner(options);
    const result = await scanner.scan(projectPath);

    if (options.format === 'html') {
      if (options.output) {
        // For HTML format, the Go binary outputs HTML directly
        fs.writeFileSync(options.output, result);
        console.log(`HTML report written to ${options.output}`);
      } else {
        // Output HTML to stdout
        console.log(result);
      }
    } else {
      // JSON format
      if (options.output) {
        fs.writeFileSync(options.output, JSON.stringify(result, null, 2));
        console.log(`Results written to ${options.output}`);
      } else {
        console.log(JSON.stringify(result, null, 2));
      }
    }
  } catch (error) {
    console.error('Error:', error.message);
    process.exit(1);
  }
}

if (require.main === module) {
  main();
}