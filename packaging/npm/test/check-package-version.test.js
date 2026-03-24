const assert = require("node:assert/strict");
const http = require("node:http");
const test = require("node:test");
const { EventEmitter } = require("node:events");
const { Readable } = require("node:stream");

const { ensurePackageVersionMissing } = require("../check-package-version");

function installHTTPMock(routes) {
  const originalGet = http.get;

  http.get = (url, callback) => {
    const resolvedURL = typeof url === "string" ? url : url.toString();
    const request = new EventEmitter();

    process.nextTick(() => {
      const route = routes.get(resolvedURL);
      if (!route) {
        request.emit("error", new Error(`unexpected request: ${resolvedURL}`));
        return;
      }

      const response = Readable.from(route.body ? [route.body] : []);
      response.statusCode = route.statusCode;
      response.headers = route.headers || {};
      callback(response);
    });

    return request;
  };

  return () => {
    http.get = originalGet;
  };
}

test("ensurePackageVersionMissing allows unpublished packages", async (t) => {
  const registry = "http://registry.test";
  const restoreHTTP = installHTTPMock(
    new Map([[`${registry}/claudecc`, { body: Buffer.alloc(0), statusCode: 404 }]]),
  );
  t.after(restoreHTTP);

  const version = await ensurePackageVersionMissing({
    packageName: "claudecc",
    registry,
    tag: "v1.2.3",
  });

  assert.equal(version, "1.2.3");
});

test("ensurePackageVersionMissing rejects already published versions", async (t) => {
  const registry = "http://registry.test";
  const restoreHTTP = installHTTPMock(
    new Map([
      [
        `${registry}/claudecc`,
        {
          body: Buffer.from(JSON.stringify({ versions: { "1.2.3": { name: "claudecc" } } })),
          statusCode: 200,
        },
      ],
    ]),
  );
  t.after(restoreHTTP);

  await assert.rejects(
    ensurePackageVersionMissing({
      packageName: "claudecc",
      registry,
      tag: "v1.2.3",
    }),
    /already published/,
  );
});
