const crypto = require("node:crypto");
const fs = require("node:fs");
const http = require("node:http");
const https = require("node:https");
const os = require("node:os");
const path = require("node:path");
const { spawnSync } = require("node:child_process");

const DEFAULT_REPO = "ishiyama0530/ccc";
const DEFAULT_DOWNLOAD_BASE = `https://github.com/${DEFAULT_REPO}/releases/download`;
const DEFAULT_CHECKSUMS_NAME = "checksums.txt";
const STABLE_TAG_PATTERN = /^v\d+\.\d+\.\d+$/;
const STABLE_VERSION_PATTERN = /^\d+\.\d+\.\d+$/;

function npmVersionFromTag(tag) {
  if (!STABLE_TAG_PATTERN.test(tag)) {
    throw new Error("stable release tags must look like vX.Y.Z");
  }

  return tag.slice(1);
}

function releaseTagFromPackageVersion(version) {
  if (!STABLE_VERSION_PATTERN.test(version)) {
    throw new Error("package versions must look like X.Y.Z");
  }

  return `v${version}`;
}

function normalizePlatform(platform) {
  switch (platform) {
    case "darwin":
      return "darwin";
    case "linux":
      return "linux";
    case "win32":
      return "windows";
    default:
      throw new Error(`unsupported platform: ${platform}`);
  }
}

function normalizeArch(arch) {
  switch (arch) {
    case "x64":
      return "amd64";
    case "arm64":
      return "arm64";
    default:
      throw new Error(`unsupported architecture: ${arch}`);
  }
}

function resolveReleaseTarget(platform = process.platform, arch = process.arch) {
  const osName = normalizePlatform(platform);
  const archName = normalizeArch(arch);
  const archiveFormat = osName === "windows" ? "zip" : "tar.gz";
  const archiveName = `ccc_${osName}_${archName}.${archiveFormat}`;
  const binaryName = osName === "windows" ? "ccc.exe" : "ccc";

  return {
    archiveFormat,
    archiveName,
    arch: archName,
    binaryName,
    os: osName,
  };
}

function resolveInstalledBinaryPath({
  packageRoot,
  platform = process.platform,
  arch = process.arch,
}) {
  const resolvedPackageRoot = packageRoot || path.resolve(__dirname, "..");
  const { binaryName } = resolveReleaseTarget(platform, arch);
  return path.join(resolvedPackageRoot, "runtime", binaryName);
}

function resolveDownloadBase(env = process.env) {
  if (env.CCC_NPM_GITHUB_DOWNLOAD_BASE) {
    return env.CCC_NPM_GITHUB_DOWNLOAD_BASE.replace(/\/+$/, "");
  }

  const repo = env.CCC_NPM_GITHUB_REPO || DEFAULT_REPO;
  return `https://github.com/${repo}/releases/download`;
}

function findChecksumForArchive(checksumsText, archiveName) {
  const lines = checksumsText.split(/\r?\n/);
  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed) {
      continue;
    }

    const match = /^([a-fA-F0-9]+)\s+\*?(.+)$/.exec(trimmed);
    if (!match) {
      continue;
    }

    if (match[2] === archiveName) {
      return match[1].toLowerCase();
    }
  }

  throw new Error(`checksum for ${archiveName} not found`);
}

function verifyChecksum(archivePath, checksumsText, archiveName) {
  const expected = findChecksumForArchive(checksumsText, archiveName);
  const actual = crypto
    .createHash("sha256")
    .update(fs.readFileSync(archivePath))
    .digest("hex")
    .toLowerCase();

  if (actual !== expected) {
    throw new Error(`checksum mismatch for ${archiveName}`);
  }

  return actual;
}

function download(url, redirectCount = 0) {
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
        resolve(download(redirectURL, redirectCount + 1));
        return;
      }

      if (statusCode !== 200) {
        response.resume();
        reject(new Error(`download failed: ${url} returned ${statusCode}`));
        return;
      }

      const chunks = [];
      response.on("data", (chunk) => chunks.push(chunk));
      response.on("end", () => resolve(Buffer.concat(chunks)));
      response.on("error", reject);
    });

    request.on("error", reject);
  });
}

async function downloadToFile(url, destination) {
  const body = await download(url);
  fs.mkdirSync(path.dirname(destination), { recursive: true });
  fs.writeFileSync(destination, body);
}

async function downloadText(url) {
  const body = await download(url);
  return body.toString("utf8");
}

function extractArchive(archivePath, destinationDir, archiveFormat) {
  fs.mkdirSync(destinationDir, { recursive: true });

  const args =
    archiveFormat === "zip"
      ? ["-xf", archivePath, "-C", destinationDir]
      : ["-xzf", archivePath, "-C", destinationDir];
  const result = spawnSync("tar", args, { encoding: "utf8" });

  if (result.error) {
    throw new Error(
      `failed to extract ${path.basename(archivePath)}: ${result.error.message}`,
    );
  }

  if (result.status !== 0) {
    throw new Error(
      `failed to extract ${path.basename(archivePath)}: ${
        result.stderr.trim() || result.stdout.trim() || `tar exited with status ${result.status}`
      }`,
    );
  }
}

async function installPackageBinary({
  packageRoot,
  packageVersion,
  env = process.env,
  platform = process.platform,
  arch = process.arch,
}) {
  const resolvedPackageRoot = packageRoot || path.resolve(__dirname, "..");
  const target = resolveReleaseTarget(platform, arch);
  const checksumsName = env.CCC_NPM_CHECKSUMS_NAME || DEFAULT_CHECKSUMS_NAME;
  const downloadBase = resolveDownloadBase(env);
  const releaseTag =
    env.CCC_NPM_RELEASE_TAG || releaseTagFromPackageVersion(packageVersion);
  const runtimeDir = path.join(resolvedPackageRoot, "runtime");
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "ccc-npm-"));
  const archivePath = path.join(tempDir, target.archiveName);

  try {
    await downloadToFile(
      `${downloadBase}/${releaseTag}/${target.archiveName}`,
      archivePath,
    );
    const checksumsText = await downloadText(
      `${downloadBase}/${releaseTag}/${checksumsName}`,
    );
    verifyChecksum(archivePath, checksumsText, target.archiveName);

    fs.rmSync(runtimeDir, { force: true, recursive: true });
    extractArchive(archivePath, runtimeDir, target.archiveFormat);

    const binaryPath = path.join(runtimeDir, target.binaryName);
    if (!fs.existsSync(binaryPath)) {
      throw new Error(`${target.binaryName} was not found in ${target.archiveName}`);
    }

    if (target.os !== "windows") {
      fs.chmodSync(binaryPath, 0o755);
    }

    return binaryPath;
  } finally {
    fs.rmSync(tempDir, { force: true, recursive: true });
  }
}

function spawnInstalledBinary({
  args = [],
  arch = process.arch,
  cwd = process.cwd(),
  env = process.env,
  packageRoot,
  platform = process.platform,
  stdio = "inherit",
}) {
  const binaryPath = resolveInstalledBinaryPath({ packageRoot, platform, arch });
  if (!fs.existsSync(binaryPath)) {
    throw new Error(
      "ccc binary is not installed. Reinstall the package to download the matching release asset.",
    );
  }

  const result = spawnSync(binaryPath, args, {
    cwd,
    encoding: "utf8",
    env,
    stdio,
    windowsHide: false,
  });

  if (result.error) {
    throw result.error;
  }

  return result;
}

module.exports = {
  DEFAULT_CHECKSUMS_NAME,
  DEFAULT_DOWNLOAD_BASE,
  DEFAULT_REPO,
  downloadText,
  downloadToFile,
  findChecksumForArchive,
  installPackageBinary,
  npmVersionFromTag,
  releaseTagFromPackageVersion,
  resolveDownloadBase,
  resolveInstalledBinaryPath,
  resolveReleaseTarget,
  spawnInstalledBinary,
  verifyChecksum,
};
