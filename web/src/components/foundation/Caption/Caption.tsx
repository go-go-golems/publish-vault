/**
 * FOUNDATION: Caption
 * Design: Retro System 1 — small uppercase label used for section headings,
 * metadata rows, and panel titles.
 */
import React from "react";
import { clsx } from "clsx";

export interface CaptionProps {
  as?: "span" | "h3" | "h4" | "div";
  className?: string;
  children: React.ReactNode;
}

export const Caption: React.FC<CaptionProps> = ({
  as: Tag = "span",
  className,
  children,
}) => (
  <Tag
    className={clsx(
      "text-[10px] font-bold uppercase tracking-widest text-[var(--color-muted-foreground)]",
      className
    )}
  >
    {children}
  </Tag>
);
