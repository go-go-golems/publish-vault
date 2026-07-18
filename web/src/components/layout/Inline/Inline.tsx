/**
 * LAYOUT: Inline
 * Horizontal flex container with a fixed gap scale and wrap control.
 * Structural only.
 */
import React from "react";
import { clsx } from "clsx";

export type InlineGap = "none" | "xs" | "sm" | "md" | "lg";

const GAP_CLASSES: Record<InlineGap, string> = {
  none: "gap-0",
  xs: "gap-1",
  sm: "gap-2",
  md: "gap-3",
  lg: "gap-4",
};

export interface InlineProps {
  gap?: InlineGap;
  align?: "start" | "center" | "baseline" | "end";
  wrap?: boolean;
  className?: string;
  children: React.ReactNode;
}

export const Inline: React.FC<InlineProps> = ({
  gap = "sm",
  align = "center",
  wrap = false,
  className,
  children,
}) => (
  <div
    className={clsx(
      "flex flex-row",
      GAP_CLASSES[gap],
      align === "start" && "items-start",
      align === "center" && "items-center",
      align === "baseline" && "items-baseline",
      align === "end" && "items-end",
      wrap && "flex-wrap",
      className
    )}
  >
    {children}
  </div>
);
