/**
 * ATOM: Input
 * Design: Retro System 1 — inset border, paper background, no radius.
 */
import React from "react";
import { clsx } from "clsx";

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
  icon?: React.ReactNode;
}

export const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ label, error, icon, className, id, ...props }, ref) => {
    const inputId = id ?? label?.toLowerCase().replace(/\s+/g, "-");
    return (
      <div className="flex flex-col gap-0.5">
        {label && (
          <label
            htmlFor={inputId}
            className="text-[11px] font-bold uppercase tracking-widest text-[var(--color-ink)]"
          >
            {label}
          </label>
        )}
        <div className="relative flex items-center">
          {icon && (
            <span className="absolute left-2 text-[var(--color-muted-foreground)] pointer-events-none">
              {icon}
            </span>
          )}
          <input
            ref={ref}
            id={inputId}
            className={clsx(
              "retro-search w-full",
              icon && "pl-7",
              error && "border-[var(--color-destructive-accent)]",
              className
            )}
            {...props}
          />
        </div>
        {error && (
          <span className="text-[11px] text-[var(--color-destructive-accent)] font-bold">
            {error}
          </span>
        )}
      </div>
    );
  }
);

Input.displayName = "Input";
