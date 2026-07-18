/**
 * FOUNDATION: CodeText
 * Design: Retro System 1 — inline monospace text on a sunken panel chip.
 */
import React from "react";
import { clsx } from "clsx";

export interface CodeTextProps {
  className?: string;
  children: React.ReactNode;
}

export const CodeText: React.FC<CodeTextProps> = ({ className, children }) => (
  <code
    className={clsx(
      "bg-[var(--color-panel)] border border-[var(--color-chrome)] px-1 text-[12px]",
      className
    )}
  >
    {children}
  </code>
);
