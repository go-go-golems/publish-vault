/**
 * useWidgetPage — fetches a WidgetPage IR document from the widget host.
 * Ported from rag-evaluation-system packages/rag-evaluation-site
 * src/hooks/useWidgetPage.ts (v0.1.7); shell/shortcut specs are carried as
 * opaque data until publish-vault adopts them (design doc open question 4).
 */
import { useCallback, useEffect, useState } from "react";
import type { WidgetNode } from "../widgets/ir";

export interface WidgetPageResponse {
  id: string;
  title: string;
  schemaVersion?: string;
  shell?: unknown;
  shortcuts?: unknown;
  root: WidgetNode;
  meta?: Record<string, unknown>;
}

export interface UseWidgetPageOptions {
  enabled?: boolean;
  fetcher?: typeof fetch;
}

export interface UseWidgetPageResult {
  page: WidgetPageResponse | null;
  loading: boolean;
  error: Error | null;
  refresh: () => void;
}

export function useWidgetPage(
  url: string,
  options: UseWidgetPageOptions = {}
): UseWidgetPageResult {
  const { enabled = true, fetcher = fetch } = options;
  const [page, setPage] = useState<WidgetPageResponse | null>(null);
  const [loading, setLoading] = useState(Boolean(enabled && url));
  const [error, setError] = useState<Error | null>(null);
  const [version, setVersion] = useState(0);

  const refresh = useCallback(() => setVersion(current => current + 1), []);

  useEffect(() => {
    // Reading the refresh counter makes the deliberate re-fetch dependency explicit.
    void version;
    if (!enabled || !url) {
      setLoading(false);
      return;
    }

    const controller = new AbortController();
    setLoading(true);
    setError(null);

    fetcher(url, { signal: controller.signal })
      .then(async response => {
        if (!response.ok) {
          throw new Error(`Widget page request failed: ${response.status} ${response.statusText}`);
        }
        return response.json() as Promise<WidgetPageResponse>;
      })
      .then(nextPage => {
        if (!controller.signal.aborted) {
          setPage(nextPage);
        }
      })
      .catch((err: unknown) => {
        if (controller.signal.aborted) return;
        setError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        if (!controller.signal.aborted) {
          setLoading(false);
        }
      });

    return () => controller.abort();
  }, [enabled, fetcher, url, version]);

  return { page, loading, error, refresh };
}
