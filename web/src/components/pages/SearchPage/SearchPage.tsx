/**
 * PAGE: SearchPage
 * Design: Retro System 1 — search results list driven by the canonical URL.
 *
 * The browser URL is the committed search request (text + structured filters +
 * sort + pagination). The page decodes it into a typed SearchRequest, queries
 * the shared advanced-search endpoint, and renders results with authored dates.
 * A filter panel edits the structured filters; the text field and sort live in
 * the header. Invalid URL filters render a reset action rather than being
 * silently dropped.
 */
import React, { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { NoteCard } from "../../molecules/NoteCard/NoteCard";
import { TagCloud } from "../../molecules/TagCloud/TagCloud";
import { SearchBar } from "../../molecules/SearchBar/SearchBar";
import { AdvancedSearchPanel } from "../../molecules/AdvancedSearchPanel/AdvancedSearchPanel";
import { Badge } from "../../atoms/Badge/Badge";
import { Icon } from "../../atoms/Icon/Icon";
import { ScrollArea } from "../../atoms/ScrollArea/ScrollArea";
import {
  useGetConfigQuery,
  useListTagsQuery,
  useSearchAdvancedQuery,
} from "../../../store/vaultApi";
import { useAppDispatch } from "../../../hooks/redux";
import { setActiveNote } from "../../../store/uiSlice";
import type { SearchRequest, SearchSort } from "../../../types";
import {
  decodeSearchParams,
  encodeSearchParams,
  isEffective,
  normalizeSearchRequest,
} from "../../../search/searchParams";

export interface SearchPageProps {
  // No required external props — navigation is handled internally
}

const DEFAULT_REQUEST: SearchRequest = {
  query: "",
  tags: [],
  tagMode: "all",
  pathPrefixes: [],
  dateField: "",
  sort: "",
  limit: 0,
  offset: 0,
};

export const SearchPage: React.FC<SearchPageProps> = () => {
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const [filterOpen, setFilterOpen] = useState(false);
  const { data: config } = useGetConfigQuery();

  // ── URL → typed request ──────────────────────────────────────
  const decoded = useMemo(() => decodeSearchParams(searchParams), [searchParams]);
  const normalized = useMemo(
    () => normalizeSearchRequest(decoded.request),
    [decoded],
  );
  const request = normalized.request;
  const errors = useMemo(
    () => [...decoded.errors, ...normalized.errors],
    [decoded, normalized],
  );
  const effective = isEffective(request);

  const { data: tags, isLoading: tagsLoading } = useListTagsQuery();
  const { data, isFetching } = useSearchAdvancedQuery(request, {
    skip: !effective || errors.length > 0,
  });

  // Update page title
  useEffect(() => {
    const siteTitle = config?.pageTitle || config?.vaultName || "Retro Obsidian Publish";
    document.title = `Search${request.query ? `: ${request.query}` : ""} — ${siteTitle}`;
  }, [config?.pageTitle, config?.vaultName, request.query]);

  // ── Commit a request back to the URL ────────────────────────
  const commitRequest = useCallback(
    (req: SearchRequest) => {
      const { request: norm, errors: errs } = normalizeSearchRequest(req);
      if (errs.length > 0) return;
      setSearchParams(encodeSearchParams(norm), { replace: true });
    },
    [setSearchParams],
  );

  const handleSearch = useCallback(
    (q: string) => commitRequest({ ...request, query: q, offset: 0 }),
    [commitRequest, request],
  );

  const handleSortChange = useCallback(
    (sort: SearchSort) => commitRequest({ ...request, sort, offset: 0 }),
    [commitRequest, request],
  );

  const handleApplyFilters = useCallback(
    (req: SearchRequest) => commitRequest(req),
    [commitRequest],
  );

  const handleSelectNote = useCallback(
    (slug: string) => {
      dispatch(setActiveNote(slug));
      navigate(`/note/${slug}`);
    },
    [dispatch, navigate],
  );

  const handleTagClick = useCallback(
    (tag: string) => commitRequest({ ...request, query: `#${tag}`, offset: 0 }),
    [commitRequest, request],
  );

  const handleRemoveTag = useCallback(
    (tag: string) =>
      commitRequest({ ...request, tags: request.tags.filter((t) => t !== tag), offset: 0 }),
    [commitRequest, request],
  );

  const handleRemovePath = useCallback(
    (p: string) =>
      commitRequest({ ...request, pathPrefixes: request.pathPrefixes.filter((x) => x !== p), offset: 0 }),
    [commitRequest, request],
  );

  const handleClearDates = useCallback(
    () => commitRequest({ ...request, dateField: "", dateFrom: undefined, dateTo: undefined, offset: 0 }),
    [commitRequest, request],
  );

  const handleResetFilters = useCallback(() => {
    commitRequest({ ...DEFAULT_REQUEST, query: request.query, sort: request.sort });
  }, [commitRequest, request.query, request.sort]);

  const handleResetAll = useCallback(() => {
    setSearchParams(new URLSearchParams(), { replace: true });
  }, [setSearchParams]);

  const activeFilterCount =
    request.tags.length +
    request.pathPrefixes.length +
    (request.dateFrom || request.dateTo ? 1 : 0);

  const total = data?.total ?? 0;
  const limit = request.limit || 30;
  const offset = request.offset;
  const hasNext = offset + limit < total;
  const hasPrev = offset > 0;

  const dateRangeLabel =
    request.dateFrom || request.dateTo
      ? `${request.dateFrom ? `${request.dateFrom.year}-${String(request.dateFrom.month).padStart(2, "0")}-${String(request.dateFrom.day).padStart(2, "0")}` : "…"} → ${request.dateTo ? `${request.dateTo.year}-${String(request.dateTo.month).padStart(2, "0")}-${String(request.dateTo.day).padStart(2, "0")}` : "…"}`
      : "";

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="border-b border-[var(--color-ink)] p-3 bg-[var(--color-panel)] shrink-0">
        <div className="flex items-center gap-2 mb-2">
          <Icon name="search" size={14} />
          <span className="text-xs font-bold uppercase tracking-widest">Search Vault</span>
          <button
            type="button"
            onClick={() => setFilterOpen(true)}
            className="ml-auto flex items-center gap-1 rounded border border-[var(--color-ink)] px-2 py-0.5 text-xs"
          >
            <Icon name="menu" size={11} />
            Filters
            {activeFilterCount > 0 && (
              <Badge variant="default" className="ml-1">
                {activeFilterCount}
              </Badge>
            )}
          </button>
          <select
            value={request.sort}
            onChange={(e) => handleSortChange(e.target.value as SearchSort)}
            className="rounded border border-[var(--color-ink)] bg-transparent px-1 py-0.5 text-xs"
            aria-label="Sort order"
          >
            <option value="relevance">Relevance</option>
            <option value="newest">Newest</option>
            <option value="oldest">Oldest</option>
          </select>
          {data && (
            <Badge variant="muted">
              {total} result{total !== 1 ? "s" : ""}
            </Badge>
          )}
        </div>
        <SearchBar
          onSearch={handleSearch}
          value={request.query}
          onChange={(q) => commitRequest({ ...request, query: q, offset: 0 })}
          autoFocus
          debounceMs={300}
        />
      </div>

      {/* Applied filter chips */}
      {activeFilterCount > 0 && (
        <div className="flex items-center gap-1.5 px-3 py-1.5 border-b border-[var(--color-ink)] flex-wrap bg-[var(--color-panel)]">
          {request.tags.map((t) => (
            <button
              key={t}
              type="button"
              onClick={() => handleRemoveTag(t)}
              className="flex items-center gap-1 rounded border border-[var(--color-ink)] px-1.5 py-0.5 text-[11px]"
            >
              #{t} <Icon name="close" size={9} />
            </button>
          ))}
          {request.pathPrefixes.map((p) => (
            <button
              key={p}
              type="button"
              onClick={() => handleRemovePath(p)}
              className="flex items-center gap-1 rounded border border-[var(--color-ink)] px-1.5 py-0.5 text-[11px]"
            >
              {p} <Icon name="close" size={9} />
            </button>
          ))}
          {dateRangeLabel && (
            <button
              type="button"
              onClick={handleClearDates}
              className="flex items-center gap-1 rounded border border-[var(--color-ink)] px-1.5 py-0.5 text-[11px]"
            >
              {dateRangeLabel} <Icon name="close" size={9} />
            </button>
          )}
          <button
            type="button"
            onClick={handleResetFilters}
            className="ml-auto text-[11px] underline"
          >
            Reset filters
          </button>
        </div>
      )}

      {/* Results */}
      <ScrollArea className="flex-1 p-2">
        {errors.length > 0 ? (
          <div className="flex flex-col items-center justify-center py-12 gap-2 text-[var(--color-muted-foreground)]">
            <Icon name="alert" size={24} strokeWidth={1} />
            <p className="text-xs font-bold">Invalid search filters in the URL</p>
            <ul className="text-[11px] list-disc">
              {errors.map((e, i) => (
                <li key={i}>{e.field}: {e.message}</li>
              ))}
            </ul>
            <button type="button" onClick={handleResetAll} className="text-xs underline">
              Reset all
            </button>
          </div>
        ) : !effective ? (
          <div className="p-4">
            <TagCloud
              tags={tags ?? []}
              onTagClick={(t) => handleTagClick(t)}
              className={tagsLoading ? "opacity-50" : ""}
            />
          </div>
        ) : isFetching ? (
          <div className="flex items-center justify-center py-8 gap-2 text-[var(--color-muted-foreground)] text-xs">
            <Icon name="search" size={14} className="animate-pulse" />
            Searching…
          </div>
        ) : data && data.results.length > 0 ? (
          <div className="flex flex-col gap-2">
            {data.results.map((r) => (
              <NoteCard
                key={r.slug}
                slug={r.slug}
                title={r.title}
                excerpt={r.excerpt}
                tags={r.tags}
                path={r.path}
                date={r.date}
                onClick={handleSelectNote}
                onTagClick={handleTagClick}
              />
            ))}
            <div className="flex items-center justify-center gap-2 py-3 text-xs">
              <button
                type="button"
                disabled={!hasPrev}
                onClick={() => commitRequest({ ...request, offset: Math.max(0, offset - limit) })}
                className="rounded border border-[var(--color-ink)] px-2 py-0.5 disabled:opacity-40"
              >
                Previous
              </button>
              <span className="text-[var(--color-muted-foreground)]">
                {offset + 1}–{Math.min(offset + limit, total)} of {total}
              </span>
              <button
                type="button"
                disabled={!hasNext}
                onClick={() => commitRequest({ ...request, offset: offset + limit })}
                className="rounded border border-[var(--color-ink)] px-2 py-0.5 disabled:opacity-40"
              >
                Next
              </button>
            </div>
          </div>
        ) : (
          <div className="flex flex-col items-center justify-center py-16 gap-2 text-[var(--color-muted-foreground)]">
            <Icon name="alert" size={24} strokeWidth={1} />
            <p className="text-xs font-bold">No results for &ldquo;{request.query}&rdquo;</p>
            {activeFilterCount > 0 && (
              <button type="button" onClick={handleResetFilters} className="text-xs underline">
                Reset filters
              </button>
            )}
          </div>
        )}
      </ScrollArea>

      <AdvancedSearchPanel
        open={filterOpen}
        onOpenChange={setFilterOpen}
        current={request}
        onApply={handleApplyFilters}
      />
    </div>
  );
};
