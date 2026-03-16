#!/usr/bin/env node

const { spawnSync } = require('node:child_process');
const { existsSync } = require('node:fs');
const path = require('node:path');

const exe = process.platform === 'win32' ? 'ccc.exe' : 'ccc';
const binaryPath = path.join(__dirname, '..', 'bin', exe);

if (!existsSync(binaryPath)) {
  console.error('ccc binary is not installed. Reinstall with: npm install -g @ishiyama0530/ccc');
  process.exit(1);
}

const result = spawnSync(binaryPath, process.argv.slice(2), { stdio: 'inherit' });
if (result.error) {
  console.error(result.error.message);
  process.exit(1);
}
process.exit(result.status ?? 1);
