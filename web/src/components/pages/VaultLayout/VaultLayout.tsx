/**
 * PAGE: VaultLayout
 * Design: Retro System 1 — fixed menubar at top, resizable sidebar left, content pane right.
 * Navigation is handled internally via Wouter's useLocation hook.
 *
 * Retro macOS 1 design language:
 *   - Zero border-radius, 1px hard borders, Chicago system-ui stack
 *   - Monochrome base with #0000cc link blue and #005500 tag green accents
 *   - Aged-paper (#faf8f4) background, ink (#1a1a1a) foreground
 */
import React, { useCallback } from "react";
import { useLocation } from "wouter";
import { clsx } from "clsx";
import { Sidebar } from "../../organisms/Sidebar/Sidebar";
import { Icon } from "../../atoms/Icon/Icon";
import {
  ResizablePanelGroup,
  ResizablePanel,
  ResizableHandle,
} from "../../ui/resizable";
import { useGetTreeQuery } from "../../../store/vaultApi";
import { useAppDispatch, useAppSelector } from "../../../hooks/redux";
import {
  toggleSidebar,
  toggleRightPanel,
  setSearchQuery,
  setActiveNote,
  toggleGraph,
} from "../../../store/uiSlice";

export interface VaultLayoutProps {
  children: React.ReactNode;
  vaultName?: string;
}

export const VaultLayout: React.FC<VaultLayoutProps> = ({
  children,
  vaultName = "Demo Vault",
}) => {
  const dispatch = useAppDispatch();
  const [, navigate] = useLocation();
  const sidebarOpen = useAppSelector((s) => s.ui.sidebarOpen);
  const rightPanelOpen = useAppSelector((s) => s.ui.rightPanelOpen);
  const graphVisible = useAppSelector((s) => s.ui.graphVisible);
  const activeSlug = useAppSelector((s) => s.ui.activeNoteSlug);
  const { data: tree, isLoading: treeLoading } = useGetTreeQuery();

  const handleNavigate = useCallback(
    (slug: string) => {
      dispatch(setActiveNote(slug));
      navigate(`/note/${slug}`);
    },
    [dispatch, navigate]
  );

  const handleSearch = useCallback(
    (q: string) => {
      dispatch(setSearchQuery(q));
      if (q.trim()) {
        navigate("/search");
      }
    },
    [dispatch, navigate]
  );

  return (
    <div className="flex flex-col h-screen overflow-hidden bg-[var(--color-paper)]">
      {/* ── Menu Bar ── */}
      <header className="retro-menubar shrink-0 z-10">
        <button
          type="button"
          className="retro-menubar-item"
          onClick={() => dispatch(toggleSidebar())}
          aria-label="Toggle sidebar"
        >
          <Icon name="menu" size={13} />
        </button>

        <button
          type="button"
          className="retro-menubar-item font-bold tracking-widest"
          onClick={() => navigate("/")}
          aria-label="Go to vault home"
        >
          &#9670; {vaultName}
        </button>

        <div className="retro-menubar-separator" />

        <button
          type="button"
          className="retro-menubar-item"
          onClick={() => navigate("/search")}
        >
          Search
        </button>

        <div className="flex-1" />

        <button
          type="button"
          className={clsx(
            "retro-menubar-item",
            rightPanelOpen && "bg-[var(--color-paper)] text-[var(--color-ink)]"
          )}
          onClick={() => dispatch(toggleRightPanel())}
          title="Toggle right panel"
        >
          <Icon name="panel-right" size={13} />
        </button>

        <button
          type="button"
          className={clsx(
            "retro-menubar-item",
            graphVisible && "bg-[var(--color-paper)] text-[var(--color-ink)]"
          )}
          onClick={() => dispatch(toggleGraph())}
          title="Toggle graph view"
        >
          <Icon name="graph" size={13} />
        </button>

        <span className="retro-menubar-item text-[10px] tabular-nums select-none">
          {new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
        </span>
      </header>

      {/* ── Body with resizable panels ── */}
      <div className="flex flex-1 overflow-hidden">
        {sidebarOpen ? (
          <ResizablePanelGroup direction="horizontal" className="flex-1">
            <ResizablePanel
              defaultSize={20}
              minSize={12}
              maxSize={40}
              order={1}
              className="flex flex-col overflow-hidden"
            >
              <Sidebar
                tree={tree ?? null}
                activeSlug={activeSlug ?? undefined}
                onSelectNote={handleNavigate}
                onSearch={handleSearch}
                vaultName={vaultName}
                isLoading={treeLoading}
                className="border-r border-[var(--color-ink)] h-full"
              />
            </ResizablePanel>

            <ResizableHandle withHandle className="retro-resize-handle" />

            <ResizablePanel defaultSize={80} order={2}>
              <main className="h-full overflow-y-auto retro-scroll">
                {children}
              </main>
            </ResizablePanel>
          </ResizablePanelGroup>
        ) : (
          <main className="flex-1 overflow-y-auto retro-scroll">
            {children}
          </main>
        )}
      </div>
    </div>
  );
};
