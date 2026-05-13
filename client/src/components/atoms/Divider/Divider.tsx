/**
 * ATOM: Divider
 * Design: Retro System 1 — 1px solid ink line, optional label.
 */
import React from "react";
import { clsx } from "clsx";

export interface DividerProps {
  label?: string;
  className?: string;
  orientation?: "horizontal" | "vertical";
}

export const Divider: React.FC<DividerProps> = ({
  label,
  className,
  orientation = "horizontal",
}) => {
  if (orientation === "vertical") {
    return (
      <div
        className={clsx(
          "w-px bg-[var(--color-ink)] self-stretch",
          className
        )}
      />
    );
  }

  if (label) {
    return (
      <div className={clsx("flex items-center gap-2 my-2", className)}>
        <hr className="flex-1 retro-divider" />
        <span className="text-[10px] font-bold uppercase tracking-widest text-[var(--color-muted-foreground)]">
          {label}
        </span>
        <hr className="flex-1 retro-divider" />
      </div>
    );
  }

  return <hr className={clsx("retro-divider my-2", className)} />;
};
