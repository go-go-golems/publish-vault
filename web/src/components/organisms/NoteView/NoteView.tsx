/**
 * ORGANISM: NoteView
 * Design: Retro System 1 — composition root for a full note page: breadcrumb,
 * title, markdown actions, frontmatter, and the rendered body.
 *
 * The body (HTML hosting, wiki-link resolution, enhancement pipeline, click
 * delegation, lightbox) is owned by NoteHtml — the same component the widget
 * registry renders for JS-composed pages — so there is exactly one
 * note-rendering path. NoteView only adds the page furniture around it.
 * All network access goes through RTK Query (vaultApi), never raw fetch.
 */
import React, { useCallback, useMemo } from "react";
import { clsx } from "clsx";
import { FrontmatterPanel } from "../../molecules/FrontmatterPanel/FrontmatterPanel";
import { BreadcrumbBar } from "../../molecules/BreadcrumbBar/BreadcrumbBar";
import { useAppDispatch } from "../../../hooks/redux";
import { vaultApi } from "../../../store/vaultApi";
import { NoteHtml } from "../NoteHtml/NoteHtml";
import { NoteActions } from "./NoteActions";
import type { Note } from "../../../types";

export interface NoteViewProps {
  note: Note;
  onNavigate: (slug: string) => void;
  onTagClick?: (tag: string) => void;
  className?: string;
}

export const NoteView: React.FC<NoteViewProps> = ({
  note,
  onNavigate,
  onTagClick,
  className,
}) => {
  const dispatch = useAppDispatch();

  const loadRawMarkdown = useCallback(async (): Promise<string> => {
    const sub = dispatch(vaultApi.endpoints.getNoteRaw.initiate(note.slug));
    try {
      return await sub.unwrap();
    } finally {
      sub.unsubscribe();
    }
  }, [dispatch, note.slug]);

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

      {/* Content — single note-rendering path shared with widget pages */}
      <NoteHtml html={note.html} slug={note.slug} onWikiLinkNavigate={onNavigate} />
    </article>
  );
};
