#!/usr/bin/env node

const http = require("node:http");
const https = require("node:https");

const { npmVersionFromTag } = require("./lib/runtime");

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

function fetchJSON(url, redirectCount = 0) {
  return new Promise((resolve, reject) => {
    const client = url.startsWith("https:") ? https : http;
    const request = client.get(url, (response) => {
      const statusCode = response.statusCode || 0;

      if (
        statusCode >= 300 &&
        statusCode < 400 &&
        response.headers.location &&
        redirectCount < 5
      ) {
        response.resume();
        const redirectURL = new URL(response.headers.location, url).toString();
        resolve(fetchJSON(redirectURL, redirectCount + 1));
        return;
      }

      if (statusCode === 404) {
        response.resume();
        resolve(null);
        return;
      }

      if (statusCode !== 200) {
        response.resume();
        reject(new Error(`registry request failed with status ${statusCode}`));
        return;
      }

      const chunks = [];
      response.on("data", (chunk) => chunks.push(chunk));
      response.on("end", () => {
        try {
          resolve(JSON.parse(Buffer.concat(chunks).toString("utf8")));
        } catch (error) {
          reject(error);
        }
      });
      response.on("error", reject);
    });

    request.on("error", reject);
  });
}

async function ensurePackageVersionMissing({ packageName, registry, tag }) {
  const version = npmVersionFromTag(tag);
  const baseURL = registry.replace(/\/+$/, "");
  const metadata = await fetchJSON(`${baseURL}/${encodeURIComponent(packageName)}`);

  if (metadata && metadata.versions && metadata.versions[version]) {
    throw new Error(`${packageName}@${version} is already published`);
  }

  return version;
}

async function main(argv = process.argv.slice(2)) {
  const args = parseArgs(argv);
  const packageName = args.get("package");
  const registry = args.get("registry") || "https://registry.npmjs.org";
  const tag = args.get("tag");

  if (!packageName) {
    throw new Error("--package is required");
  }
  if (!tag) {
    throw new Error("--tag is required");
  }

  await ensurePackageVersionMissing({ packageName, registry, tag });
}

if (require.main === module) {
  main().catch((error) => {
    console.error(error.message);
    process.exit(1);
  });
}

module.exports = {
  ensurePackageVersionMissing,
  main,
};
