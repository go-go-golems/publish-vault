/**
 * MOLECULE: NoteCard
 * Design: Retro System 1 — bordered card with title, excerpt, tags, date.
 * Used in search results and note lists.
 */
import React from "react";
import { clsx } from "clsx";
import { Tag } from "../../atoms/Tag/Tag";
import { Icon } from "../../atoms/Icon/Icon";

export interface NoteCardProps {
  slug: string;
  title: string;
  excerpt?: string;
  tags?: string[];
  modTime?: string;
  onClick?: (slug: string) => void;
  active?: boolean;
  className?: string;
}

export const NoteCard: React.FC<NoteCardProps> = ({
  slug,
  title,
  excerpt,
  tags,
  modTime,
  onClick,
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
          {(tags?.length || modTime) && (
            <div className="flex items-center gap-2 mt-1.5 flex-wrap">
              {tags?.slice(0, 3).map((t) => (
                <Tag
                  key={t}
                  label={t}
                  className={active ? "border-[var(--color-paper)]/60 text-[var(--color-paper)]/80" : ""}
                />
              ))}
              {modTime && (
                <span
                  className={clsx(
                    "text-[10px] ml-auto flex items-center gap-0.5",
                    active ? "text-[var(--color-paper)]/60" : "text-[var(--color-muted-foreground)]"
                  )}
                >
                  <Icon name="clock" size={9} />
                  {modTime}
                </span>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
