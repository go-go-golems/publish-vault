import { configureStore } from "@reduxjs/toolkit";
import { vaultApi } from "./vaultApi";
import uiReducer from "./uiSlice";

export const store = configureStore({
  reducer: {
    [vaultApi.reducerPath]: vaultApi.reducer,
    ui: uiReducer,
  },
  middleware: (getDefaultMiddleware) =>
    getDefaultMiddleware().concat(vaultApi.middleware),
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
