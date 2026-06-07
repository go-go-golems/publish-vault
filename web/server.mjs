// server.mjs — Node.js SSR sidecar for Retro Obsidian Publish.
//
// This is a lightweight Express server that:
// 1. Receives page requests from the Go server's reverse proxy
// 2. Pre-fetches data from the Go API (localhost:8080)
// 3. Renders the React app to HTML using renderToString
// 4. Returns complete HTML with preloaded state for client hydration
//
// In production (k3s), this runs as a sidecar container in the same pod
// as the Go server. In local dev, it runs alongside the Vite dev server
// and Go server.

import express from "express";
import { readFileSync } from "fs";

// --- Config ---
const PORT = parseInt(process.env.SSR_PORT || "8089", 10);
const API_BASE = process.env.API_BASE || "http://localhost:8080";
const BASE_URL = process.env.BASE_URL || "http://localhost:8080";

// --- Dynamic import of the SSR bundle ---
const { renderApp } = await import("./dist/ssr/entry-server.js");

// --- Express app ---
const app = express();

// Health check endpoint — used by Go server and k8s probes
app.get("/health", (_req, res) => {
  res.json({ ok: true });
});

// Helper: fetch JSON from the Go API
async function fetchAPI(path) {
  try {
    const res = await fetch(`${API_BASE}${path}`);
    if (!res.ok) return null;
    return await res.json();
  } catch {
    return null;
  }
}

// Parse URL path into route type + optional slug
function parseRoute(pathname) {
  if (pathname === "/search") return { type: "search" };
  if (pathname.startsWith("/note/")) {
    const slug = pathname.replace(/^\/note\//, "");
    return { type: "note", slug };
  }
  return { type: "home" };
}

// Read the SPA index.html shell (built by Vite)
function getIndexHtml() {
  try {
    return readFileSync("./dist/index.html", "utf-8");
  } catch {
    // Fallback: minimal shell
    return `<!doctype html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Retro Obsidian Publish</title>
  <link rel="stylesheet" href="/assets/index.css">
</head>
<body>
  <div id="root"><!--SSR_CONTENT--></div>
  <script>/*PRELOADED_STATE*/</script>
  <script type="module" src="/assets/index.js"></script>
</body>
</html>`;
  }
}

// Serialize state for inline <script>, escaping dangerous characters
function serializeForInlineScript(value) {
  return JSON.stringify(value)
    .replace(/</g, "\\u003c")
    .replace(/>/g, "\\u003e")
    .replace(/&/g, "\\u0026")
    .replace(/\u2028/g, "\\u2028")
    .replace(/\u2029/g, "\\u2029");
}

// Cache the index.html template
let indexHtmlTemplate = null;

// Catch-all page handler (Express 4 wildcard syntax)
app.get("*", async (req, res) => {
  try {
    const url = req.originalUrl;
    const pathname = req.path;
    const route = parseRoute(pathname);

    // Load template once
    if (!indexHtmlTemplate) {
      indexHtmlTemplate = getIndexHtml();
    }

    // 1. Pre-fetch common data from the Go API
    const [config, notes, tree] = await Promise.all([
      fetchAPI("/api/config"),
      fetchAPI("/api/notes"),
      fetchAPI("/api/tree"),
    ]);

    // 2. Pre-fetch route-specific data
    let note = null;
    if (route.type === "note" && route.slug) {
      note = await fetchAPI(`/api/notes/${encodeURIComponent(route.slug)}`);
    }

    // 3. Render React to HTML
    const { html, preloadedState } = await renderApp(url, {
      config,
      notes,
      tree,
      note,
    });
    const serializedPreloadedState = serializeForInlineScript(preloadedState);

    // 4. Determine page title and description
    const vaultName = config?.vaultName || "Vault";
    const title = note?.title
      ? `${note.title} — ${vaultName}`
      : `${config?.pageTitle || vaultName}`;
    const description =
      note?.excerpt ||
      (notes?.length
        ? `${vaultName}: ${notes.length} notes`
        : `${vaultName} documentation`);

    // 5. Assemble HTML
    let htmlPage = indexHtmlTemplate;

    // Inject server-rendered React content into <div id="root">
    htmlPage = htmlPage.replace(
      /<div id="root">([\s\S]*?)<\/div>/,
      `<div id="root">${html}</div>`
    );

    // Add <noscript> fallback with headings and text for agent readability
    let noscriptContent = "";
    if (note?.title) {
      noscriptContent = `
  <h1>${note.title.replace(/</g, "&lt;")}</h1>
  <p>${(note.excerpt || description).replace(/</g, "&lt;")}</p>`;
    } else {
      noscriptContent = `
  <h1>${vaultName}</h1>
  <p>${notes?.length || 0} notes</p>`;
    }
    if (notes?.length) {
      noscriptContent += "\n  <h2>Notes</h2>\n  <ul>";
      for (const n of notes.slice(0, 30)) {
        noscriptContent += `<li><a href="/note/${n.slug}">${n.title.replace(/</g, "&lt;")}</a></li>`;
      }
      noscriptContent += "</ul>";
    }
    htmlPage = htmlPage.replace(
      "</body>",
      `<noscript>${noscriptContent}</noscript>\n</body>`
    );

    // Inject preloaded state + meta tags + JSON-LD into <head>
    const jsonLd = {
      "@context": "https://schema.org",
      "@type": "WebPage",
      name: title,
      description: description,
      url: `${BASE_URL}${url.split("#")[0]}`,
    };

    const breadcrumbItems = [
      {
        "@type": "ListItem",
        position: 1,
        name: "Home",
        item: BASE_URL,
      },
    ];
    if (route.type === "note" && note) {
      breadcrumbItems.push({
        "@type": "ListItem",
        position: 2,
        name: note.title,
        item: `${BASE_URL}/note/${route.slug}`,
      });
    }
    const breadcrumbLd = {
      "@context": "https://schema.org",
      "@type": "BreadcrumbList",
      itemListElement: breadcrumbItems,
    };

    htmlPage = htmlPage.replace(
      "</head>",
      `<script>window.__PRELOADED_STATE__=${serializedPreloadedState};</script>
  <meta name="description" content="${description.replace(/"/g, "&quot;")}" />
  <meta property="og:title" content="${title.replace(/"/g, "&quot;")}" />
  <meta property="og:description" content="${description.replace(/"/g, "&quot;")}" />
  <link rel="canonical" href="${BASE_URL}${url.split("#")[0]}" />
  <script type="application/ld+json">${JSON.stringify(jsonLd)}</script>
  <script type="application/ld+json">${JSON.stringify(breadcrumbLd)}</script>
  </head>`
    );

    // Update the page title
    htmlPage = htmlPage.replace(
      /<title>.*?<\/title>/,
      `<title>${title}</title>`
    );

    res.type("html").send(htmlPage);
  } catch (err) {
    console.error("SSR render error:", err);
    res.status(500).send("SSR render error");
  }
});

app.listen(PORT, () => {
  console.log(`SSR sidecar listening on :${PORT}`);
  console.log(`  API base: ${API_BASE}`);
  console.log(`  Base URL: ${BASE_URL}`);
});
