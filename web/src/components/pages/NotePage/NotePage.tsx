/**
 * PAGE: NotePage
 * Design: Retro System 1 — note content + right panel with backlinks + graph.
 * Fetches note by slug via RTK Query.
 */
import React, { useMemo, useCallback } from "react";
import { useLocation } from "wouter";
import { NoteRenderer } from "../../organisms/NoteRenderer/NoteRenderer";
import { BacklinksPanel } from "../../organisms/BacklinksPanel/BacklinksPanel";
import { GraphView } from "../../organisms/GraphView/GraphView";
import { ScrollArea } from "../../atoms/ScrollArea/ScrollArea";
import { Icon } from "../../atoms/Icon/Icon";
import {
  useGetNoteQuery,
  useListNotesQuery,
  useGetGraphQuery,
} from "../../../store/vaultApi";
import { useAppSelector, useAppDispatch } from "../../../hooks/redux";
import { setActiveNote } from "../../../store/uiSlice";

export interface NotePageProps {
  slug: string;
}

export const NotePage: React.FC<NotePageProps> = ({ slug }) => {
  const dispatch = useAppDispatch();
  const [, navigate] = useLocation();
  const graphVisible = useAppSelector((s) => s.ui.graphVisible);
  const rightPanelOpen = useAppSelector((s) => s.ui.rightPanelOpen);

  const handleNavigate = useCallback(
    (targetSlug: string) => {
      dispatch(setActiveNote(targetSlug));
      navigate(`/note/${targetSlug}`);
    },
    [dispatch, navigate]
  );

  const handleTagClick = useCallback(
    (_tag: string) => { navigate("/search"); },
    [navigate]
  );

  const {
    data: note,
    isLoading,
    isError,
  } = useGetNoteQuery(slug, { skip: !slug });

  const { data: allNotes } = useListNotesQuery();
  const { data: graphData } = useGetGraphQuery(undefined, { skip: !graphVisible });

  const allSlugs = useMemo(
    () => allNotes?.map((n) => n.slug) ?? [],
    [allNotes]
  );

  // Build backlink entries from note backlinks + allNotes index
  const backlinkEntries = useMemo(() => {
    if (!note || !allNotes) return [];
    return (note.backlinks ?? [])
      .map((bSlug) => {
        const found = allNotes.find((n) => n.slug === bSlug);
        return found
          ? { slug: found.slug, title: found.title, excerpt: found.excerpt }
          : null;
      })
      .filter(Boolean) as { slug: string; title: string; excerpt?: string }[];
  }, [note, allNotes]);

  if (!slug) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-4 text-[var(--color-muted-foreground)]">
        <Icon name="book" size={48} strokeWidth={1} />
        <p className="text-sm font-bold">Select a note to begin reading</p>
        <p className="text-xs">Use the sidebar or press <kbd className="border border-[var(--color-muted)] px-1">/</kbd> to search</p>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full gap-2 text-[var(--color-muted-foreground)] text-xs">
        <Icon name="file" size={14} className="animate-pulse" />
        Loading note…
      </div>
    );
  }

  if (isError || !note) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-2 text-[var(--color-destructive-accent)]">
        <Icon name="alert" size={24} />
        <p className="text-sm font-bold">Note not found</p>
        <p className="text-xs text-[var(--color-muted-foreground)]">Slug: {slug}</p>
      </div>
    );
  }

  return (
    <div className="flex h-full">
      {/* Note content */}
      <ScrollArea className="flex-1 p-6">
        <NoteRenderer
          note={note}
          allSlugs={allSlugs}
          onNavigate={handleNavigate}
          onTagClick={handleTagClick}
          className="max-w-3xl"
        />
      </ScrollArea>

      {/* Right panel */}
      {rightPanelOpen && (
        <aside className="w-56 shrink-0 border-l border-[var(--color-ink)] flex flex-col overflow-hidden">
          {/* Graph */}
          {graphVisible && graphData && (
            <div className="border-b border-[var(--color-ink)]">
              <div className="retro-window-title bg-[var(--color-ink)] text-[var(--color-paper)] text-[10px] font-bold uppercase tracking-widest px-2 py-1 flex items-center gap-1">
                <Icon name="graph" size={10} />
                Graph
              </div>
              <GraphView
                data={graphData}
                activeNodeId={slug}
                onNodeClick={handleNavigate}
                width={224}
                height={200}
              />
            </div>
          )}

          {/* Backlinks */}
          <div className="flex-1 overflow-hidden">
            <BacklinksPanel
              backlinks={backlinkEntries}
              onNavigate={handleNavigate}
              maxHeight="100%"
            />
          </div>
        </aside>
      )}
    </div>
  );
};
