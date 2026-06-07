import { configureStore } from "@reduxjs/toolkit";
import { vaultApi } from "./vaultApi";
import uiReducer from "./uiSlice";

// Store factory for SSR — each server request gets its own store.
// The browser uses the exported singleton below.
export function makeStore(preloadedState?: unknown) {
  return configureStore({
    reducer: {
      [vaultApi.reducerPath]: vaultApi.reducer,
      ui: uiReducer,
    },
    middleware: (getDefaultMiddleware) =>
      getDefaultMiddleware().concat(vaultApi.middleware),
    ...(preloadedState ? { preloadedState } : {}),
  });
}

// Browser/dev singleton. SSR must call makeStore() per request instead.
export const store = makeStore();

export type AppStore = ReturnType<typeof makeStore>;
export type RootState = ReturnType<AppStore["getState"]>;
export type AppDispatch = AppStore["dispatch"];
