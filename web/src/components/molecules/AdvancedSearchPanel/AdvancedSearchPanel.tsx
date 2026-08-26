/**
 * MOLECULE: AdvancedSearchPanel
 * A modal filter panel for advanced search. Holds a draft of the structured
 * filters (tags, tag mode, path prefixes, date field, date range) and calls
 * onApply with a merged SearchRequest. The text query and sort are owned by the
 * SearchPage header; this panel preserves them and resets offset to 0 on apply.
 */
import React, { useEffect, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogClose,
} from "../../ui/dialog";
import type { DateField, SearchRequest, TagMode } from "../../../types";
import { dateOnlyString, parseDateOnly } from "../../../search/searchParams";

export interface AdvancedSearchPanelProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  current: SearchRequest;
  onApply: (req: SearchRequest) => void;
}

function splitList(value: string): string[] {
  return value
    .split(/[,\s]+/)
    .map((s) => s.trim())
    .filter((s) => s.length > 0);
}

export const AdvancedSearchPanel: React.FC<AdvancedSearchPanelProps> = ({
  open,
  onOpenChange,
  current,
  onApply,
}) => {
  const [tagsText, setTagsText] = useState(current.tags.join(", "));
  const [tagMode, setTagMode] = useState<TagMode | "">(current.tagMode);
  const [pathsText, setPathsText] = useState(current.pathPrefixes.join(", "));
  const [dateField, setDateField] = useState<DateField | "">(current.dateField);
  const [dateFrom, setDateFrom] = useState(current.dateFrom ? dateOnlyString(current.dateFrom) : "");
  const [dateTo, setDateTo] = useState(current.dateTo ? dateOnlyString(current.dateTo) : "");

  // Re-sync the draft from the committed request each time the panel opens.
  useEffect(() => {
    if (open) {
      setTagsText(current.tags.join(", "));
      setTagMode(current.tagMode);
      setPathsText(current.pathPrefixes.join(", "));
      setDateField(current.dateField);
      setDateFrom(current.dateFrom ? dateOnlyString(current.dateFrom) : "");
      setDateTo(current.dateTo ? dateOnlyString(current.dateTo) : "");
    }
  }, [open, current]);

  const handleApply = () => {
    const from = dateFrom ? (parseDateOnly(dateFrom) ?? undefined) : undefined;
    const to = dateTo ? (parseDateOnly(dateTo) ?? undefined) : undefined;
    onApply({
      query: current.query,
      tags: splitList(tagsText),
      tagMode: (tagMode || "all") as TagMode,
      pathPrefixes: splitList(pathsText),
      dateField: (dateField || (from || to ? "display" : "")) as DateField,
      dateFrom: from,
      dateTo: to,
      sort: current.sort,
      limit: current.limit,
      // Any filter change resets pagination.
      offset: 0,
    });
    onOpenChange(false);
  };

  const handleClear = () => {
    setTagsText("");
    setTagMode("all");
    setPathsText("");
    setDateField("");
    setDateFrom("");
    setDateTo("");
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Search filters</DialogTitle>
          <DialogDescription>
            Refine results by tags, folder, and authored date. The text query and sort are kept.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-4">
          <label className="flex flex-col gap-1 text-sm">
            <span>Tags</span>
            <input
              type="text"
              value={tagsText}
              onChange={(e) => setTagsText(e.target.value)}
              placeholder="go, performance"
              className="rounded border border-[var(--color-ink)] bg-transparent px-2 py-1"
            />
          </label>

          <label className="flex flex-col gap-1 text-sm">
            <span>Tag match</span>
            <select
              value={tagMode || "all"}
              onChange={(e) => setTagMode(e.target.value as TagMode)}
              className="rounded border border-[var(--color-ink)] bg-transparent px-2 py-1"
            >
              <option value="all">All tags (AND)</option>
              <option value="any">Any tag (OR)</option>
            </select>
          </label>

          <label className="flex flex-col gap-1 text-sm">
            <span>Folder prefixes</span>
            <input
              type="text"
              value={pathsText}
              onChange={(e) => setPathsText(e.target.value)}
              placeholder="research/kb, projects/2026"
              className="rounded border border-[var(--color-ink)] bg-transparent px-2 py-1"
            />
          </label>

          <div className="flex flex-col gap-1 text-sm">
            <span>Date range</span>
            <div className="flex items-center gap-2">
              <select
                value={dateField || "display"}
                onChange={(e) => setDateField(e.target.value as DateField)}
                className="rounded border border-[var(--color-ink)] bg-transparent px-2 py-1"
                aria-label="Date field"
              >
                <option value="display">Display date</option>
                <option value="created">Created</option>
                <option value="updated">Updated</option>
              </select>
              <input
                type="date"
                value={dateFrom}
                onChange={(e) => setDateFrom(e.target.value)}
                aria-label="Date from"
                className="rounded border border-[var(--color-ink)] bg-transparent px-2 py-1"
              />
              <span>to</span>
              <input
                type="date"
                value={dateTo}
                onChange={(e) => setDateTo(e.target.value)}
                aria-label="Date to"
                className="rounded border border-[var(--color-ink)] bg-transparent px-2 py-1"
              />
            </div>
          </div>
        </div>

        <DialogFooter>
          <button
            type="button"
            onClick={handleClear}
            className="rounded border border-[var(--color-ink)] px-3 py-1 text-sm"
          >
            Clear filters
          </button>
          <DialogClose asChild>
            <button
              type="button"
              onClick={handleApply}
              className="rounded bg-[var(--color-ink)] px-3 py-1 text-sm text-[var(--color-paper)]"
            >
              Apply
            </button>
          </DialogClose>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
