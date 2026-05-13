/**
 * ATOM: ScrollArea
 * Design: Retro System 1 — thin retro scrollbar, overflow container.
 */
import React from "react";
import { clsx } from "clsx";

export interface ScrollAreaProps {
  children: React.ReactNode;
  className?: string;
  maxHeight?: string;
  horizontal?: boolean;
}

export const ScrollArea: React.FC<ScrollAreaProps> = ({
  children,
  className,
  maxHeight,
  horizontal = false,
}) => (
  <div
    className={clsx(
      "retro-scroll",
      horizontal && "overflow-x-auto",
      className
    )}
    style={maxHeight ? { maxHeight } : undefined}
  >
    {children}
  </div>
);
