/**
 * LAYOUT: Panel
 * Design: Retro System 1 — bordered surface. Variants map to the retro
 * chrome classes: "window" (raised, drop shadow), "inset" (sunken), and
 * "plain" (1px border only).
 */
import React from "react";
import { clsx } from "clsx";

export type PanelVariant = "window" | "inset" | "plain";

const VARIANT_CLASSES: Record<PanelVariant, string> = {
  window: "retro-window",
  inset: "retro-inset",
  plain: "border border-[var(--color-ink)] bg-[var(--color-paper)]",
};

export interface PanelProps {
  variant?: PanelVariant;
  padded?: boolean;
  className?: string;
  children: React.ReactNode;
}

export const Panel: React.FC<PanelProps> = ({
  variant = "plain",
  padded = true,
  className,
  children,
}) => (
  <div className={clsx(VARIANT_CLASSES[variant], padded && "p-3", className)}>
    {children}
  </div>
);
