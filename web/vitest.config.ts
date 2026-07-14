import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import path from "node:path";

const WEB_ROOT = import.meta.dirname;

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(WEB_ROOT, "src"),
      "@highlight-languages": path.resolve(
        WEB_ROOT,
        "src/lib/highlightLanguages.server.ts"
      ),
    },
  },
  test: {
    // Use 'node' environment for SSR tests (renderToString doesn't need a DOM).
    // If DOM-dependent tests are added later, use jsdom per-test or per-file.
    environment: "node",
  },
});
