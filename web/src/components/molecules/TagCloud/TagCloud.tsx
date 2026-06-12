/**
 * MOLECULE: TagCloud
 * Design: Retro System 1 — collection of clickable tags sorted by count.
 * Used on the search page when no query is active.
 */
import React from "react";
import { clsx } from "clsx";
import { Tag } from "../../atoms/Tag/Tag";
import { Icon } from "../../atoms/Icon/Icon";
import type { TagCount } from "../../../types";

export interface TagCloudProps {
  tags: TagCount[];
  onTagClick?: (tag: string) => void;
  className?: string;
}

export const TagCloud: React.FC<TagCloudProps> = ({
  tags,
  onTagClick,
  className,
}) => {
  if (tags.length === 0) {
    return (
      <div className={clsx("text-[11px] text-[var(--color-muted-foreground)] italic px-1", className)}>
        No tags found
      </div>
    );
  }

  // Sort by count descending, then alphabetically
  const sorted = [...tags].sort((a, b) => {
    if (b.count !== a.count) return b.count - a.count;
    return a.tag.localeCompare(b.tag);
  });

  return (
    <div className={clsx("flex flex-col gap-2", className)}>
      <div className="flex items-center gap-1 text-[10px] font-bold uppercase tracking-widest text-[var(--color-muted-foreground)]">
        <Icon name="hash" size={10} />
        Browse by Tag
      </div>
      <div className="flex flex-wrap gap-1.5">
        {sorted.map((tc) => (
          <Tag
            key={tc.tag}
            label={`${tc.tag} (${tc.count})`}
            onClick={onTagClick ? () => onTagClick(tc.tag) : undefined}
          />
        ))}
      </div>
    </div>
  );
};
