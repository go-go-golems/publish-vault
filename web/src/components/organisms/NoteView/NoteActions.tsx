/**
 * ORGANISM PART: NoteActions
 * Design: Retro System 1 — copy / download / view-raw markdown action row.
 * Raw markdown is loaded through the injected loader (RTK Query in NoteView)
 * rather than a direct fetch, so it works in both backend and static modes.
 */
import React, { useCallback, useMemo, useState } from "react";

export interface NoteActionsProps {
  slug: string;
  /** Bundled raw markdown, when the note carries it (static vault mode). */
  rawMarkdown?: string;
  /** Loads the raw markdown source on demand. */
  loadRawMarkdown: () => Promise<string>;
}

export const NoteActions: React.FC<NoteActionsProps> = ({
  slug,
  rawMarkdown,
  loadRawMarkdown,
}) => {
  const [copied, setCopied] = useState(false);

  const rawMarkdownUrl = useMemo(() => {
    if (rawMarkdown !== undefined) {
      return `data:text/markdown;charset=utf-8,${encodeURIComponent(rawMarkdown)}`;
    }
    return `/api/notes/${encodeURIComponent(slug)}/raw`;
  }, [rawMarkdown, slug]);

  const handleCopyMarkdown = useCallback(async () => {
    const source =
      rawMarkdown !== undefined ? rawMarkdown : await loadRawMarkdown();
    await navigator.clipboard.writeText(source);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }, [rawMarkdown, loadRawMarkdown]);

  return (
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
        download={`${slug}.md`}
        className="retro-btn-flat"
        title="Download markdown file"
      >
        ↓ Download .md
      </a>
      <a href={rawMarkdownUrl} className="retro-btn-flat" title="View raw markdown">
        .md
      </a>
    </div>
  );
};
