// entry-server.tsx — SSR entry point for the Node.js sidecar.
//
// The SSR server (server.mjs) calls renderApp() after pre-fetching data
// from the Go API. The pre-fetched data is inserted into RTK Query's
// cache before renderToString(), so React components render with real
// vault data on the server.
//
// Wouter doesn't support server-side rendering (no StaticRouter), so we
// parse the URL and render the matching page content directly. The server-
// rendered HTML is a simplified version focused on content (note body,
// title, backlinks) for SEO and agent readability. The client hydrates
// the full interactive app on top.

import React from "react";
import { renderToString } from "react-dom/server";
import { Provider } from "react-redux";
import { makeStore } from "./store/store";
import { vaultApi } from "./store/vaultApi";
import type {
  SiteConfig,
  NoteListItem,
  Note,
  FileNode,
} from "./types/index";

export interface SSRData {
  config?: SiteConfig | null;
  notes?: NoteListItem[] | null;
  tree?: FileNode | null;
  note?: Note | null;
}

export interface SSRResult {
  html: string;
  preloadedState: unknown;
}

/**
 * Preload RTK Query cache with server-fetched data so components render
 * with real data during renderToString().
 */
async function preloadCache(
  store: ReturnType<typeof makeStore>,
  data: SSRData,
  slug?: string
) {
  const actions: Array<Promise<unknown>> = [];

  if (data.config) {
    actions.push(
      store.dispatch(
        vaultApi.util.upsertQueryData("getConfig", undefined, data.config)
      ) as unknown as Promise<unknown>
    );
  }

  if (data.notes) {
    actions.push(
      store.dispatch(
        vaultApi.util.upsertQueryData("listNotes", undefined, data.notes)
      ) as unknown as Promise<unknown>
    );
  }

  if (data.tree) {
    actions.push(
      store.dispatch(
        vaultApi.util.upsertQueryData("getTree", undefined, data.tree)
      ) as unknown as Promise<unknown>
    );
  }

  if (data.note && slug) {
    actions.push(
      store.dispatch(
        vaultApi.util.upsertQueryData("getNote", slug, data.note)
      ) as unknown as Promise<unknown>
    );
  }

  await Promise.all(actions);
}

// ---------------------------------------------------------------------------
// SSR-safe page components
// ---------------------------------------------------------------------------

/**
 * SSR note page: renders the note title and HTML body without interactive
 * features (no sidebar, no resizable panels, no Wouter navigation).
 * This produces the SEO-visible content that crawlers and agents see.
 */
function SSRNotePage({ note }: { note: Note }) {
  return React.createElement("div", { className: "ssr-note" }, [
    React.createElement("h1", { key: "title", className: "text-xl font-bold" }, note.title),
    note.tags?.length > 0
      ? React.createElement(
          "div",
          { key: "tags", className: "flex gap-1 mt-2 mb-4 flex-wrap" },
          note.tags.map((tag) =>
            React.createElement(
              "span",
              { key: tag, className: "text-xs px-1 border border-current" },
              tag
            )
          )
        )
      : null,
    React.createElement("div", {
      key: "body",
      className: "note-prose",
      dangerouslySetInnerHTML: { __html: note.html },
    }),
    note.backlinks?.length > 0
      ? React.createElement("div", { key: "backlinks", className: "mt-8 pt-4 border-t" }, [
          React.createElement(
            "h2",
            { className: "text-sm font-bold uppercase tracking-wider mb-2" },
            `Linked Mentions (${note.backlinks.length})`
          ),
          React.createElement(
            "ul",
            { className: "list-disc pl-4" },
            note.backlinks.map((slug) =>
              React.createElement("li", { key: slug }, [
                React.createElement(
                  "a",
                  { href: `/note/${slug}`, className: "wiki-link" },
                  slug
                ),
              ])
            )
          ),
        ])
      : null,
  ]);
}

/**
 * SSR home page: renders a list of available notes.
 */
function SSRHomePage({
  notes,
  config,
}: {
  notes: NoteListItem[];
  config: SiteConfig;
}) {
  return React.createElement("div", { className: "ssr-home p-6" }, [
    React.createElement("h1", { key: "title" }, config.pageTitle || config.vaultName),
    React.createElement(
      "p",
      { key: "count", className: "text-sm text-gray-500 mb-4" },
      `${notes.length} notes`
    ),
    React.createElement(
      "ul",
      { key: "list" },
      notes.slice(0, 50).map((note) =>
        React.createElement("li", { key: note.slug }, [
          React.createElement(
            "a",
            { href: `/note/${note.slug}` },
            note.title
          ),
          note.excerpt
            ? React.createElement(
                "p",
                { className: "text-sm text-gray-500" },
                note.excerpt.slice(0, 120)
              )
            : null,
        ])
      )
    ),
  ]);
}

/**
 * SSR search page: renders a placeholder — search is client-side.
 */
function SSRSearchPage({ config }: { config: SiteConfig }) {
  return React.createElement("div", { className: "ssr-search p-6" }, [
    React.createElement("h1", { key: "title" }, `Search — ${config.vaultName}`),
    React.createElement(
      "p",
      { key: "desc", className: "text-sm text-gray-500" },
      "Search requires JavaScript. Use the sidebar to browse notes."
    ),
  ]);
}

// ---------------------------------------------------------------------------
// URL parsing
// ---------------------------------------------------------------------------

interface ParsedRoute {
  type: "home" | "note" | "search" | "unknown";
  slug?: string;
}

function parseRoute(url: string): ParsedRoute {
  const pathname = url.split("#")[0]?.split("?")[0] || "/";
  if (pathname === "/search") return { type: "search" };
  if (pathname.startsWith("/note/")) {
    const slug = pathname.replace(/^\/note\//, "");
    return { type: "note", slug: decodeURIComponent(slug) };
  }
  if (pathname === "/") return { type: "home" };
  return { type: "unknown" };
}

// ---------------------------------------------------------------------------
// Main render function
// ---------------------------------------------------------------------------

/**
 * Render the React app to an HTML string for the given URL.
 * Called by the Node.js SSR sidecar (server.mjs).
 */
export async function renderApp(
  url: string,
  data: SSRData
): Promise<SSRResult> {
  const store = makeStore();
  const route = parseRoute(url);
  const slug = route.type === "note" ? route.slug : undefined;

  await preloadCache(store, data, slug);

  // Render the matching page content
  let content: React.ReactElement;

  switch (route.type) {
    case "note":
      if (data.note) {
        content = React.createElement(SSRNotePage, { note: data.note });
      } else {
        content = React.createElement("div", null, "Note not found");
      }
      break;

    case "search":
      content = React.createElement(SSRSearchPage, {
        config: data.config || { vaultName: "Vault", pageTitle: "Vault", notes: 0 },
      });
      break;

    case "home":
    default:
      content = React.createElement(SSRHomePage, {
        notes: data.notes || [],
        config: data.config || { vaultName: "Vault", pageTitle: "Vault", notes: 0 },
      });
      break;
  }

  const html = renderToString(
    React.createElement(Provider, { store }, content)
  );

  return { html, preloadedState: store.getState() };
}
