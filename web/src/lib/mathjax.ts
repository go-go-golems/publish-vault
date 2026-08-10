/**
 * Lazily-initialised MathJax (TeX → SVG) singleton.
 *
 * Uses MathJax's direct API rather than its global component bundle. The Go
 * parser has already marked every element that holds TeX (`.math`), so
 * document scanning is pure waste — and disabling it (inlineMath/displayMath
 * set to []) removes a whole class of "MathJax found math where we didn't want
 * it" bugs. We only ever call `convert()` on a string we chose.
 *
 * Everything in this module is browser-only. It is reached exclusively through
 * a dynamic import() from an effect, and vite.config.ts aliases the module to
 * mathjax.server.ts for the SSR build, so it never enters the Node graph.
 *
 * SVG output rather than CHTML: SVG compiles its glyph outlines into the JS
 * chunk, so there are no .woff files to plumb through pkg/web/embed and no
 * flash of unstyled math while fonts load. The trade is a larger chunk and
 * math that is not selectable as text.
 */
import type { MathDocument } from "@mathjax/src/js/core/MathDocument.js";

/**
 * TeX packages, imported for side effects: each module calls
 * Configuration.create(<name>) at module scope, which registers it in
 * MathJax's ConfigurationHandler. A name in `packages` below that has no
 * corresponding import here silently does nothing.
 *
 * MathJax 4 has no AllPackages export (that was a v3 API), and pulling in all
 * ~60 of them would roughly double the chunk. This list covers what actually
 * shows up in a notes vault; extend it when a note needs more.
 */
const TEX_PACKAGES = [
  "base",
  "ams",
  "newcommand",
  "configmacros",
  "noundefined",
  "boldsymbol",
  "braket",
  "cancel",
  "cases",
  "color",
  "mathtools",
  "physics",
  "textmacros",
  "unicode",
  "verb",
];

/**
 * MathJax 4 splits its fonts into ~40 glyph-range files and pulls them in on
 * demand through `mathjax.asyncLoad` — `\mathbb` needs "double-struck",
 * `\mathfrak` needs "fraktur", and so on. In a `<script>`-tag deployment its
 * own loader resolves those; in a bundler there is no loader, and the first
 * `\mathbb{E}` fails with:
 *
 *   Can't load '@mathjax/mathjax-newcm-font/js/svg/dynamic/double-struck.js':
 *   No mathjax.asyncLoad method specified
 *
 * So we supply the loader ourselves. Each dynamic file calls
 * `MathJaxNewcmFont.dynamicSetup(...)` at module scope, which means a plain
 * side-effect import is enough — the module's exports are never read.
 *
 * The imports are written out rather than globbed for the same reason
 * highlightLanguages.ts writes its language map out: Vite emits one browser
 * chunk per static-analysable dynamic import, so a note using `\mathbb` fetches
 * exactly one ~100 kB range file and nothing else. All 40 eagerly would be 12 MB.
 */
const FONT_RANGES: Record<string, () => Promise<unknown>> = {
  accents: () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/accents.js"),
  "accents-b-i": () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/accents-b-i.js"),
  arabic: () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/arabic.js"),
  arrows: () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/arrows.js"),
  braille: () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/braille.js"),
  "braille-d": () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/braille-d.js"),
  calligraphic: () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/calligraphic.js"),
  cherokee: () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/cherokee.js"),
  cyrillic: () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/cyrillic.js"),
  "cyrillic-ss": () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/cyrillic-ss.js"),
  devanagari: () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/devanagari.js"),
  "double-struck": () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/double-struck.js"),
  fraktur: () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/fraktur.js"),
  greek: () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/greek.js"),
  "greek-ss": () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/greek-ss.js"),
  hebrew: () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/hebrew.js"),
  latin: () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/latin.js"),
  "latin-b": () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/latin-b.js"),
  "latin-bi": () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/latin-bi.js"),
  "latin-i": () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/latin-i.js"),
  marrows: () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/marrows.js"),
  math: () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/math.js"),
  monospace: () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/monospace.js"),
  "monospace-ex": () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/monospace-ex.js"),
  "monospace-l": () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/monospace-l.js"),
  mshapes: () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/mshapes.js"),
  phonetics: () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/phonetics.js"),
  "phonetics-ss": () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/phonetics-ss.js"),
  PUA: () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/PUA.js"),
  "sans-serif": () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/sans-serif.js"),
  "sans-serif-b": () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/sans-serif-b.js"),
  "sans-serif-bi": () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/sans-serif-bi.js"),
  "sans-serif-ex": () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/sans-serif-ex.js"),
  "sans-serif-i": () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/sans-serif-i.js"),
  "sans-serif-r": () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/sans-serif-r.js"),
  script: () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/script.js"),
  shapes: () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/shapes.js"),
  symbols: () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/symbols.js"),
  "symbols-b-i": () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/symbols-b-i.js"),
  variants: () => import("@mathjax/mathjax-newcm-font/js/svg/dynamic/variants.js"),
};

/** Resolve a font-range request like ".../dynamic/double-struck.js" to a loader. */
function loadFontRange(file: string): Promise<unknown> {
  const range = file.split("/").pop()?.replace(/\.js$/, "") ?? "";
  const loader = FONT_RANGES[range];
  if (!loader) {
    return Promise.reject(new Error(`unknown MathJax font range: ${range}`));
  }
  return loader();
}

type Doc = MathDocument<HTMLElement, Text, Document>;

let docPromise: Promise<Doc> | null = null;

async function getDocument(): Promise<Doc> {
  if (docPromise) return docPromise;

  docPromise = (async () => {
    const [{ mathjax }, { TeX }, { SVG }, { browserAdaptor }, { RegisterHTMLHandler }] =
      await Promise.all([
        import("@mathjax/src/js/mathjax.js"),
        import("@mathjax/src/js/input/tex.js"),
        import("@mathjax/src/js/output/svg.js"),
        import("@mathjax/src/js/adaptors/browserAdaptor.js"),
        import("@mathjax/src/js/handlers/html.js"),
        import("@mathjax/src/js/input/tex/base/BaseConfiguration.js"),
        import("@mathjax/src/js/input/tex/ams/AmsConfiguration.js"),
        import("@mathjax/src/js/input/tex/newcommand/NewcommandConfiguration.js"),
        import("@mathjax/src/js/input/tex/configmacros/ConfigMacrosConfiguration.js"),
        import("@mathjax/src/js/input/tex/noundefined/NoUndefinedConfiguration.js"),
        import("@mathjax/src/js/input/tex/boldsymbol/BoldsymbolConfiguration.js"),
        import("@mathjax/src/js/input/tex/braket/BraketConfiguration.js"),
        import("@mathjax/src/js/input/tex/cancel/CancelConfiguration.js"),
        import("@mathjax/src/js/input/tex/cases/CasesConfiguration.js"),
        import("@mathjax/src/js/input/tex/color/ColorConfiguration.js"),
        import("@mathjax/src/js/input/tex/mathtools/MathtoolsConfiguration.js"),
        import("@mathjax/src/js/input/tex/physics/PhysicsConfiguration.js"),
        import("@mathjax/src/js/input/tex/textmacros/TextMacrosConfiguration.js"),
        import("@mathjax/src/js/input/tex/unicode/UnicodeConfiguration.js"),
        import("@mathjax/src/js/input/tex/verb/VerbConfiguration.js"),
      ]);

    RegisterHTMLHandler(browserAdaptor());
    mathjax.asyncLoad = loadFontRange;

    return mathjax.document(document, {
      // Delimiter scanning off: we always call convert() explicitly.
      InputJax: new TeX({ packages: TEX_PACKAGES, inlineMath: [], displayMath: [] }),
      OutputJax: new SVG({
        // "local" keeps each formula's glyph <defs> inside its own <svg>. The
        // default global cache emits <use> references into a shared <defs>
        // block, which dangle the moment a formula node is cloned into the
        // lightbox or its container is rewritten by dangerouslySetInnerHTML.
        fontCache: "local",
        // MathJax 4 breaks inline math to fit its container, and measures that
        // container from the DOM. convert() builds a detached node — there is
        // nothing to measure — so the width comes back ~0 and every formula
        // breaks at every operator ("e^{i\pi}" / "+ 1" / "= 0" on three
        // lines). We insert the node ourselves and let CSS handle overflow, so
        // MathJax's own inline breaking is both wrong here and unnecessary.
        linebreaks: { inline: false },
      }),
    }) as Doc;
  })();

  return docPromise;
}

export interface TypesetResult {
  /** The rendered node, or null when the TeX could not be typeset. */
  node: HTMLElement | null;
  error?: string;
}

/**
 * Convert a single TeX string to a DOM node. Never throws — a TeX syntax error
 * must leave the surrounding note intact, with the source still readable.
 */
export async function typesetTeX(
  tex: string,
  display: boolean
): Promise<TypesetResult> {
  try {
    const doc = await getDocument();
    const { mathjax } = await import("@mathjax/src/js/mathjax.js");
    // convert() is synchronous, so when it needs a glyph range that is not
    // resident it throws a retry signal carrying the load promise.
    // handleRetriesFor awaits that promise and re-runs the conversion; calling
    // convert() bare instead surfaces as "dynamic file 'x' failed to load".
    const node = (await mathjax.handleRetriesFor(() =>
      doc.convert(tex, { display })
    )) as unknown as HTMLElement;
    return { node };
  } catch (err) {
    return { node: null, error: err instanceof Error ? err.message : String(err) };
  }
}

/**
 * Inject the stylesheet the SVG output jax needs (container display rules,
 * assistive-MathML hiding). With the direct API nothing does this for us.
 * Idempotent, and guarded on the DOM as well as on module state so a dev-mode
 * hot reload does not stack duplicate <style> elements.
 */
export async function ensureMathStyles(): Promise<void> {
  const STYLE_ID = "mathjax-styles";
  if (document.getElementById(STYLE_ID)) return;

  const doc = await getDocument();
  const output = doc.outputJax as unknown as {
    styleSheet: (d: Doc) => unknown;
    adaptor: { textContent: (node: unknown) => string };
  };
  const css = output.adaptor.textContent(output.styleSheet(doc));

  if (document.getElementById(STYLE_ID)) return;
  const style = document.createElement("style");
  style.id = STYLE_ID;
  style.textContent = css;
  document.head.appendChild(style);
}
