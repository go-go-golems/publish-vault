// SSR replacement for the browser-only MathJax loader.
//
// enhanceMath runs from a useEffect, which renderToString never invokes, so
// this module is never called during server rendering. Keeping the real one
// out of the SSR graph avoids bundling MathJax's TeX engine and glyph tables
// into the Node sidecar — the same arrangement highlightLanguages.server.ts
// has for highlight.js.
export interface TypesetResult {
  node: HTMLElement | null;
  error?: string;
}

export async function typesetTeX(
  _tex: string,
  _display: boolean
): Promise<TypesetResult> {
  return { node: null, error: "MathJax is not available during SSR" };
}

export async function ensureMathStyles(): Promise<void> {
  return;
}
