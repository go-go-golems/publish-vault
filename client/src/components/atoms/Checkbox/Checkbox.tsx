/**
 * ATOM: Checkbox
 * Design: Retro System 1 — square checkbox, bold X mark when checked.
 */
import React from "react";
import { clsx } from "clsx";

export interface CheckboxProps {
  checked?: boolean;
  onChange?: (checked: boolean) => void;
  label?: string;
  disabled?: boolean;
  className?: string;
  id?: string;
}

export const Checkbox: React.FC<CheckboxProps> = ({
  checked,
  onChange,
  label,
  disabled,
  className,
  id,
}) => {
  const inputId = id ?? label?.toLowerCase().replace(/\s+/g, "-");
  return (
    <label
      htmlFor={inputId}
      className={clsx(
        "inline-flex items-center gap-2 text-xs cursor-pointer select-none",
        disabled && "opacity-40 cursor-not-allowed",
        className
      )}
    >
      <input
        type="checkbox"
        id={inputId}
        checked={checked}
        disabled={disabled}
        onChange={(e) => onChange?.(e.target.checked)}
        className="sr-only"
      />
      <span
        className={clsx(
          "w-4 h-4 border border-[var(--color-ink)] flex items-center justify-center shrink-0",
          checked ? "bg-[var(--color-ink)]" : "bg-[var(--color-paper)]"
        )}
      >
        {checked && (
          <svg width="10" height="10" viewBox="0 0 10 10" fill="none">
            <path d="M1.5 5L4 7.5L8.5 2.5" stroke="#f0ede8" strokeWidth="1.5" strokeLinecap="square" />
          </svg>
        )}
      </span>
      {label && <span className="font-medium">{label}</span>}
    </label>
  );
};
