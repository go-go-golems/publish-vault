/**
 * ORGANISM: NoteView
 * Design: Retro System 1 — composition root for a rendered note: breadcrumb,
 * title, markdown actions, frontmatter, enhanced HTML body, and lightbox.
 *
 * Replaces the former monolithic NoteRenderer. Responsibilities are split:
 *   - NoteBody owns the dangerouslySetInnerHTML container.
 *   - noteEnhancements.ts owns the post-render DOM pipeline
 *     (mermaid → highlight/copy-buttons → embeds → heading anchors).
 *   - NoteActions owns the copy/download markdown row.
 *   - All network access goes through RTK Query (vaultApi), never raw fetch.
 */
import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { clsx } from "clsx";
import { resolveWikiLinks, buildSlugSet } from "../../../lib/wikiLinks";
import { FrontmatterPanel } from "../../molecules/FrontmatterPanel/FrontmatterPanel";
import { BreadcrumbBar } from "../../molecules/BreadcrumbBar/BreadcrumbBar";
import { LightboxModal } from "../../atoms/LightboxModal/LightboxModal";
import { useAppDispatch } from "../../../hooks/redux";
import { vaultApi } from "../../../store/vaultApi";
import {
  enhanceMermaid,
  enhanceCodeBlocks,
  enhanceHeadingAnchors,
  resolveEmbeds,
} from "./noteEnhancements";
import { NoteBody } from "./NoteBody";
import { NoteActions } from "./NoteActions";
import type { Note } from "../../../types";

/** Lightbox content descriptor */
interface LightboxState {
  type: "image" | "mermaid";
  src?: string;
  alt?: string;
  svgHtml?: string;
}

export interface NoteViewProps {
  note: Note;
  allSlugs: string[];
  onNavigate: (slug: string) => void;
  onTagClick?: (tag: string) => void;
  className?: string;
}

export const NoteView: React.FC<NoteViewProps> = ({
  note,
  allSlugs,
  onNavigate,
  onTagClick,
  className,
}) => {
  const dispatch = useAppDispatch();
  const contentRef = useRef<HTMLDivElement>(null);

  // Lightbox state for full-screen image/mermaid viewing
  const [lightbox, setLightbox] = useState<LightboxState | null>(null);

  // Build slug set for broken-link detection
  const slugSet = useMemo(() => buildSlugSet(allSlugs), [allSlugs]);

  // SSR has no DOMParser, while the browser's resolver normalizes HTML. Start
  // with the raw server HTML so hydration sees identical markup, then resolve
  // wiki links in an effect after hydration has completed.
  const [resolvedHtml, setResolvedHtml] = useState(note.html);
  const renderedNote = useRef({ slug: note.slug, html: note.html });

  // Reset synchronously during render when SPA navigation swaps notes. Waiting
  // for the link-resolution effect would briefly commit the previous note body
  // under the next note's title/frontmatter.
  if (
    renderedNote.current.slug !== note.slug ||
    renderedNote.current.html !== note.html
  ) {
    renderedNote.current = { slug: note.slug, html: note.html };
    setResolvedHtml(note.html);
  }

  useEffect(() => {
    setResolvedHtml(resolveWikiLinks(note.html, slugSet));
  }, [note.html, slugSet]);

  // ── RTK-backed loaders injected into enhancements/actions ──
  const loadNoteHtml = useCallback(
    async (slug: string): Promise<string | null> => {
      const sub = dispatch(vaultApi.endpoints.getNote.initiate(slug));
      try {
        const target = await sub.unwrap();
        return target.html ?? null;
      } catch {
        return null;
      } finally {
        sub.unsubscribe();
      }
    },
    [dispatch]
  );

  const loadRawMarkdown = useCallback(async (): Promise<string> => {
    const sub = dispatch(vaultApi.endpoints.getNoteRaw.initiate(note.slug));
    try {
      return await sub.unwrap();
    } finally {
      sub.unsubscribe();
    }
  }, [dispatch, note.slug]);

  // Intercept clicks inside the note body for SPA navigation, lightbox, and
  // collapsible callouts.
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
          if (toggle) toggle.textContent = isHidden ? "▼" : "▶";
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
    return enhanceMermaid(el);
  }, [resolvedHtml]);

  // Syntax highlighting + copy buttons — runs after mermaid has consumed its blocks
  useEffect(() => {
    const el = contentRef.current;
    if (!el) return;
    return enhanceCodeBlocks(el);
  }, [resolvedHtml]);

  // Embed rendering — resolve ![[Note]] placeholders with inline note content
  useEffect(() => {
    const el = contentRef.current;
    if (!el) return;
    resolveEmbeds(el, loadNoteHtml);
  }, [resolvedHtml, loadNoteHtml]);

  // Heading permalinks — deferred until hydration has completed; the SSR
  // markup intentionally contains only the raw note HTML.
  useEffect(() => {
    const timer = window.setTimeout(() => {
      const el = contentRef.current;
      if (!el) return;
      enhanceHeadingAnchors(el);
    }, 0);
    return () => window.clearTimeout(timer);
  }, [resolvedHtml]);

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
    return parts.map(p => ({
      label: p,
      slug: undefined,
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
      <NoteActions
        slug={note.slug}
        rawMarkdown={note.rawMarkdown}
        loadRawMarkdown={loadRawMarkdown}
      />

      {/* Frontmatter */}
      <FrontmatterPanel
        frontmatter={note.frontmatter}
        tags={note.tags}
        modTime={note.modTime}
        onTagClick={onTagClick}
        className="note-column-width"
      />

      {/* Content */}
      <NoteBody ref={contentRef} html={resolvedHtml} />

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
