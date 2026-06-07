// entry-client.tsx — Client-side entry point.
//
// Hydrates the server-rendered React tree with the same app/routes that the
// SSR sidecar rendered. The server serializes the RTK Query cache into
// window.__PRELOADED_STATE__; the browser restores it before hydration so the
// first client render matches the server render.

import React from "react";
import { hydrateRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { Provider } from "react-redux";
import { makeStore } from "./store/store";
import { AppRoutes } from "./App";
import "./index.css";

declare global {
  interface Window {
    __PRELOADED_STATE__?: Record<string, unknown>;
  }
}

// Read preloaded state from SSR, then delete it so it is not lingering on
// window after the initial client store has been created.
const preloadedState = window.__PRELOADED_STATE__;
delete window.__PRELOADED_STATE__;

const store = makeStore(preloadedState);
const root = document.getElementById("root")!;

hydrateRoot(
  root,
  <React.StrictMode>
    <Provider store={store}>
      <BrowserRouter>
        <AppRoutes />
      </BrowserRouter>
    </Provider>
  </React.StrictMode>
);
