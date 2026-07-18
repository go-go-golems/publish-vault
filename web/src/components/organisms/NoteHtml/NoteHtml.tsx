/**
 * ORGANISM: NoteHtml
 * Design: Retro System 1 — standalone rendered-note-HTML widget: server HTML
 * plus the enhancement pipeline (mermaid → highlight/copy → embeds →
 * heading anchors), each stage gated by a prop so widget pages can opt out.
 *
 * This is the IR-consumable sibling of NoteView's body: no breadcrumb,
 * title, actions, or frontmatter — compose those separately (they are their
 * own widgets). Wiki-link clicks surface through onWikiLinkNavigate so the
 * widget adapter can translate them into dispatched navigate actions.
 */
import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { resolveWikiLinks, buildSlugSet } from "../../../lib/wikiLinks";
import { LightboxModal } from "../../atoms/LightboxModal/LightboxModal";
import { useAppDispatch } from "../../../hooks/redux";
import { vaultApi, useListNotesQuery } from "../../../store/vaultApi";
import {
  enhanceMermaid,
  enhanceCodeBlocks,
  enhanceHeadingAnchors,
  resolveEmbeds,
} from "../NoteView/noteEnhancements";
import { NoteBody } from "../NoteView/NoteBody";

interface LightboxState {
  type: "image" | "mermaid";
  src?: string;
  alt?: string;
  svgHtml?: string;
}

export interface NoteHtmlProps {
  html: string;
  slug?: string;
  /** Enhancement toggles; all default to true. */
  mermaid?: boolean;
  highlight?: boolean;
  embeds?: boolean;
  anchors?: boolean;
  onWikiLinkNavigate?: (slug: string) => void;
}

export const NoteHtml: React.FC<NoteHtmlProps> = ({
  html,
  slug,
  mermaid = true,
  highlight = true,
  embeds = true,
  anchors = true,
  onWikiLinkNavigate,
}) => {
  const dispatch = useAppDispatch();
  const contentRef = useRef<HTMLDivElement>(null);
  const [lightbox, setLightbox] = useState<LightboxState | null>(null);

  const { data: allNotes } = useListNotesQuery();
  const slugSet = useMemo(
    () => buildSlugSet(allNotes?.map(n => n.slug) ?? []),
    [allNotes]
  );

  // Same hydration-safe two-phase resolution as NoteView: raw server HTML
  // first, wiki-link resolution in a post-hydration effect.
  const [resolvedHtml, setResolvedHtml] = useState(html);
  const renderedHtml = useRef(html);
  if (renderedHtml.current !== html) {
    renderedHtml.current = html;
    setResolvedHtml(html);
  }

  useEffect(() => {
    setResolvedHtml(resolveWikiLinks(html, slugSet));
  }, [html, slugSet]);

  const loadNoteHtml = useCallback(
    async (target: string): Promise<string | null> => {
      const sub = dispatch(vaultApi.endpoints.getNote.initiate(target));
      try {
        const note = await sub.unwrap();
        return note.html ?? null;
      } catch {
        return null;
      } finally {
        sub.unsubscribe();
      }
    },
    [dispatch]
  );

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

      const img = target.closest("img");
      if (img && contentRef.current?.contains(img)) {
        e.preventDefault();
        setLightbox({ type: "image", src: img.src, alt: img.alt || undefined });
        return;
      }

      const mermaidEl = target.closest(".mermaid-svg");
      if (mermaidEl && contentRef.current?.contains(mermaidEl)) {
        e.preventDefault();
        setLightbox({ type: "mermaid", svgHtml: mermaidEl.innerHTML });
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
          onWikiLinkNavigate?.(`${wikiTarget}${hash}`);
        }
        return;
      }

      const href = anchor.getAttribute("href");
      if (href?.startsWith("#")) return;
      if (href?.startsWith("/note/")) {
        e.preventDefault();
        onWikiLinkNavigate?.(href.replace("/note/", ""));
      }
    },
    [onWikiLinkNavigate]
  );

  useEffect(() => {
    const el = contentRef.current;
    if (!el) return;
    el.addEventListener("click", handleClick);
    return () => el.removeEventListener("click", handleClick);
  }, [handleClick]);

  // Ordering constraint: mermaid must consume its blocks before highlight.
  useEffect(() => {
    const el = contentRef.current;
    if (!el || !mermaid) return;
    return enhanceMermaid(el);
  }, [resolvedHtml, mermaid]);

  useEffect(() => {
    const el = contentRef.current;
    if (!el || !highlight) return;
    return enhanceCodeBlocks(el);
  }, [resolvedHtml, highlight]);

  useEffect(() => {
    const el = contentRef.current;
    if (!el || !embeds) return;
    resolveEmbeds(el, loadNoteHtml);
  }, [resolvedHtml, embeds, loadNoteHtml]);

  useEffect(() => {
    if (!anchors) return;
    const timer = window.setTimeout(() => {
      const el = contentRef.current;
      if (!el) return;
      enhanceHeadingAnchors(el);
    }, 0);
    return () => window.clearTimeout(timer);
  }, [resolvedHtml, anchors]);

  // Scroll to heading on hash navigation
  useEffect(() => {
    const hash = window.location.hash.slice(1);
    if (!hash || !contentRef.current) return;
    const timer = setTimeout(() => {
      const target = contentRef.current?.querySelector(`#${CSS.escape(hash)}`);
      if (target) {
        target.scrollIntoView({ behavior: "smooth", block: "start" });
      }
    }, 200);
    return () => clearTimeout(timer);
  }, [slug, resolvedHtml]);

  return (
    <>
      <NoteBody ref={contentRef} html={resolvedHtml} />
      <LightboxModal
        open={lightbox !== null}
        onOpenChange={open => {
          if (!open) setLightbox(null);
        }}
        imageSrc={lightbox?.type === "image" ? lightbox.src : undefined}
        imageAlt={lightbox?.type === "image" ? lightbox.alt : undefined}
        svgHtml={lightbox?.type === "mermaid" ? lightbox.svgHtml : undefined}
      />
    </>
  );
};
