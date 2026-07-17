/**
 * LAYOUT: SectionBlock
 * Design: Retro System 1 — titled content section: window-title caption bar
 * followed by the section body. The structural unit widget pages and panels
 * compose from.
 */
import React from "react";
import { clsx } from "clsx";

export interface SectionBlockProps {
  title: string;
  caption?: string;
  actions?: React.ReactNode;
  className?: string;
  children: React.ReactNode;
}

export const SectionBlock: React.FC<SectionBlockProps> = ({
  title,
  caption,
  actions,
  className,
  children,
}) => (
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
