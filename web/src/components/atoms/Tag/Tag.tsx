/**
 * ATOM: Tag
 * Design: Retro System 1 — green border pill, no fill, lowercase.
 * Used for note tags. Optionally clickable to filter by tag.
 */
import React from "react";
import { clsx } from "clsx";

export interface TagProps {
  label: string;
  onClick?: () => void;
  className?: string;
  active?: boolean;
}

export const Tag: React.FC<TagProps> = ({ label, onClick, className, active }) => {
  const base = clsx(
    "retro-tag",
    active && "bg-[var(--color-tag)] text-white",
    onClick ? "cursor-pointer" : "cursor-default",
    className
  );

  if (onClick) {
    return (
      <button type="button" className={base} onClick={onClick}>
        #{label}
      </button>
    );
  }
  return <span className={base}>#{label}</span>;
};
