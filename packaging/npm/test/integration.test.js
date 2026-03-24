const assert = require("node:assert/strict");
const crypto = require("node:crypto");
const fs = require("node:fs");
const http = require("node:http");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");
const { spawnSync } = require("node:child_process");
const { EventEmitter } = require("node:events");
const { Readable } = require("node:stream");

const { preparePackage } = require("../prepare-package");
const { resolveReleaseTarget } = require("../lib/runtime");

function createTarGzArchive({ archivePath, fileName, sourceDir }) {
  const result = spawnSync("tar", ["-czf", archivePath, "-C", sourceDir, fileName], {
    encoding: "utf8",
  });

  assert.equal(result.status, 0, result.stderr || result.stdout);
}

function installHTTPMock(routes, requests) {
  const originalGet = http.get;

  http.get = (url, callback) => {
    const resolvedURL = typeof url === "string" ? url : url.toString();
    requests.push(resolvedURL);

    const request = new EventEmitter();
    process.nextTick(() => {
      const route = routes.get(resolvedURL);
      if (!route) {
        request.emit("error", new Error(`unexpected request: ${resolvedURL}`));
        return;
      }

      const response = Readable.from(route.body ? [route.body] : []);
      response.statusCode = route.statusCode || 200;
      response.headers = route.headers || {};
      callback(response);
    });

    return request;
  };

  return () => {
    http.get = originalGet;
  };
}

test("postinstall downloads the matching release asset and the shim runs it", async (t) => {
  const target = resolveReleaseTarget(process.platform, process.arch);
  if (target.archiveFormat !== "tar.gz") {
    t.skip("integration archive execution is covered on unix targets");
    return;
  }

  const tarVersion = spawnSync("tar", ["--version"], { encoding: "utf8" });
  if (tarVersion.error || tarVersion.status !== 0) {
    t.skip("tar is required for the npm packaging integration test");
    return;
  }

  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "ccc-npm-integration-"));
  const packageDir = path.join(tempDir, "package");
  const releaseDir = path.join(tempDir, "release");
  const version = "v1.2.3";
  const requests = [];

  try {
    preparePackage({ outDir: packageDir, tag: version });

    fs.mkdirSync(releaseDir, { recursive: true });
    const binaryPath = path.join(releaseDir, target.binaryName);
    fs.writeFileSync(binaryPath, "#!/bin/sh\nprintf '%s\\n' \"$@\"\n");
    fs.chmodSync(binaryPath, 0o755);

    const archivePath = path.join(releaseDir, target.archiveName);
    createTarGzArchive({
      archivePath,
      fileName: target.binaryName,
      sourceDir: releaseDir,
    });

    const checksum = crypto
      .createHash("sha256")
      .update(fs.readFileSync(archivePath))
      .digest("hex");

    const downloadBase = "http://ccc.test/download";
    const restoreHTTP = installHTTPMock(
      new Map([
        [
          `${downloadBase}/${version}/${target.archiveName}`,
          {
            body: fs.readFileSync(archivePath),
            headers: { "content-type": "application/gzip" },
          },
        ],
        [
          `${downloadBase}/${version}/checksums.txt`,
          {
            body: Buffer.from(`${checksum}  ${target.archiveName}\n`),
            headers: { "content-type": "text/plain" },
          },
        ],
      ]),
      requests,
    );
    t.after(restoreHTTP);

    const originalDownloadBase = process.env.CCC_NPM_GITHUB_DOWNLOAD_BASE;
    process.env.CCC_NPM_GITHUB_DOWNLOAD_BASE = downloadBase;
    t.after(() => {
      if (originalDownloadBase === undefined) {
        delete process.env.CCC_NPM_GITHUB_DOWNLOAD_BASE;
        return;
      }

      process.env.CCC_NPM_GITHUB_DOWNLOAD_BASE = originalDownloadBase;
    });

    const postinstall = require(path.join(packageDir, "scripts", "postinstall.js"));
    await postinstall.main();

    assert.deepEqual(requests, [
      `${downloadBase}/${version}/${target.archiveName}`,
      `${downloadBase}/${version}/checksums.txt`,
    ]);
    assert.equal(
      fs.existsSync(path.join(packageDir, "runtime", target.binaryName)),
      true,
    );

    const shim = spawnSync(
      "node",
      [path.join(packageDir, "bin", "ccc.js"), "--hello", "world"],
      {
        cwd: packageDir,
        encoding: "utf8",
      },
    );
    assert.equal(shim.status, 0, shim.stderr || shim.stdout);
    assert.equal(shim.stdout, "--hello\nworld\n");
  } finally {
    fs.rmSync(tempDir, { force: true, recursive: true });
  }
});
