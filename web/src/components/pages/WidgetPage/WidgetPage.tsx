/**
 * PAGE: WidgetPage
 * Design: Retro System 1 — renders a server-driven widget.dsl page
 * (GET /api/widget/pages/{id}) through the generic WidgetRenderer and the
 * default registry. Navigate actions route through React Router; server
 * actions round-trip to POST /api/widget/actions/{name} and refresh the IR.
 */
import React, { useCallback, useEffect, useMemo } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { ScrollArea } from "../../atoms/ScrollArea/ScrollArea";
import { Icon } from "../../atoms/Icon/Icon";
import { SidebarNav } from "../../molecules/SidebarNav/SidebarNav";
import { useWidgetPage } from "../../../hooks/useWidgetPage";
import { resolvePageShell } from "../../../widgets/shell";
import { useSidebarOverride } from "../../../widgets/sidebarSlot";
import {
  confirmWidgetAction,
  dispatchWidgetAction,
  structuredNavigationTarget,
  type WidgetActionContext,
} from "../../../widgets/actions";
import type { ActionSpec } from "../../../widgets/ir";
import { defaultWidgetRegistry } from "../../../widgets/defaultRegistry";
import { WidgetRenderer, WidgetToastRegion } from "../../../widgets/WidgetRenderer";
import { useGetConfigQuery } from "../../../store/vaultApi";

export interface WidgetPageProps {
  pageId: string;
}

export const WidgetPage: React.FC<WidgetPageProps> = ({ pageId }) => {
  const navigate = useNavigate();
  const location = useLocation();
  // Forward the route's query string so page scripts can read request.query
  // (e.g. /w/reader?slug=foo).
  const { page, loading, error, refresh } = useWidgetPage(
    `/api/widget/pages/${encodeURIComponent(pageId)}${location.search}`
  );
  const { data: config } = useGetConfigQuery();

  useEffect(() => {
    if (!page) return;
    const siteTitle = config?.pageTitle || config?.vaultName || "Retro Obsidian Publish";
    document.title = `${page.title} — ${siteTitle}`;
  }, [config?.pageTitle, config?.vaultName, page]);

  // Server actions dispatch fire-and-forget; refresh the IR when a result
  // asks for it.
  useEffect(() => {
    const listener = (event: Event) => {
      const detail = (event as CustomEvent<{ responseOk?: boolean; result?: { refresh?: boolean } }>)
        .detail;
      if (detail?.responseOk && detail.result?.refresh) refresh();
    };
    window.addEventListener("widget:action-result", listener);
    return () => window.removeEventListener("widget:action-result", listener);
  }, [refresh]);

  const handleAction = useCallback(
    (action: ActionSpec, context: WidgetActionContext) => {
      if (action.kind === "navigate") {
        // The dispatcher's confirm gate is bypassed when a handler takes over,
        // so honor action.confirm here before routing.
        if (!confirmWidgetAction(action, context)) return;
        navigate(structuredNavigationTarget(action, context), {
          replace: action.replace ?? false,
        });
        return;
      }
      // Delegate everything else (server/copy/download/event/overlay) to the
      // default dispatcher — called WITHOUT onAction to avoid recursion; it
      // runs its own confirm gate.
      dispatchWidgetAction(action, context);
    },
    [navigate]
  );

  // JS-declared shell: an app shell with sidebar placement replaces the
  // vault file tree for the lifetime of this page (kind "none"/"root-owned"
  // currently renders with vault chrome — v1 limitation, see -016 §3.3).
  const shell = useMemo(() => resolvePageShell(page?.shell), [page?.shell]);
  const sidebarElement = useMemo(() => {
    if (shell.kind !== "app" || shell.navigation.placement !== "sidebar") {
      return null;
    }
    return (
      <SidebarNav
        navigation={shell.navigation}
        onItemSelect={item => {
          if (item.action) handleAction(item.action, { navItemId: item.id });
        }}
      />
    );
  }, [shell, handleAction]);
  useSidebarOverride(sidebarElement);

  if (loading && !page) {
    return (
      <div className="flex items-center justify-center h-full gap-2 text-[var(--color-muted-foreground)] text-xs">
        <Icon name="file" size={14} className="animate-pulse" />
        Loading page…
      </div>
    );
  }

  if (error || !page) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-2 text-[var(--color-destructive-accent)]">
        <Icon name="alert" size={24} />
        <p className="text-sm font-bold">Widget page not found</p>
        <p className="text-xs text-[var(--color-muted-foreground)]">
          {error ? error.message : `Page: ${pageId}`}
        </p>
      </div>
    );
  }

  return (
    <ScrollArea className="h-full p-6">
      <div className="max-w-5xl flex flex-col gap-4 retro-fade-in">
        <h1 className="text-2xl font-bold text-[var(--color-ink)] leading-tight border-b border-[var(--color-ink)] pb-2">
          {page.title}
        </h1>
        <WidgetRenderer node={page.root} registry={defaultWidgetRegistry} onAction={handleAction} />
      </div>
      <WidgetToastRegion />
    </ScrollArea>
  );
};
