# License Scanner

[![npm](https://img.shields.io/npm/v/@stefanoa1/license-scanner.svg)](https://www.npmjs.com/package/@stefanoa1/license-scanner)
[![GitHub Actions](https://github.com/stefano/license-scanner/actions/workflows/ci.yml/badge.svg)](https://github.com/stefano/license-scanner/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

A high-performance npm package that scans project dependencies to detect and report their licenses. Uses a Go binary for fast scanning wrapped in a Node.js package for easy integration.

## Installation

```console
npm install @stefanoa1/license-scanner
```
**or** 

```console
pnpm install @stefanoa1/license-scanner
```
Or any other package install method should be covered.

It supports Node.js >= v16.

## Usage

### CLI Usage

```bash
# Basic scan (includes summary by default)
license-scanner

# Scan production dependencies only
license-scanner --prod-only

# Generate HTML report
license-scanner --format html --output report.html

# Skip license summary
license-scanner --no-summary

# Scan specific directory
license-scanner /path/to/project
```

### Programmatic Usage

```javascript
const { scanLicenses } = require('@stefanoa1/license-scanner');

// Scan current directory
const result = await scanLicenses('.');

console.log(result.summary);
// {
//   totalDependencies: 245,
//   uniqueLicenses: ["MIT", "Apache-2.0", "BSD-3-Clause"],
//   riskLevel: "low",
//   conflicts: [],
//   recommendations: ["All licenses are permissive and compatible"]
// }

console.log(result.dependencies);
// [
//   {
//     name: "react",
//     version: "18.2.0",
//     license: "MIT",
//     confidence: 1.0,
//     source: "package.json"
//   }
// ]
```

## Features

- **⚡ High Performance**: Go-powered core for fast file system traversal and pattern matching
- **🔍 Multi-Source Detection**: Analyzes package.json files, LICENSE files, and lock files
- **📊 Confidence Scoring**: Rates license detection confidence from 0.0 to 1.0
- **🌍 Cross-Platform**: Works on Linux, macOS, and Windows
- **📦 Multiple Package Managers**: Supports npm, yarn, and pnpm
- **🎯 Zero Dependencies**: No runtime dependencies for fast installation
- **📈 Comprehensive Reports**: Detailed license analysis with compatibility insights

## Supported Package Managers

- **npm** (package-lock.json)
- **yarn** (yarn.lock)
- **pnpm** (pnpm-lock.yaml)

## Confidence Scoring System

- **1.0**: Explicit license field in package.json
- **0.9**: LICENSE file with clear license pattern match (e.g., MIT, Apache-2.0)
- **0.8**: LICENSE file with recognizable license text patterns
- **0.2**: LICENSE file exists but patterns not recognized
- **0.0**: No license information found

## Supported License Types

- MIT, Apache-2.0, GPL-2.0/3.0, BSD-2/3-Clause, ISC
- Handles both string and object license fields
- Recognizes common license variations (e.g., "apache2", "gplv3")

## Output Example

```json
{
  "summary": {
    "totalDependencies": 69,
    "uniqueLicenses": ["MIT", "Apache-2.0"],
    "riskLevel": "low",
    "conflicts": [],
    "recommendations": ["All licenses are permissive and compatible"]
  },
  "dependencies": [
    {
      "name": "lodash",
      "version": "4.17.21",
      "license": "MIT",
      "confidence": 1.0,
      "source": "package.json"
    },
    {
      "name": "express",
      "version": "4.18.2",
      "license": "MIT",
      "confidence": 1.0,
      "source": "package.json"
    }
  ]
}
```

