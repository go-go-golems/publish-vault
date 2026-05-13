/**
 * ATOM: Button
 * Design: Retro System 1 — raised 1px border, hard shadow, square corners.
 * Variants: default (raised), primary (inverted), ghost (no border), danger.
 */
import React from "react";
import { clsx } from "clsx";

export type ButtonVariant = "default" | "primary" | "ghost" | "danger";
export type ButtonSize = "sm" | "md" | "lg";

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  icon?: React.ReactNode;
  iconPosition?: "left" | "right";
}

const variantClasses: Record<ButtonVariant, string> = {
  default: "retro-btn",
  primary: "retro-btn retro-btn-primary",
  ghost:
    "inline-flex items-center gap-1 px-2 py-0.5 text-xs font-bold text-[var(--color-ink)] hover:underline cursor-pointer",
  danger:
    "retro-btn border-[var(--color-destructive-accent)] text-[var(--color-destructive-accent)] hover:bg-[var(--color-destructive-accent)] hover:text-[var(--color-paper)]",
};

const sizeClasses: Record<ButtonSize, string> = {
  sm: "text-[11px] px-2 py-0.5",
  md: "text-xs px-[10px] py-[3px]",
  lg: "text-sm px-4 py-1",
};

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      variant = "default",
      size = "md",
      icon,
      iconPosition = "left",
      className,
      children,
      disabled,
      ...props
    },
    ref
  ) => {
    return (
      <button
        ref={ref}
        disabled={disabled}
        className={clsx(
          variantClasses[variant],
          sizeClasses[size],
          disabled && "opacity-40 cursor-not-allowed pointer-events-none",
          className
        )}
        {...props}
      >
        {icon && iconPosition === "left" && (
          <span className="inline-flex shrink-0">{icon}</span>
        )}
        {children}
        {icon && iconPosition === "right" && (
          <span className="inline-flex shrink-0">{icon}</span>
        )}
      </button>
    );
  }
);

Button.displayName = "Button";
