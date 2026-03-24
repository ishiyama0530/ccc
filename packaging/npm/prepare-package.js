#!/usr/bin/env node

const fs = require("node:fs");
const path = require("node:path");

const { npmVersionFromTag } = require("./lib/runtime");

function repoRoot() {
  return path.resolve(__dirname, "..", "..");
}

function parseArgs(argv) {
  const args = new Map();

  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (!arg.startsWith("--")) {
      continue;
    }

    const key = arg.slice(2);
    const value = argv[index + 1];
    if (!value || value.startsWith("--")) {
      throw new Error(`missing value for --${key}`);
    }

    args.set(key, value);
    index += 1;
  }

  return args;
}

function copyDirectory(source, destination) {
  fs.cpSync(source, destination, { recursive: true });
}

function preparePackage({ outDir, tag }) {
  if (!outDir) {
    throw new Error("outDir is required");
  }
  if (!tag) {
    throw new Error("tag is required");
  }

  const version = npmVersionFromTag(tag);
  const root = repoRoot();
  const templatePath = path.join(__dirname, "package.json.template");
  const packageJSON = fs
    .readFileSync(templatePath, "utf8")
    .replace("__VERSION__", version);
  const resolvedOutDir = path.resolve(outDir);

  fs.rmSync(resolvedOutDir, { force: true, recursive: true });
  fs.mkdirSync(resolvedOutDir, { recursive: true });

  copyDirectory(path.join(__dirname, "bin"), path.join(resolvedOutDir, "bin"));
  copyDirectory(path.join(__dirname, "lib"), path.join(resolvedOutDir, "lib"));
  copyDirectory(
    path.join(__dirname, "scripts"),
    path.join(resolvedOutDir, "scripts"),
  );

  fs.writeFileSync(path.join(resolvedOutDir, "package.json"), packageJSON);
  fs.copyFileSync(path.join(root, "README.md"), path.join(resolvedOutDir, "README.md"));
  fs.copyFileSync(path.join(root, "LICENSE"), path.join(resolvedOutDir, "LICENSE"));
  fs.chmodSync(path.join(resolvedOutDir, "bin", "ccc.js"), 0o755);
  fs.chmodSync(path.join(resolvedOutDir, "scripts", "postinstall.js"), 0o755);

  return {
    outDir: resolvedOutDir,
    version,
  };
}

function main(argv = process.argv.slice(2)) {
  const args = parseArgs(argv);
  const tag = args.get("tag");
  const outDir = args.get("out-dir");
  const result = preparePackage({ outDir, tag });
  process.stdout.write(`${result.outDir}\n`);
}

if (require.main === module) {
  try {
    main();
  } catch (error) {
    console.error(error.message);
    process.exit(1);
  }
}

module.exports = {
  main,
  preparePackage,
};
