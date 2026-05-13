/**
 * MOLECULE: BacklinkItem
 * Design: Retro System 1 — compact row with link icon, note title, excerpt snippet.
 */
import React from "react";
import { clsx } from "clsx";
import { Icon } from "../../atoms/Icon/Icon";

export interface BacklinkItemProps {
  slug: string;
  title: string;
  excerpt?: string;
  onClick?: (slug: string) => void;
  className?: string;
}

export const BacklinkItem: React.FC<BacklinkItemProps> = ({
  slug,
  title,
  excerpt,
  onClick,
  className,
}) => (
  <button
    type="button"
    className={clsx(
      "w-full text-left flex items-start gap-2 px-3 py-2",
      "hover:bg-[var(--color-panel)] border-b border-[var(--color-border)] last:border-b-0",
      "transition-none",
      className
    )}
    onClick={() => onClick?.(slug)}
  >
    <Icon
      name="arrow-left"
      size={11}
      className="mt-0.5 shrink-0 text-[var(--color-link)]"
    />
    <div className="min-w-0">
      <span className="text-xs font-bold text-[var(--color-link)] block truncate">
        {title}
      </span>
      {excerpt && (
        <span className="text-[11px] text-[var(--color-muted-foreground)] line-clamp-1 block mt-0.5">
          {excerpt}
        </span>
      )}
    </div>
  </button>
);
