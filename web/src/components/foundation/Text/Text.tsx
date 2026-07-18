/**
 * FOUNDATION: Text
 * Design: Retro System 1 — body text primitive with size and tone variants.
 * Use for any prose-adjacent text that is not a heading; keeps font sizing
 * and ink tones consistent across tiers.
 */
import React from "react";
import { clsx } from "clsx";

export type TextSize = "xs" | "sm" | "md";
export type TextTone = "default" | "muted" | "danger" | "success";

const SIZE_CLASSES: Record<TextSize, string> = {
  xs: "text-[11px]",
  sm: "text-xs",
  md: "text-sm",
};

const TONE_CLASSES: Record<TextTone, string> = {
  default: "text-[var(--color-ink)]",
  muted: "text-[var(--color-muted-foreground)]",
  danger: "text-[var(--color-destructive-accent)]",
  success: "text-[var(--color-tag)]",
};

export interface TextProps {
  size?: TextSize;
  tone?: TextTone;
  bold?: boolean;
  as?: "span" | "p" | "div";
  className?: string;
  children: React.ReactNode;
}

export const Text: React.FC<TextProps> = ({
  size = "md",
  tone = "default",
  bold = false,
  as: Tag = "span",
  className,
  children,
}) => (
  <Tag
    className={clsx(
      SIZE_CLASSES[size],
      TONE_CLASSES[tone],
      bold && "font-bold",
      className
    )}
  >
    {children}
  </Tag>
);
