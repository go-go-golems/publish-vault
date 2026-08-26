/**
 * MOLECULE: NoteCard
 * Design: Retro System 1 — bordered card with title, excerpt, tags, date.
 * Used in search results and note lists.
 */
import React from "react";
import { clsx } from "clsx";
import { Tag } from "../../atoms/Tag/Tag";
import { Icon } from "../../atoms/Icon/Icon";
import type { SearchResultDate } from "../../../types";

export interface NoteCardProps {
  slug: string;
  title: string;
  excerpt?: string;
  tags?: string[];
  modTime?: string;
  /** Authored display date (created/updated) from the search result. */
  date?: SearchResultDate;
  /** Vault-relative path, shown as a small breadcrumb. */
  path?: string;
  onClick?: (slug: string) => void;
  onTagClick?: (tag: string) => void;
  active?: boolean;
  className?: string;
}

function dateLabel(date: SearchResultDate): string {
  const kind = date.kind === "updated" ? "Updated" : "Created";
  // Date precision shows the literal; timestamp shows the UTC date for the chip.
  const shown = date.precision === "timestamp" ? date.value.slice(0, 10) : date.value;
  return `${kind} ${shown}`;
}

export const NoteCard: React.FC<NoteCardProps> = ({
  slug,
  title,
  excerpt,
  tags,
  modTime,
  date,
  path,
  onClick,
  onTagClick,
  active,
  className,
}) => {
  return (
    <div
      role="button"
      tabIndex={0}
      onClick={() => onClick?.(slug)}
      onKeyDown={(e) => e.key === "Enter" && onClick?.(slug)}
      className={clsx(
        "retro-window p-3 cursor-pointer transition-none select-none",
        "hover:bg-[var(--color-panel)]",
        active && "bg-[var(--color-ink)] text-[var(--color-paper)]",
        className
      )}
    >
      <div className="flex items-start gap-2">
        <Icon
          name="file"
          size={13}
          className={active ? "text-[var(--color-paper)]" : "text-[var(--color-muted-foreground)]"}
        />
        <div className="flex-1 min-w-0">
          {path && (
            <div
              className={clsx(
                "text-[10px] truncate mb-0.5",
                active ? "text-[var(--color-paper)]/60" : "text-[var(--color-muted-foreground)]"
              )}
            >
              {path}
            </div>
          )}
          <h3
            className={clsx(
              "text-xs font-bold leading-tight truncate",
              active ? "text-[var(--color-paper)]" : "text-[var(--color-ink)]"
            )}
          >
            {title}
          </h3>
          {excerpt && (
            <p
              className={clsx(
                "text-[11px] mt-0.5 line-clamp-2 leading-snug",
                active ? "text-[var(--color-paper)]/80" : "text-[var(--color-muted-foreground)]"
              )}
            >
              {excerpt}
            </p>
          )}
          {(tags?.length || date || modTime) && (
            <div className="flex items-center gap-2 mt-1.5 flex-wrap">
              {tags?.slice(0, 3).map((t) => (
                <Tag
                  key={t}
                  label={t}
                  onClick={
                    onTagClick
                      ? (event) => {
                          event.stopPropagation();
                          onTagClick(t);
                        }
                      : undefined
                  }
                  className={active ? "border-[var(--color-paper)]/60 text-[var(--color-paper)]/80" : ""}
                />
              ))}
              {date ? (
                <time
                  dateTime={date.value}
                  title={date.value}
                  className={clsx(
                    "text-[10px] ml-auto flex items-center gap-0.5",
                    active ? "text-[var(--color-paper)]/60" : "text-[var(--color-muted-foreground)]"
                  )}
                >
                  <Icon name="clock" size={9} />
                  {dateLabel(date)}
                </time>
              ) : modTime ? (
                <span
                  className={clsx(
                    "text-[10px] ml-auto flex items-center gap-0.5",
                    active ? "text-[var(--color-paper)]/60" : "text-[var(--color-muted-foreground)]"
                  )}
                >
                  <Icon name="clock" size={9} />
                  {modTime}
                </span>
              ) : null}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
