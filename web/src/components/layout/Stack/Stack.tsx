/**
 * LAYOUT: Stack
 * Vertical flex container with a fixed gap scale. Structural only — no
 * borders, colors, or domain semantics.
 */
import React from "react";
import { clsx } from "clsx";

export type StackGap = "none" | "xs" | "sm" | "md" | "lg";

const GAP_CLASSES: Record<StackGap, string> = {
  none: "gap-0",
  xs: "gap-1",
  sm: "gap-2",
  md: "gap-4",
  lg: "gap-6",
};

export interface StackProps {
  gap?: StackGap;
  align?: "start" | "center" | "stretch";
  className?: string;
  children: React.ReactNode;
}

export const Stack: React.FC<StackProps> = ({
  gap = "md",
  align = "stretch",
  className,
  children,
}) => (
  <div
    className={clsx(
      "flex flex-col",
      GAP_CLASSES[gap],
      align === "start" && "items-start",
      align === "center" && "items-center",
      className
    )}
  >
    {children}
  </div>
);
