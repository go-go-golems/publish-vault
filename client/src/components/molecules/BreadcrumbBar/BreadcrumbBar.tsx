/**
 * MOLECULE: BreadcrumbBar
 * Design: Retro System 1 — path segments separated by ›, clickable.
 */
import React from "react";
import { clsx } from "clsx";
import { Icon } from "../../atoms/Icon/Icon";

export interface BreadcrumbSegment {
  label: string;
  slug?: string;
}

export interface BreadcrumbBarProps {
  segments: BreadcrumbSegment[];
  onNavigate?: (slug: string) => void;
  className?: string;
}

export const BreadcrumbBar: React.FC<BreadcrumbBarProps> = ({
  segments,
  onNavigate,
  className,
}) => (
  <nav
    aria-label="Breadcrumb"
    className={clsx(
      "flex items-center gap-0 text-[11px] font-bold tracking-wide",
      className
    )}
  >
    <Icon name="home" size={11} className="text-[var(--color-muted-foreground)] mr-1" />
    {segments.map((seg, i) => {
      const isLast = i === segments.length - 1;
      return (
        <React.Fragment key={seg.slug ?? seg.label}>
          {i > 0 && (
            <span className="mx-1 text-[var(--color-muted-foreground)]">›</span>
          )}
          {seg.slug && !isLast ? (
            <button
              type="button"
              className="text-[var(--color-link)] hover:underline cursor-pointer"
              onClick={() => onNavigate?.(seg.slug!)}
            >
              {seg.label}
            </button>
          ) : (
            <span
              className={clsx(
                isLast
                  ? "text-[var(--color-ink)] font-bold"
                  : "text-[var(--color-muted-foreground)]"
              )}
            >
              {seg.label}
            </span>
          )}
        </React.Fragment>
      );
    })}
  </nav>
);
