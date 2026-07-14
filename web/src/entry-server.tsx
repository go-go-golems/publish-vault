// entry-server.tsx — SSR entry point for the Node.js sidecar.
//
// The SSR server (server.mjs) calls renderApp() after pre-fetching data from
// the Go API. The pre-fetched data is inserted into RTK Query's cache before
// renderToString(), so the real application tree renders with vault data on
// the server. The browser hydrates the same tree in entry-client.tsx.

import React from "react";
import { renderToString } from "react-dom/server";
import { StaticRouter } from "react-router";
import { Provider } from "react-redux";
import { makeStore } from "./store/store";
import { vaultApi } from "./store/vaultApi";
import { AppRoutes } from "./App";
import { NotePage } from "./components/pages/NotePage/NotePage";
import type { SiteConfig, NoteListItem, Note, FileNode } from "./types/index";

export interface SSRData {
  config?: SiteConfig | null;
  notes?: NoteListItem[] | null;
  tree?: FileNode | null;
  note?: Note | null;
  homeSlug?: string | null;
}

export interface SSRResult {
  html: string;
  preloadedState: unknown;
}

/**
 * Extract the note slug from a page URL for RTK Query cache key preloading.
 * React Router owns page matching; this helper only mirrors the current
 * sidecar prefetch contract so useGetNoteQuery(slug) sees the seeded cache.
 */
export function extractNoteSlug(url: string): string | undefined {
  const pathname = url.split("#")[0]?.split("?")[0] || "/";
  if (!pathname.startsWith("/note/")) return undefined;
  const raw = pathname.replace(/^\/note\//, "");
  return raw ? decodeURIComponent(raw) : undefined;
}

/**
 * Preload RTK Query cache with server-fetched data so components render with
 * real data during renderToString(). The resulting store state is serialized
 * by server.mjs and restored by entry-client.tsx before hydrateRoot().
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

/**
 * Render the real React app to an HTML string for the given URL. The same
 * AppRoutes component is used on the client, where BrowserRouter replaces the
 * server-only StaticRouter and hydrateRoot attaches to this markup.
 */
export async function renderApp(
  url: string,
  data: SSRData
): Promise<SSRResult> {
  const store = makeStore();
  const pathname = url.split("#")[0]?.split("?")[0] || "/";
  const slug =
    extractNoteSlug(url) ?? (pathname === "/" ? data.note?.slug : undefined);

  await preloadCache(store, data, slug);

  const html = renderToString(
    <Provider store={store}>
      <StaticRouter location={url}>
        <AppRoutes
          NotePageComponent={NotePage}
          initialHomeSlug={data.homeSlug ?? undefined}
        />
      </StaticRouter>
    </Provider>
  );

  return { html, preloadedState: store.getState() };
}
