/**
 * ORGANISM: NoteRenderer
 * Design: Retro System 1 — renders server-side HTML with wiki-link resolution,
 * syntax highlighting, and mermaid diagram rendering.
 * Handles click events on wiki-links for SPA navigation.
 */
import React, { useCallback, useEffect, useMemo, useRef } from "react";
import { clsx } from "clsx";
import hljs from "highlight.js";
import mermaid from "mermaid";
import { nanoid } from "nanoid";
import { resolveWikiLinks, buildSlugSet } from "../../../lib/wikiLinks";
import { FrontmatterPanel } from "../../molecules/FrontmatterPanel/FrontmatterPanel";
import { BreadcrumbBar } from "../../molecules/BreadcrumbBar/BreadcrumbBar";
import type { Note } from "../../../types";

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
      const anchor = target.closest("a");
      if (!anchor) return;

      const wikiTarget = anchor.getAttribute("data-target");
      if (wikiTarget && anchor.classList.contains("wiki-link")) {
        e.preventDefault();
        if (!anchor.classList.contains("broken")) {
          onNavigate(wikiTarget);
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

    blocks.forEach((block) => {
      const pre = block.parentElement;
      if (!pre || pre.tagName !== "PRE") return;
      const src = block.textContent ?? "";
      const id = `mermaid-${nanoid(6)}`;
      mermaid
        .render(id, src)
        .then(({ svg }) => {
          const container = document.createElement("div");
          container.className = "mermaid-svg retro-inset my-2 overflow-x-auto";
          container.innerHTML = svg;
          pre.replaceWith(container);
        })
        .catch(() => {
          // Leave raw <pre> as fallback
        });
    });
  }, [resolvedHtml]);

  // Syntax highlighting — runs after mermaid has consumed its blocks
  useEffect(() => {
    const el = contentRef.current;
    if (!el) return;

    const codeBlocks = el.querySelectorAll<HTMLElement>(
      "pre code:not(.language-mermaid)"
    );
    codeBlocks.forEach((block) => {
      // Only highlight if not already processed
      if (!block.dataset.highlighted) {
        hljs.highlightElement(block);
        block.dataset.highlighted = "true";
      }
    });
  }, [resolvedHtml]);

  // Build breadcrumb from path
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
      <h1 className="text-2xl font-bold text-[var(--color-ink)] leading-tight border-b border-[var(--color-ink)] pb-2">
        {note.title}
      </h1>

      {/* Frontmatter */}
      <FrontmatterPanel
        frontmatter={note.frontmatter}
        tags={note.tags}
        modTime={note.modTime}
        onTagClick={onTagClick}
      />

      {/* Content */}
      <div
        ref={contentRef}
        className="note-prose"
        dangerouslySetInnerHTML={{ __html: resolvedHtml }}
      />
    </article>
  );
};
