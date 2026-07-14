/**
 * ORGANISM: NoteRenderer
 * Design: Retro System 1 — renders server-side HTML with wiki-link resolution,
 * syntax highlighting, and mermaid diagram rendering.
 * Handles click events on wiki-links for SPA navigation.
 */
import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { clsx } from "clsx";
import { nanoid } from "nanoid";
import { highlightCodeBlocks } from "../../../lib/highlightLanguages";
import { resolveWikiLinks, buildSlugSet } from "../../../lib/wikiLinks";
import { FrontmatterPanel } from "../../molecules/FrontmatterPanel/FrontmatterPanel";
import { BreadcrumbBar } from "../../molecules/BreadcrumbBar/BreadcrumbBar";
import { LightboxModal } from "../../atoms/LightboxModal/LightboxModal";
import type { Note } from "../../../types";

/** Lightbox content descriptor */
interface LightboxState {
  type: "image" | "mermaid";
  src?: string;
  alt?: string;
  svgHtml?: string;
}

export interface NoteRendererProps {
  note: Note;
  allSlugs: string[];
  onNavigate: (slug: string) => void;
  onTagClick?: (tag: string) => void;
  className?: string;
}

let mermaidInitialized = false;

export const NoteRenderer: React.FC<NoteRendererProps> = ({
  note,
  allSlugs,
  onNavigate,
  onTagClick,
  className,
}) => {
  const contentRef = useRef<HTMLDivElement>(null);
  const [copied, setCopied] = useState(false);

  // Lightbox state for full-screen image/mermaid viewing
  const [lightbox, setLightbox] = useState<LightboxState | null>(null);

  // Build slug set for broken-link detection
  const slugSet = useMemo(() => buildSlugSet(allSlugs), [allSlugs]);

  // Resolve wiki links in HTML
  const resolvedHtml = useMemo(
    () => resolveWikiLinks(note.html, slugSet),
    [note.html, slugSet]
  );

  // Intercept wiki-link clicks for SPA navigation
  const handleClick = useCallback(
    (e: MouseEvent) => {
      const target = e.target as HTMLElement;

      // Collapsible callout toggle
      const titleEl = target.closest(".callout-collapsible .callout-title");
      if (titleEl) {
        const callout = titleEl.closest(".callout");
        const body = callout?.querySelector(".callout-body");
        const toggle = titleEl.querySelector(".callout-toggle");
        if (body instanceof HTMLElement) {
          const isHidden = body.style.display === "none";
          body.style.display = isHidden ? "" : "none";
          if (toggle) toggle.textContent = isHidden ? "\u25BC" : "\u25B6";
        }
        return;
      }

      // Image click → open lightbox
      const img = target.closest("img");
      if (img && contentRef.current?.contains(img)) {
        e.preventDefault();
        setLightbox({
          type: "image",
          src: img.src,
          alt: img.alt || undefined,
        });
        return;
      }

      // Mermaid diagram click → open lightbox
      const mermaidEl = target.closest(".mermaid-svg");
      if (mermaidEl && contentRef.current?.contains(mermaidEl)) {
        e.preventDefault();
        setLightbox({
          type: "mermaid",
          svgHtml: mermaidEl.innerHTML,
        });
        return;
      }

      const anchor = target.closest("a");
      if (!anchor) return;

      const wikiTarget = anchor.getAttribute("data-target");
      if (wikiTarget && anchor.classList.contains("wiki-link")) {
        e.preventDefault();
        if (!anchor.classList.contains("broken")) {
          const href = anchor.getAttribute("href") ?? "";
          const hashIndex = href.indexOf("#");
          const hash = hashIndex >= 0 ? href.slice(hashIndex) : "";
          onNavigate(`${wikiTarget}${hash}`);
        }
        return;
      }

      // Internal hash links — let browser handle
      const href = anchor.getAttribute("href");
      if (href?.startsWith("#")) return;

      // Internal /note/ links
      if (href?.startsWith("/note/")) {
        e.preventDefault();
        const slug = href.replace("/note/", "");
        onNavigate(slug);
      }
    },
    [onNavigate]
  );

  // Click handler setup
  useEffect(() => {
    const el = contentRef.current;
    if (!el) return;
    el.addEventListener("click", handleClick);
    return () => el.removeEventListener("click", handleClick);
  }, [handleClick]);

  // Mermaid rendering — must run before hljs so mermaid blocks are removed first
  useEffect(() => {
    const el = contentRef.current;
    if (!el) return;

    const blocks = el.querySelectorAll<HTMLElement>("code.language-mermaid");
    if (blocks.length === 0) return;

    let cancelled = false;

    const renderMermaid = async () => {
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
            container.className =
              "mermaid-svg retro-inset my-2 overflow-x-auto";
            container.innerHTML = svg;
            pre.replaceWith(container);
          } catch {
            // Leave raw <pre> as fallback
          }
        })
      );
    };

    void renderMermaid();
    return () => {
      cancelled = true;
    };
  }, [resolvedHtml]);

  // Syntax highlighting — runs after mermaid has consumed its blocks
  useEffect(() => {
    const el = contentRef.current;
    if (!el) return;

    let cancelled = false;

    const highlight = async () => {
      await highlightCodeBlocks(el);
      if (cancelled) return;

      // Add copy buttons to pre blocks that don't already have one
      const pres = el.querySelectorAll<HTMLElement>("pre");
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
    };

    void highlight();
    return () => {
      cancelled = true;
    };
  }, [resolvedHtml]);

  // Embed rendering — resolve ![[Note]] placeholders with inline note content
  useEffect(() => {
    const el = contentRef.current;
    if (!el) return;

    const embeds = el.querySelectorAll<HTMLElement>(".wiki-embed");
    embeds.forEach(embed => {
      const target = embed.getAttribute("data-target") ?? "";
      if (!target) return;
      // Don't re-render already-populated embeds
      if (embed.dataset.resolved) return;
      embed.dataset.resolved = "true";

      fetch(`/api/notes/${encodeURIComponent(target)}`)
        .then(res => res.json())
        .then(data => {
          if (data.html) {
            const container = document.createElement("div");
            container.className = "wiki-embed-content retro-inset my-2";
            container.innerHTML = data.html;
            embed.appendChild(container);
          }
        })
        .catch(() => {
          // Show broken link indicator
          embed.textContent = `⚠ Embed not found: ${target}`;
          embed.className = "wiki-embed wiki-embed-broken";
        });
    });
  }, [resolvedHtml]);

  // Heading permalinks — inject # link after each heading
  useEffect(() => {
    const el = contentRef.current;
    if (!el) return;

    const headings = el.querySelectorAll<HTMLElement>("h1, h2, h3, h4, h5, h6");
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
  }, [resolvedHtml]);

  const rawMarkdownUrl = useMemo(() => {
    if (note.rawMarkdown !== undefined) {
      return `data:text/markdown;charset=utf-8,${encodeURIComponent(note.rawMarkdown)}`;
    }
    return `/api/notes/${encodeURIComponent(note.slug)}/raw`;
  }, [note.rawMarkdown, note.slug]);

  // Copy raw markdown to clipboard. Live API notes intentionally do not carry
  // rawMarkdown; static vault notes keep their bundled source because there is
  // no same-origin Go /raw endpoint in VITE_STATIC_VAULT builds.
  const handleCopyMarkdown = useCallback(async () => {
    if (note.rawMarkdown !== undefined) {
      await navigator.clipboard.writeText(note.rawMarkdown);
    } else {
      const response = await fetch(
        `/api/notes/${encodeURIComponent(note.slug)}/raw`
      );
      if (!response.ok) {
        throw new Error(`failed to fetch markdown source: ${response.status}`);
      }
      await navigator.clipboard.writeText(await response.text());
    }
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }, [note.rawMarkdown, note.slug]);

  // Scroll to heading on hash navigation
  useEffect(() => {
    const hash = window.location.hash.slice(1);
    if (!hash || !contentRef.current) return;
    // Small delay to let content render
    const timer = setTimeout(() => {
      const target = contentRef.current?.querySelector(`#${CSS.escape(hash)}`);
      if (target) {
        target.scrollIntoView({ behavior: "smooth", block: "start" });
      }
    }, 200);
    return () => clearTimeout(timer);
  }, [note.slug, resolvedHtml]);
  const breadcrumbs = useMemo(() => {
    const parts = note.path.replace(/\.md$/, "").split("/");
    return parts.map((p, i) => ({
      label: p,
      slug: i < parts.length - 1 ? undefined : undefined,
    }));
  }, [note.path]);

  return (
    <article className={clsx("flex flex-col gap-4 retro-fade-in", className)}>
      {/* Breadcrumb */}
      <BreadcrumbBar segments={breadcrumbs} onNavigate={onNavigate} />

      {/* Title */}
      <h1 className="note-column-width text-2xl font-bold text-[var(--color-ink)] leading-tight border-b border-[var(--color-ink)] pb-2">
        {note.title}
      </h1>

      {/* Markdown actions */}
      <div className="note-column-width flex items-center gap-3 text-[11px]">
        <button
          type="button"
          className="retro-btn-flat"
          onClick={handleCopyMarkdown}
          title="Copy raw markdown to clipboard"
        >
          {copied ? "✓ Copied" : "⎘ Copy as Markdown"}
        </button>
        <a
          href={rawMarkdownUrl}
          download={`${note.slug}.md`}
          className="retro-btn-flat"
          title="Download markdown file"
        >
          ↓ Download .md
        </a>
        <a
          href={rawMarkdownUrl}
          className="retro-btn-flat"
          title="View raw markdown"
        >
          .md
        </a>
      </div>

      {/* Frontmatter */}
      <FrontmatterPanel
        frontmatter={note.frontmatter}
        tags={note.tags}
        modTime={note.modTime}
        onTagClick={onTagClick}
        className="note-column-width"
      />

      {/* Content */}
      <div
        ref={contentRef}
        className="note-prose"
        dangerouslySetInnerHTML={{ __html: resolvedHtml }}
      />

      {/* Lightbox for full-screen image/mermaid viewing */}
      <LightboxModal
        open={lightbox !== null}
        onOpenChange={open => {
          if (!open) setLightbox(null);
        }}
        imageSrc={lightbox?.type === "image" ? lightbox.src : undefined}
        imageAlt={lightbox?.type === "image" ? lightbox.alt : undefined}
        svgHtml={lightbox?.type === "mermaid" ? lightbox.svgHtml : undefined}
      />
    </article>
  );
};
