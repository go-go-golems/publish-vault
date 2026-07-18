// entry-client.tsx — Client-side entry point.
//
// Hydrates the server-rendered React tree with the same app/routes that the
// SSR sidecar rendered. The server serializes the RTK Query cache into
// window.__PRELOADED_STATE__; the browser restores it before hydration so the
// first client render matches the server render.

import React, { lazy, type ComponentType } from "react";
import { createRoot, hydrateRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { Provider } from "react-redux";
import { makeStore } from "./store/store";
import { AppRoutes } from "./App";
import type { NotePageProps } from "./components/pages/NotePage/NotePage";

let notePageModulePromise:
  | Promise<typeof import("./components/pages/NotePage/NotePage")>
  | undefined;

function loadNotePage() {
  return (notePageModulePromise ??= import(
    "./components/pages/NotePage/NotePage"
  ));
}

const LazyNotePage = lazy(async () => {
  const module = await loadNotePage();
  return { default: module.NotePage };
});
import "./index.css";

declare global {
  interface Window {
    __PRELOADED_STATE__?: Record<string, unknown>;
    __HOME_SLUG__?: string | null;
  }
}

// Read preloaded state from SSR, then delete it so it is not lingering on
// window after the initial client store has been created.
const preloadedState = window.__PRELOADED_STATE__;
const initialHomeSlug = window.__HOME_SLUG__;
delete window.__PRELOADED_STATE__;
delete window.__HOME_SLUG__;

const store = makeStore(preloadedState);
const root = document.getElementById("root")!;

async function hydrate() {
  // The server renders the home route as a note, so resolve that route chunk
  // before hydration. Search and other routes stay fully lazy on first load.
  const pathname = window.location.pathname;
  let NotePageComponent: ComponentType<NotePageProps> = LazyNotePage;
  if (pathname === "/" || pathname.startsWith("/note/")) {
    // Use the resolved component itself for the initial tree. This preserves
    // React's useId path across SSR and hydration; subsequent navigations use
    // the lazy wrapper without a hydration constraint.
    NotePageComponent = (await loadNotePage()).NotePage;
  }

  const app = (
    <React.StrictMode>
      <Provider store={store}>
        <BrowserRouter>
          <AppRoutes
            NotePageComponent={NotePageComponent}
            initialHomeSlug={initialHomeSlug ?? undefined}
            suspendNotePage={pathname !== "/" && !pathname.startsWith("/note/")}
          />
        </BrowserRouter>
      </Provider>
    </React.StrictMode>
  );

  // Hydrate only when the server actually rendered into #root (SSR sidecar
  // mode). Without SSR the container is empty; hydrating it makes React 19
  // report a mismatch (#418) and run a recovery render that can replace DOM
  // nodes AFTER the note-enhancement effects mutated them — which is how
  // heading anchors/copy buttons intermittently vanished (hash navigation
  // made the bad interleaving reliable). createRoot renders exactly once.
  if (root.hasChildNodes()) {
    hydrateRoot(root, app);
  } else {
    createRoot(root).render(app);
  }
}

void hydrate();
