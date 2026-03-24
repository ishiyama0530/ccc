const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");
const crypto = require("node:crypto");

const {
  npmVersionFromTag,
  resolveInstalledBinaryPath,
  resolveReleaseTarget,
  verifyChecksum,
} = require("../lib/runtime");

test("npmVersionFromTag strips the leading v from stable tags", () => {
  assert.equal(npmVersionFromTag("v1.2.3"), "1.2.3");
});

test("npmVersionFromTag rejects prerelease tags", () => {
  assert.throws(
    () => npmVersionFromTag("v1.2.3-beta.1"),
    /stable release tags must look like vX.Y.Z/,
  );
});

test("resolveReleaseTarget uses tar.gz for unix targets", () => {
  assert.deepEqual(resolveReleaseTarget("linux", "x64"), {
    archiveFormat: "tar.gz",
    archiveName: "claudecc_linux_amd64.tar.gz",
    arch: "amd64",
    binaryName: "claudecc",
    os: "linux",
  });
});

test("resolveReleaseTarget uses zip archives and .exe on windows", () => {
  assert.deepEqual(resolveReleaseTarget("win32", "arm64"), {
    archiveFormat: "zip",
    archiveName: "claudecc_windows_arm64.zip",
    arch: "arm64",
    binaryName: "claudecc.exe",
    os: "windows",
  });
});

test("verifyChecksum accepts matching checksums", () => {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "claudecc-runtime-test-"));
  const archivePath = path.join(tempDir, "archive.tar.gz");
  const archiveBody = Buffer.from("archive-body");

  try {
    fs.writeFileSync(archivePath, archiveBody);
    const checksum = crypto.createHash("sha256").update(archiveBody).digest("hex");
    const actual = verifyChecksum(
      archivePath,
      `${checksum}  archive.tar.gz\n`,
      "archive.tar.gz",
    );
    assert.equal(actual, checksum);
  } finally {
    fs.rmSync(tempDir, { force: true, recursive: true });
  }
});

test("verifyChecksum rejects mismatches", () => {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "claudecc-runtime-test-"));
  const archivePath = path.join(tempDir, "archive.tar.gz");

  try {
    fs.writeFileSync(archivePath, "archive-body");
    assert.throws(
      () => verifyChecksum(archivePath, "deadbeef  archive.tar.gz\n", "archive.tar.gz"),
      /checksum mismatch/,
    );
  } finally {
    fs.rmSync(tempDir, { force: true, recursive: true });
  }
});

test("resolveInstalledBinaryPath uses .exe for windows installs", () => {
  const binaryPath = resolveInstalledBinaryPath({
    arch: "x64",
    packageRoot: "/tmp/claudecc-package",
    platform: "win32",
  });

  assert.equal(
    binaryPath,
    path.join("/tmp/claudecc-package", "runtime", "claudecc.exe"),
  );
});
