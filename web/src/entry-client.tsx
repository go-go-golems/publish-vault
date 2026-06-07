// entry-client.tsx — Client-side entry point.
//
// Uses createRoot() to replace the server-rendered HTML with the full
// interactive React app. The SSR HTML serves as a temporary preview
// for crawlers and agents that don't execute JavaScript.
//
// We don't use hydrateRoot() because the SSR components (SSRNotePage,
// SSRHomePage, SSRSearchPage) are simplified versions that don't match
// the full client component tree (VaultLayout + Wouter routing).
// Using hydrateRoot with mismatched DOM causes React error #418.

import React from "react";
import { createRoot } from "react-dom/client";
import { Provider } from "react-redux";
import { makeStore } from "./store/store";
import App from "./App";
import "./index.css";

declare global {
  interface Window {
    __PRELOADED_STATE__?: Record<string, unknown>;
  }
}

// Read preloaded state from SSR, then delete it so it's not lingering
const preloadedState = window.__PRELOADED_STATE__;
delete window.__PRELOADED_STATE__;

const store = makeStore(preloadedState);

const root = document.getElementById("root")!;
// Clear the SSR-rendered content before mounting the client app.
// This avoids hydration mismatches since our SSR components are
// simplified versions that don't match the full client component tree.
root.textContent = "";

createRoot(root).render(
  <React.StrictMode>
    <Provider store={store}>
      <App />
    </Provider>
  </React.StrictMode>
);
