/**
 * Post-render DOM enhancement pipeline for note content.
 *
 * The server delivers pre-rendered HTML; these functions progressively
 * enhance it inside the .note-prose container after hydration. Each
 * enhancement is idempotent (safe to re-run on the same root) and takes the
 * container element plus explicit dependencies, so it can be unit-tested
 * without React.
 *
 * Ordering constraint: mermaid must run before syntax highlighting so that
 * `language-mermaid` code blocks are replaced by SVG containers before hljs
 * walks the remaining <pre> blocks.
 */
import { nanoid } from "nanoid";
import { highlightCodeBlocks } from "@highlight-languages";

let mermaidInitialized = false;

/**
 * Replace `code.language-mermaid` blocks with rendered SVG containers.
 * Returns a cancel function; after cancellation, in-flight renders stop
 * mutating the DOM.
 */
export function enhanceMermaid(root: HTMLElement): () => void {
  const blocks = root.querySelectorAll<HTMLElement>("code.language-mermaid");
  if (blocks.length === 0) return () => {};

  let cancelled = false;

  const render = async () => {
    const { default: mermaid } = await import("mermaid");
    if (cancelled) return;

    if (!mermaidInitialized) {
      mermaid.initialize({
        startOnLoad: false,
        theme: "base",
        themeVariables: {
          primaryColor: "#1a1a1a",
          primaryTextColor: "#faf8f4",
          primaryBorderColor: "#1a1a1a",
          lineColor: "#555",
          secondaryColor: "#f0ede8",
          tertiaryColor: "#faf8f4",
          fontSize: "12px",
        },
      });
      mermaidInitialized = true;
    }

    await Promise.all(
      Array.from(blocks).map(async block => {
        const pre = block.parentElement;
        if (!pre || pre.tagName !== "PRE") return;
        const src = block.textContent ?? "";
        const id = `mermaid-${nanoid(6)}`;

        try {
          const { svg } = await mermaid.render(id, src);
          if (cancelled || !pre.isConnected) return;
          const container = document.createElement("div");
          container.className = "mermaid-svg retro-inset my-2 overflow-x-auto";
          container.innerHTML = svg;
          pre.replaceWith(container);
        } catch {
          // Leave raw <pre> as fallback
        }
      })
    );
  };

  void render();
  return () => {
    cancelled = true;
  };
}

/**
 * Syntax-highlight code blocks and attach copy-to-clipboard buttons.
 * Returns a cancel function for the async highlight pass.
 */
export function enhanceCodeBlocks(root: HTMLElement): () => void {
  let cancelled = false;

  const run = async () => {
    await highlightCodeBlocks(root);
    if (cancelled) return;
    addCopyButtons(root);
  };

  void run();
  return () => {
    cancelled = true;
  };
}

/** Attach a copy button to every <pre> that does not already have one. */
export function addCopyButtons(root: HTMLElement): void {
  const pres = root.querySelectorAll<HTMLElement>("pre");
  pres.forEach(pre => {
    if (pre.querySelector(".copy-code-btn")) return;
    const btn = document.createElement("button");
    btn.className = "copy-code-btn";
    btn.title = "Copy code";
    btn.textContent = "⎘";
    btn.addEventListener("click", () => {
      const code = pre.querySelector("code");
      if (!code) return;
      navigator.clipboard.writeText(code.textContent ?? "").then(() => {
        btn.textContent = "✓";
        setTimeout(() => {
          btn.textContent = "⎘";
        }, 1500);
      });
    });
    pre.style.position = "relative";
    pre.appendChild(btn);
  });
}

/**
 * Inject a `#` permalink anchor into each heading that has an id.
 * Idempotent: headings that already carry an anchor are skipped.
 */
export function enhanceHeadingAnchors(root: HTMLElement): void {
  const headings = root.querySelectorAll<HTMLElement>("h1, h2, h3, h4, h5, h6");
  headings.forEach(heading => {
    if (heading.querySelector(".heading-anchor")) return;
    const id = heading.id;
    if (!id) return;
    const anchor = document.createElement("a");
    anchor.className = "heading-anchor";
    anchor.href = `#${id}`;
    anchor.title = "Link to this section";
    anchor.textContent = "#";
    anchor.addEventListener("click", e => {
      e.preventDefault();
      window.location.hash = id;
      heading.scrollIntoView({ behavior: "smooth", block: "start" });
    });
    heading.appendChild(anchor);
  });
}

/**
 * Resolve `.wiki-embed` placeholders (![[Note]]) by fetching the target
 * note's rendered HTML through the supplied loader. The loader is injected
 * so the component layer can route it through RTK Query (and the static
 * vault in VITE_STATIC_VAULT builds) instead of raw fetch().
 */
export function resolveEmbeds(
  root: HTMLElement,
  loadNoteHtml: (slug: string) => Promise<string | null>
): void {
  const embeds = root.querySelectorAll<HTMLElement>(".wiki-embed");
  embeds.forEach(embed => {
    const target = embed.getAttribute("data-target") ?? "";
    if (!target) return;
    // Don't re-render already-populated embeds
    if (embed.dataset.resolved) return;
    embed.dataset.resolved = "true";

    loadNoteHtml(target)
      .then(html => {
        if (!html) throw new Error("embed target has no html");
        const container = document.createElement("div");
        container.className = "wiki-embed-content retro-inset my-2";
        container.innerHTML = html;
        embed.appendChild(container);
      })
      .catch(() => {
        // Show broken link indicator
        embed.textContent = `⚠ Embed not found: ${target}`;
        embed.className = "wiki-embed wiki-embed-broken";
      });
  });
}
