#!/usr/bin/env node

const path = require("node:path");

const { spawnInstalledBinary } = require("../lib/runtime");

function main() {
  const result = spawnInstalledBinary({
    args: process.argv.slice(2),
    packageRoot: path.resolve(__dirname, ".."),
  });

  if (result.signal) {
    process.kill(process.pid, result.signal);
    return;
  }

  process.exit(result.status === null ? 1 : result.status);
}

if (require.main === module) {
  try {
    main();
  } catch (error) {
    console.error(error.message);
    process.exit(1);
  }
}

module.exports = { main };
