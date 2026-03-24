#!/usr/bin/env node

const path = require("node:path");

const pkg = require("../package.json");
const { installPackageBinary } = require("../lib/runtime");

async function main() {
  await installPackageBinary({
    packageRoot: path.resolve(__dirname, ".."),
    packageVersion: pkg.version,
  });
}

if (require.main === module) {
  main().catch((error) => {
    console.error(`failed to install ccc: ${error.message}`);
    process.exit(1);
  });
}

module.exports = { main };
