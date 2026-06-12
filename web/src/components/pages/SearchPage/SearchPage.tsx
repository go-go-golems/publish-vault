/**
 * PAGE: SearchPage
 * Design: Retro System 1 — search results list with score, tags, excerpt.
 * Navigation is handled internally via React Router's useNavigate hook.
 */
import React, { useCallback, useEffect } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { NoteCard } from "../../molecules/NoteCard/NoteCard";
import { SearchBar } from "../../molecules/SearchBar/SearchBar";
import { Badge } from "../../atoms/Badge/Badge";
import { Icon } from "../../atoms/Icon/Icon";
import { ScrollArea } from "../../atoms/ScrollArea/ScrollArea";
import { useGetConfigQuery, useSearchQuery } from "../../../store/vaultApi";
import { useAppSelector, useAppDispatch } from "../../../hooks/redux";
import { setSearchQuery, setActiveNote } from "../../../store/uiSlice";

export interface SearchPageProps {
  // No required external props — navigation is handled internally
}

export const SearchPage: React.FC<SearchPageProps> = () => {
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const urlQuery = searchParams.get("q") ?? "";
  const query = useAppSelector((s) => s.ui.searchQuery);
  const { data: config } = useGetConfigQuery();

  // Sync URL → Redux on mount / URL change
  useEffect(() => {
    if (urlQuery !== query) {
      dispatch(setSearchQuery(urlQuery));
    }
  }, [urlQuery]); // eslint-disable-line react-hooks/exhaustive-deps

  // Update page title
  useEffect(() => {
    const siteTitle = config?.pageTitle || config?.vaultName || "Retro Obsidian Publish";
    document.title = `Search${query ? `: ${query}` : ""} — ${siteTitle}`;
  }, [config?.pageTitle, config?.vaultName, query]);

  const { data: results, isFetching } = useSearchQuery(query, {
    skip: query.trim().length < 2,
  });

  const handleSearch = useCallback(
    (q: string) => {
      dispatch(setSearchQuery(q));
      setSearchParams(q ? { q } : {}, { replace: true });
    },
    [dispatch, setSearchParams]
  );

  const handleSelectNote = useCallback(
    (slug: string) => {
      dispatch(setActiveNote(slug));
      navigate(`/note/${slug}`);
    },
    [dispatch, navigate]
  );

  const handleTagClick = useCallback(
    (tag: string) => {
      dispatch(setSearchQuery("#" + tag));
      setSearchParams({ q: "#" + tag }, { replace: true });
    },
    [dispatch, setSearchParams]
  );

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="border-b border-[var(--color-ink)] p-3 bg-[var(--color-panel)] shrink-0">
        <div className="flex items-center gap-2 mb-2">
          <Icon name="search" size={14} />
          <span className="text-xs font-bold uppercase tracking-widest">Search Vault</span>
          {results && (
            <Badge variant="muted" className="ml-auto">
              {results.length} result{results.length !== 1 ? "s" : ""}
            </Badge>
          )}
        </div>
        <SearchBar
          onSearch={handleSearch}
          value={query}
          onChange={(q) => dispatch(setSearchQuery(q))}
          autoFocus
          debounceMs={300}
        />
      </div>

      {/* Results */}
      <ScrollArea className="flex-1 p-2">
        {query.trim().length < 2 ? (
          <div className="flex flex-col items-center justify-center py-16 gap-2 text-[var(--color-muted-foreground)]">
            <Icon name="search" size={32} strokeWidth={1} />
            <p className="text-xs">Type at least 2 characters to search</p>
          </div>
        ) : isFetching ? (
          <div className="flex items-center justify-center py-8 gap-2 text-[var(--color-muted-foreground)] text-xs">
            <Icon name="search" size={14} className="animate-pulse" />
            Searching…
          </div>
        ) : results && results.length > 0 ? (
          <div className="flex flex-col gap-2">
            {results.map((r) => (
              <NoteCard
                key={r.slug}
                slug={r.slug}
                title={r.title}
                excerpt={r.excerpt}
                tags={r.tags}
                onClick={handleSelectNote}
                onTagClick={handleTagClick}
              />
            ))}
          </div>
        ) : (
          <div className="flex flex-col items-center justify-center py-16 gap-2 text-[var(--color-muted-foreground)]">
            <Icon name="alert" size={24} strokeWidth={1} />
            <p className="text-xs font-bold">No results for &ldquo;{query}&rdquo;</p>
          </div>
        )}
      </ScrollArea>
    </div>
  );
};
