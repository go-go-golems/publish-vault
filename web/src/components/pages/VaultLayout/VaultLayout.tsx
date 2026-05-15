/**
 * PAGE: VaultLayout
 * Design: Retro System 1 — fixed menubar at top, resizable sidebar left, content pane right.
 * Mobile (<768px): sidebar becomes an off-canvas drawer, content is full-width.
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
} from "../../../store/uiSlice";

export interface VaultLayoutProps {
  children: React.ReactNode;
  vaultName?: string;
}

export const VaultLayout: React.FC<VaultLayoutProps> = ({
  children,
  vaultName = "Vault",
}) => {
  const dispatch = useAppDispatch();
  const [, navigate] = useLocation();
  const sidebarOpen = useAppSelector((s) => s.ui.sidebarOpen);
  const rightPanelOpen = useAppSelector((s) => s.ui.rightPanelOpen);
  const activeSlug = useAppSelector((s) => s.ui.activeNoteSlug);
  const { data: tree, isLoading: treeLoading } = useGetTreeQuery();

  const handleNavigate = useCallback(
    (slug: string) => {
      dispatch(setActiveNote(slug));
      navigate(`/note/${slug}`);
      // Close sidebar on mobile after navigation
      if (window.innerWidth < 768 && sidebarOpen) {
        dispatch(toggleSidebar());
      }
    },
    [dispatch, navigate, sidebarOpen]
  );

  const handleSearch = useCallback(
    (q: string) => {
      dispatch(setSearchQuery(q));
      if (q.trim()) {
        navigate("/search");
        // Close sidebar on mobile after search
        if (window.innerWidth < 768 && sidebarOpen) {
          dispatch(toggleSidebar());
        }
      }
    },
    [dispatch, navigate, sidebarOpen]
  );

  const closeSidebarOnBackdrop = useCallback(() => {
    if (sidebarOpen) {
      dispatch(toggleSidebar());
    }
  }, [dispatch, sidebarOpen]);

  return (
    <div className="flex flex-col h-screen overflow-hidden bg-[var(--color-paper)]">
      {/* ── Menu Bar ── */}
      <header className="retro-menubar shrink-0 z-50">
        <button
          type="button"
          className="retro-menubar-item"
          onClick={() => dispatch(toggleSidebar())}
          aria-label="Toggle sidebar"
        >
          <Icon name="menu" size={13} />
        </button>

        {/* Desktop: vault name as clickable home button */}
        <button
          type="button"
          className={clsx(
            "retro-menubar-item font-bold tracking-widest",
            "hidden md:flex"
          )}
          onClick={() => navigate("/")}
          aria-label="Go to vault home"
        >
          &#9670; {vaultName}
        </button>

        {/* Mobile: truncated vault name (not clickable) */}
        <span className="retro-menubar-item font-bold tracking-widest md:hidden truncate">
          {vaultName}
        </span>

        <div className="retro-menubar-separator hidden md:block" />

        <button
          type="button"
          className={clsx("retro-menubar-item", "hidden md:flex")}
          onClick={() => navigate("/search")}
        >
          Search
        </button>

        <div className="flex-1" />

        {/* Mobile: search icon */}
        <button
          type="button"
          className="retro-menubar-item md:hidden"
          onClick={() => navigate("/search")}
          aria-label="Search"
        >
          <Icon name="search" size={13} />
        </button>

        <button
          type="button"
          className={clsx(
            "retro-menubar-item",
            "hidden md:flex",
            rightPanelOpen && "bg-[var(--color-paper)] text-[var(--color-ink)]"
          )}
          onClick={() => dispatch(toggleRightPanel())}
          title="Toggle right panel"
        >
          <Icon name="panel-right" size={13} />
        </button>

        <span className="retro-menubar-item text-[10px] tabular-nums select-none hidden md:inline-flex">
          {new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
        </span>
      </header>

      {/* ── Mobile sidebar backdrop ── */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 bg-black/30 z-30 md:hidden"
          onClick={closeSidebarOnBackdrop}
          aria-hidden="true"
        />
      )}

      {/* ── Body with responsive layout ── */}
      <div className="flex flex-1 overflow-hidden relative">
        {/* ── Mobile: sidebar as off-canvas drawer ── */}
        <div
          className={clsx(
            "fixed inset-y-0 left-0 z-40 w-[80vw] max-w-[320px] md:hidden",
            "transition-transform duration-200 ease-in-out",
            sidebarOpen ? "translate-x-0" : "-translate-x-full",
            "top-[28px]" /* below menubar */
          )}
        >
          <Sidebar
            tree={tree ?? null}
            activeSlug={activeSlug ?? undefined}
            onSelectNote={handleNavigate}
            onSearch={handleSearch}
            vaultName={vaultName}
            isLoading={treeLoading}
            className="h-full border-r border-[var(--color-ink)]"
          />
        </div>

        {/* ── Desktop: resizable panel group ── */}
        {sidebarOpen ? (
          <ResizablePanelGroup direction="horizontal" className="flex-1 hidden md:flex">
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
        ) : null}

        {/* ── Desktop content when sidebar closed ── */}
        {!sidebarOpen && (
          <main className="flex-1 overflow-y-auto retro-scroll hidden md:block">
            {children}
          </main>
        )}

        {/* ── Mobile: always full-width content ── */}
        <main className="flex-1 overflow-y-auto retro-scroll md:hidden">
          {children}
        </main>
      </div>
    </div>
  );
};
