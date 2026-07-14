// SSR replacement for the browser-only highlight.js language loader.
// NoteRenderer performs highlighting in useEffect, so this module is never
// called during render. Keeping a server stub out of the SSR graph avoids
// bundling browser language chunks into the Node sidecar.
export async function highlightCodeBlocks(_root: HTMLElement): Promise<void> {
  return;
}
