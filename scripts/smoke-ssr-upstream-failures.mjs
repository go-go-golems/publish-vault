#!/usr/bin/env node
//
// smoke-ssr-upstream-failures.mjs — regression test for PV-SLUG-020.
//
// The SSR sidecar used to collapse every failure to fetch a note into
// `404 Note not found`: a genuine miss, an unreachable backend, a 5xx, and an
// unparseable body were indistinguishable. A note that exists but whose backend
// was restarting was reported to users, crawlers and caches as a note that does
// not exist — and cached as such.
//
// This drives the *real* web/server.mjs against a stub backend that can be made
// to fail in each distinct way, and asserts the four outcomes map to different
// status codes. (scripts/03-ssr-conflation-repro.mjs in the ticket is the
// executable specification; it copies the old fetchAPI and shows the collapse.)
//
// Usage: node scripts/smoke-ssr-upstream-failures.mjs [--skip-build]

import { spawn } from "child_process";
import { createServer } from "http";
import { createServer as createSocketServer } from "net";
import { dirname, join, resolve } from "path";
import { fileURLToPath } from "url";

const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url));
const ROOT = resolve(SCRIPT_DIR, "..");
const WEB_DIR = join(ROOT, "web");
const SLUG = "transcripts/2026/08/09/designing-rag/the_algebra_of_intervention_fields";
const TIMEOUT_MS = 60_000;

const processes = [];
let failures = 0;

function log(message) {
  console.log(`[smoke:upstream] ${message}`);
}

/** How the stub backend should answer the note endpoint on the next request. */
let noteMode = "ok";

const NOTE = {
  slug: SLUG,
  title: "The Algebra of Intervention Fields",
  path: `${SLUG}.md`,
  frontmatter: {},
  tags: [],
  excerpt: "An excerpt.",
  html: "<h1>The Algebra of Intervention Fields</h1>\n<p>Body.</p>\n",
  wikiLinks: [],
  backlinks: [],
  modTime: "2026-08-09T00:00:00Z",
};

function startStubBackend(port) {
  const server = createServer((req, res) => {
    const path = decodeURIComponent(req.url.split("?")[0]);
    const json = (body, status = 200) => {
      res.writeHead(status, { "Content-Type": "application/json" });
      res.end(JSON.stringify(body));
    };

    if (path === "/api/config") {
      return json({ vaultName: "StubVault", pageTitle: "Stub Vault", notes: 1 });
    }
    if (path === "/api/tree") {
      return json({ name: "root", path: "", isFolder: true, children: [] });
    }
    if (path === "/api/notes") return json([]);

    if (path.startsWith("/api/notes/")) {
      switch (noteMode) {
        case "not_found":
          return json({ error: "note not found" }, 404);
        case "server_error":
          return json({ error: "boom" }, 500);
        case "bad_body":
          res.writeHead(200, { "Content-Type": "application/json" });
          return res.end("this is not json{{{");
        case "unreachable":
          // Hang up mid-request: fetch() rejects, exactly as it does when the
          // Go process is OOMKilled or restarting. Isolating the failure to
          // this one endpoint keeps /api/config healthy so the assertion is
          // about the note path specifically.
          return req.socket.destroy();
        default:
          return json(NOTE);
      }
    }
    json({ error: "not found" }, 404);
  });

  return new Promise((resolvePromise) => {
    server.listen(port, "127.0.0.1", () => resolvePromise(server));
  });
}

function start(command, args, options = {}) {
  const child = spawn(command, args, {
    cwd: options.cwd ?? ROOT,
    detached: true,
    env: { ...process.env, ...options.env },
    stdio: ["ignore", "pipe", "pipe"],
  });
  processes.push(child);
  child.stdout.on("data", c => process.stdout.write(`[${options.name}] ${c}`));
  child.stderr.on("data", c => process.stderr.write(`[${options.name}] ${c}`));
  return child;
}

function run(command, args) {
  log(`$ ${[command, ...args].join(" ")}`);
  return new Promise((resolvePromise, reject) => {
    const child = spawn(command, args, { cwd: ROOT, stdio: "inherit" });
    child.on("error", reject);
    child.on("close", code =>
      code === 0 ? resolvePromise() : reject(new Error(`exited with ${code}`))
    );
  });
}

async function getFreePort() {
  return await new Promise((resolvePromise, reject) => {
    const server = createSocketServer();
    server.unref();
    server.on("error", reject);
    server.listen(0, "127.0.0.1", () => {
      const port = server.address().port;
      server.close(() => (port ? resolvePromise(port) : reject(new Error("no port"))));
    });
  });
}

async function waitFor(url, timeoutMs = TIMEOUT_MS) {
  const started = Date.now();
  let lastError = null;
  while (Date.now() - started < timeoutMs) {
    try {
      const response = await fetch(url);
      if (response.ok) return;
      lastError = new Error(`returned ${response.status}`);
    } catch (error) {
      lastError = error;
    }
    await new Promise(r => setTimeout(r, 300));
  }
  throw new Error(`timed out waiting for ${url}: ${lastError?.message}`);
}

async function expect(label, url, wantStatus, wantBodyPart) {
  const response = await fetch(url, { redirect: "manual" });
  const body = await response.text();
  const okStatus = response.status === wantStatus;
  const okBody = wantBodyPart === undefined || body.includes(wantBodyPart);
  // A 404 or 5xx emitted while the backend was down must never be cached: a CDN
  // would otherwise hold a permanent verdict on a URL that is in fact fine.
  const cacheControl = response.headers.get("cache-control") ?? "";
  const okCache = wantStatus === 200 || cacheControl.includes("no-store");

  if (okStatus && okBody && okCache) {
    log(`PASS  ${label.padEnd(34)} -> ${response.status}`);
    return;
  }
  failures++;
  log(
    `FAIL  ${label.padEnd(34)} -> got ${response.status} (want ${wantStatus})` +
      (okBody ? "" : `, body missing ${JSON.stringify(wantBodyPart)}`) +
      (okCache ? "" : `, Cache-Control=${JSON.stringify(cacheControl)}`)
  );
  log(`      body: ${JSON.stringify(body.slice(0, 120))}`);
}

async function main() {
  if (!process.argv.includes("--skip-build")) {
    await run("pnpm", ["--dir", "web", "build:all"]);
  }

  const backendPort = await getFreePort();
  const ssrPort = await getFreePort();
  const ssrBase = `http://127.0.0.1:${ssrPort}`;
  const backendBase = `http://127.0.0.1:${backendPort}`;

  const backend = await startStubBackend(backendPort);
  log(`stub backend on ${backendBase}, ssr on ${ssrBase}`);

  start("node", ["server.mjs"], {
    name: "ssr",
    cwd: WEB_DIR,
    env: { SSR_PORT: String(ssrPort), API_BASE: backendBase, BASE_URL: backendBase },
  });
  await waitFor(`${ssrBase}/health`);

  const noteUrl = `${ssrBase}/note/${SLUG}`;

  noteMode = "ok";
  await expect("A. note exists, API healthy", noteUrl, 200, "Algebra of Intervention");

  noteMode = "not_found";
  await expect("B. note genuinely absent", noteUrl, 404, "Note not found");

  noteMode = "unreachable";
  await expect("C. backend hangs up", noteUrl, 503, "Backend unavailable");

  noteMode = "server_error";
  await expect("D. backend returns 5xx", noteUrl, 502, "Backend error");

  noteMode = "bad_body";
  await expect("E. backend returns bad JSON", noteUrl, 502, "Backend error");

  // The production case: the Go process is gone entirely, so even /api/config
  // fails. This used to render a 200 page titled "undefined" or a 404.
  await new Promise(r => backend.close(r));
  await expect("F. backend fully down", noteUrl, 503, "Backend unavailable");

  if (failures > 0) {
    log(`FAIL — ${failures} assertion(s) failed`);
    process.exitCode = 1;
    return;
  }
  log("PASS — every upstream failure maps to a distinct, truthful status");
}

try {
  await main();
} catch (error) {
  log(`ERROR ${error.message}`);
  process.exitCode = 1;
} finally {
  for (const child of processes.reverse()) {
    try {
      process.kill(-child.pid, "SIGKILL");
    } catch {
      try {
        child.kill("SIGKILL");
      } catch {}
    }
  }
}
