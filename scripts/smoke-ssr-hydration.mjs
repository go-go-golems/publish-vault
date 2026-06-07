#!/usr/bin/env node

import { spawn } from "child_process";
import { createRequire } from "module";
import { createServer } from "net";
import { dirname, join, resolve } from "path";
import { fileURLToPath } from "url";

const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url));
const ROOT = resolve(SCRIPT_DIR, "..");
const WEB_DIR = join(ROOT, "web");
const VAULT_DIR = join(ROOT, "vault-example");
const DEFAULT_TIMEOUT_MS = 60_000;
const webRequire = createRequire(join(WEB_DIR, "package.json"));
const playwright = await import(webRequire.resolve("playwright"));
const { chromium } = playwright.default ?? playwright;

const processes = [];

function log(message) {
  console.log(`[smoke:ssr] ${message}`);
}

function fail(message) {
  throw new Error(`[smoke:ssr] ${message}`);
}

function run(command, args, options = {}) {
  log(`$ ${[command, ...args].join(" ")}`);
  return new Promise((resolvePromise, reject) => {
    const child = spawn(command, args, {
      cwd: options.cwd ?? ROOT,
      env: { ...process.env, ...options.env },
      stdio: options.stdio ?? "inherit",
    });

    child.on("error", reject);
    child.on("close", (code) => {
      if (code === 0) resolvePromise();
      else reject(new Error(`${command} ${args.join(" ")} exited with ${code}`));
    });
  });
}

function start(command, args, options = {}) {
  log(`start: ${[command, ...args].join(" ")}`);
  const child = spawn(command, args, {
    cwd: options.cwd ?? ROOT,
    detached: true,
    env: { ...process.env, ...options.env },
    stdio: ["ignore", "pipe", "pipe"],
  });

  processes.push(child);
  child.stdout.on("data", (chunk) => process.stdout.write(`[${options.name ?? command}] ${chunk}`));
  child.stderr.on("data", (chunk) => process.stderr.write(`[${options.name ?? command}] ${chunk}`));
  child.on("exit", (code, signal) => {
    if (!options.allowExit) {
      log(`${options.name ?? command} exited early: code=${code} signal=${signal}`);
    }
  });
  return child;
}

async function getFreePort() {
  return await new Promise((resolvePromise, reject) => {
    const server = createServer();
    server.unref();
    server.on("error", reject);
    server.listen(0, "127.0.0.1", () => {
      const address = server.address();
      const port = typeof address === "object" && address ? address.port : null;
      server.close(() => {
        if (!port) reject(new Error("Could not allocate a free port"));
        else resolvePromise(port);
      });
    });
  });
}

async function waitFor(url, description, timeoutMs = DEFAULT_TIMEOUT_MS) {
  const started = Date.now();
  let lastError = null;

  while (Date.now() - started < timeoutMs) {
    try {
      const response = await fetch(url);
      if (response.ok) return response;
      lastError = new Error(`${description} returned ${response.status}`);
    } catch (error) {
      lastError = error;
    }
    await new Promise((resolvePromise) => setTimeout(resolvePromise, 500));
  }

  fail(`Timed out waiting for ${description} at ${url}: ${lastError?.message ?? "unknown error"}`);
}

async function assertRawSSR(baseUrl) {
  const response = await fetch(`${baseUrl}/`);
  const html = await response.text();
  const poweredBy = response.headers.get("x-powered-by") ?? "";

  if (!response.ok) fail(`GET / returned ${response.status}`);
  if (!poweredBy.toLowerCase().includes("express")) {
    fail(`GET / does not look proxied from SSR sidecar; X-Powered-By=${poweredBy || "<missing>"}`);
  }
  if (html.includes('<div id="root"></div>')) {
    fail("GET / returned an empty SPA root instead of populated SSR HTML");
  }
  if (html.includes('/assets/index.js')) {
    fail("GET / returned fallback index shell with /assets/index.js instead of Vite hashed assets");
  }

  if (!html.includes('<div id="root"><')) {
    fail("GET / did not contain populated server-rendered root markup");
  }
  if (!html.includes("Index") || !html.includes("window.__PRELOADED_STATE__")) {
    fail("GET / did not contain expected SSR note content and preloaded state");
  }

  log(`raw SSR HTML ok (${html.length} bytes, X-Powered-By=${poweredBy})`);
}

async function assertBrowserHydration(baseUrl) {
  const browser = await chromium.launch({ headless: true });
  const consoleProblems = [];
  const pageErrors = [];
  const attachProblemListeners = (page) => {
    page.on("console", (message) => {
      if (["error", "warning"].includes(message.type())) {
        consoleProblems.push(`${message.type()}: ${message.text()}`);
      }
    });
    page.on("pageerror", (error) => pageErrors.push(error.message));
  };

  try {
    const mobilePage = await browser.newPage({ viewport: { width: 390, height: 844 } });
    attachProblemListeners(mobilePage);
    await mobilePage.goto(`${baseUrl}/`, { waitUntil: "networkidle" });
    if ((await mobilePage.getByTestId("mobile-sidebar-backdrop").count()) > 0) {
      fail("mobile fresh load unexpectedly rendered the sidebar backdrop");
    }
    if ((await mobilePage.getByTestId("mobile-sidebar-drawer").count()) > 0) {
      fail("mobile fresh load unexpectedly rendered the off-canvas sidebar drawer");
    }
    await mobilePage.getByRole("button", { name: /Toggle sidebar/i }).click();
    await mobilePage.getByTestId("mobile-sidebar-drawer").waitFor({ timeout: 10_000 });
    await mobilePage.close();

    const page = await browser.newPage();
    attachProblemListeners(page);
    await page.goto(`${baseUrl}/`, { waitUntil: "networkidle" });
    const homeTitle = await page.title();
    if (!homeTitle.includes("Test Vault")) fail(`Unexpected / title: ${homeTitle}`);

    await page.goto(`${baseUrl}/note/index`, { waitUntil: "networkidle" });
    const noteTitle = await page.title();
    if (!noteTitle.includes("Index") || !noteTitle.includes("Test Vault")) {
      fail(`Unexpected /note/index title: ${noteTitle}`);
    }

    const targetButton = page.getByRole("button", { name: /Epistemology/i }).first();
    if ((await targetButton.count()) > 0) {
      await targetButton.click();
      await page.waitForURL("**/note/philosophy/epistemology", { timeout: 10_000 });
      await page.waitForLoadState("networkidle");
      await page.getByRole("heading", { name: /Epistemology/i }).first().waitFor({ timeout: 10_000 });
      await page.waitForFunction(() => document.title.includes("Epistemology"), undefined, { timeout: 10_000 });
      const navTitle = await page.title();
      if (!navTitle.includes("Epistemology") || !navTitle.includes("Test Vault")) {
        fail(`Unexpected sidebar navigation title: ${navTitle}`);
      }
    } else {
      log("sidebar Epistemology button not found; direct note hydration checks still passed");
    }

    if (pageErrors.length > 0) fail(`page errors:\n${pageErrors.join("\n")}`);
    if (consoleProblems.length > 0) fail(`console warnings/errors:\n${consoleProblems.join("\n")}`);

    log("browser hydration ok (0 console warnings/errors)");
  } finally {
    await browser.close();
  }
}

async function cleanup() {
  for (const child of processes.reverse()) {
    try {
      process.kill(-child.pid, "SIGTERM");
    } catch {
      try { child.kill("SIGTERM"); } catch {}
    }
  }
  await new Promise((resolvePromise) => setTimeout(resolvePromise, 500));
  for (const child of processes.reverse()) {
    try {
      process.kill(-child.pid, "SIGKILL");
    } catch {
      try { child.kill("SIGKILL"); } catch {}
    }
  }
}

async function main() {
  const skipBuild = process.argv.includes("--skip-build");
  const backendPort = await getFreePort();
  const ssrPort = await getFreePort();
  const backendBase = `http://127.0.0.1:${backendPort}`;
  const ssrBase = `http://127.0.0.1:${ssrPort}`;

  log(`using backend=${backendBase} ssr=${ssrBase}`);

  if (!skipBuild) {
    await run("pnpm", ["--dir", "web", "build:all"]);
  }

  start("node", ["server.mjs"], {
    name: "ssr",
    cwd: WEB_DIR,
    env: {
      SSR_PORT: String(ssrPort),
      API_BASE: backendBase,
      BASE_URL: backendBase,
    },
  });

  start("go", [
    "run",
    "./cmd/retro-obsidian-publish",
    "serve",
    "--vault",
    VAULT_DIR,
    "--vault-name",
    "TestVault",
    "--page-title",
    "Test Vault",
    "--port",
    String(backendPort),
    "--ssr-url",
    ssrBase,
    "--watch=false",
  ], {
    name: "backend",
    cwd: ROOT,
    env: { GOWORK: "off" },
  });

  await waitFor(`${backendBase}/api/config`, "Go backend /api/config");
  await waitFor(`${ssrBase}/health`, "SSR sidecar /health");
  await assertRawSSR(backendBase);
  await assertBrowserHydration(backendBase);
  log("PASS");
}

process.on("SIGINT", async () => {
  await cleanup();
  process.exit(130);
});
process.on("SIGTERM", async () => {
  await cleanup();
  process.exit(143);
});

main()
  .catch((error) => {
    console.error(error);
    process.exitCode = 1;
  })
  .finally(cleanup);
