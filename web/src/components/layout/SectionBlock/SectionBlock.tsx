/**
 * LAYOUT: SectionBlock
 * Design: Retro System 1 — titled content section.
 *
 * Variants:
 *   - "article" (default): mirrors note layout — an underlined heading like
 *     .note-prose h2, caption as muted text, no box chrome. Top-level page
 *     sections use this so widget pages read like articles.
 *   - "window": classic retro window chrome (inverted title bar, border,
 *     drop shadow) for nested panel-like groupings.
 */
import React from "react";
import { clsx } from "clsx";

export interface SectionBlockProps {
  title: string;
  caption?: string;
  variant?: "article" | "window";
  actions?: React.ReactNode;
  className?: string;
  children: React.ReactNode;
}

export const SectionBlock: React.FC<SectionBlockProps> = ({
  title,
  caption,
  variant = "article",
  actions,
  className,
  children,
}) => {
  if (variant === "window") {
    return (
      <section className={clsx("retro-window", className)}>
        <div className="retro-window-title">
          <span>{title}</span>
          <div className="retro-titlebar-stripes" />
          {actions}
        </div>
        <div className="p-3 flex flex-col gap-2">
          {caption && (
            <p className="text-[11px] text-[var(--color-muted-foreground)] m-0">
              {caption}
            </p>
          )}
          {children}
        </div>
      </section>
    );
  }

  return (
    <section className={clsx("flex flex-col gap-2", className)}>
      <div className="flex items-baseline gap-2 border-b border-dotted border-[var(--color-chrome)] pb-1">
        <h2 className="text-xl font-bold text-[var(--color-ink)] leading-tight m-0">
          {title}
        </h2>
        {actions && <div className="ml-auto">{actions}</div>}
      </div>
      {caption && (
        <p className="text-[11px] text-[var(--color-muted-foreground)] m-0">
          {caption}
        </p>
      )}
      {children}
    </section>
  );
};
