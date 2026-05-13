/**
 * ATOM: Badge
 * Design: Retro System 1 — inverted (black bg, white text), uppercase, tiny.
 * Used for counts, status indicators, and labels.
 */
import React from "react";
import { clsx } from "clsx";

export type BadgeVariant = "default" | "accent" | "muted";

export interface BadgeProps {
  children: React.ReactNode;
  variant?: BadgeVariant;
  className?: string;
}

const variantClasses: Record<BadgeVariant, string> = {
  default: "retro-badge",
  accent:
    "retro-badge bg-[var(--color-active)] border-[var(--color-active)] text-white",
  muted:
    "retro-badge bg-[var(--color-muted)] border-[var(--color-muted)] text-[var(--color-ink)]",
};

export const Badge: React.FC<BadgeProps> = ({
  children,
  variant = "default",
  className,
}) => (
  <span className={clsx(variantClasses[variant], className)}>{children}</span>
);
