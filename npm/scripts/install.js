#!/usr/bin/env node

const crypto = require('node:crypto');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const https = require('node:https');

const pkg = require('../../package.json');
const repo = process.env.CCC_NPM_REPO || 'ishiyama0530/ccc';
const tag = process.env.CCC_NPM_VERSION_TAG || `v${pkg.version}`;
const baseUrl = process.env.CCC_NPM_DOWNLOAD_BASE || `https://github.com/${repo}/releases/download/${tag}`;

const platformMap = {
  darwin: 'darwin',
  linux: 'linux',
  win32: 'windows'
};
const archMap = {
  x64: 'amd64',
  arm64: 'arm64'
};

const goos = platformMap[process.platform];
const goarch = archMap[process.arch];
if (!goos || !goarch) {
  console.error(`Unsupported platform: ${process.platform}/${process.arch}`);
  process.exit(1);
}

const ext = goos === 'windows' ? '.exe' : '';
const assetName = `ccc_${goos}_${goarch}${ext}`;
const checksumsName = 'checksums.txt';

const installDir = path.join(__dirname, '..', 'bin');
const installPath = path.join(installDir, `ccc${ext}`);

main().catch((error) => {
  console.error(`Failed to install ccc: ${error.message}`);
  process.exit(1);
});

async function main() {
  fs.mkdirSync(installDir, { recursive: true });

  const [binary, checksums] = await Promise.all([
    downloadBuffer(`${baseUrl}/${assetName}`),
    downloadText(`${baseUrl}/${checksumsName}`)
  ]);

  const expected = findChecksum(checksums, assetName);
  const actual = crypto.createHash('sha256').update(binary).digest('hex');
  if (expected !== actual) {
    throw new Error(`checksum mismatch for ${assetName}`);
  }

  fs.writeFileSync(installPath, binary);
  if (goos !== 'windows') {
    fs.chmodSync(installPath, 0o755);
  }

  process.stdout.write(`Installed ccc ${tag} (${goos}/${goarch})\n`);
}

function findChecksum(checksums, filename) {
  const line = checksums
    .split(/\r?\n/)
    .map((entry) => entry.trim())
    .find((entry) => entry.endsWith(` ${filename}`));

  if (!line) {
    throw new Error(`checksum for ${filename} not found`);
  }

  const [hash] = line.split(/\s+/);
  return hash;
}

function downloadText(url) {
  return downloadBuffer(url).then((buffer) => buffer.toString('utf8'));
}

function downloadBuffer(url) {
  return new Promise((resolve, reject) => {
    const request = https.get(url, {
      headers: {
        'User-Agent': 'ccc-npm-installer',
        'Accept': 'application/octet-stream'
      }
    }, (response) => {
      if (response.statusCode >= 300 && response.statusCode < 400 && response.headers.location) {
        response.resume();
        resolve(downloadBuffer(response.headers.location));
        return;
      }

      if (response.statusCode !== 200) {
        reject(new Error(`download failed: ${url} (${response.statusCode})`));
        response.resume();
        return;
      }

      const chunks = [];
      response.on('data', (chunk) => chunks.push(chunk));
      response.on('end', () => resolve(Buffer.concat(chunks)));
      response.on('error', reject);
    });

    request.setTimeout(30_000, () => {
      request.destroy(new Error(`timeout while downloading ${url}`));
    });
    request.on('error', reject);
  });
}
