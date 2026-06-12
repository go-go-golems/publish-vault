/**
 * MOLECULE: SearchBar
 * Design: Retro System 1 — inset input with search icon, keyboard shortcut hint.
 * Debounces input and calls onSearch with the query string.
 */
import React, { useCallback, useEffect, useRef, useState } from "react";
import { clsx } from "clsx";
import { Icon } from "../../atoms/Icon/Icon";

export interface SearchBarProps {
  onSearch: (query: string) => void;
  placeholder?: string;
  debounceMs?: number;
  className?: string;
  autoFocus?: boolean;
  initialValue?: string;
  /** Controlled value — when provided, the component is fully controlled */
  value?: string;
  onChange?: (value: string) => void;
}

export const SearchBar: React.FC<SearchBarProps> = ({
  onSearch,
  placeholder = "Search vault…",
  debounceMs = 250,
  className,
  autoFocus,
  initialValue = "",
  value: controlledValue,
  onChange,
}) => {
  const isControlled = controlledValue !== undefined;
  const [internalValue, setInternalValue] = useState(initialValue);
  const value = isControlled ? controlledValue : internalValue;
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const q = e.target.value;
      if (!isControlled) {
        setInternalValue(q);
      }
      onChange?.(q);
      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = setTimeout(() => onSearch(q), debounceMs);
    },
    [onSearch, debounceMs, isControlled, onChange]
  );

  const handleClear = useCallback(() => {
    if (!isControlled) {
      setInternalValue("");
    }
    onChange?.("");
    onSearch("");
  }, [onSearch, isControlled, onChange]);

  // Keyboard shortcut: / to focus
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (
        e.key === "/" &&
        !(e.target instanceof HTMLInputElement) &&
        !(e.target instanceof HTMLTextAreaElement)
      ) {
        e.preventDefault();
        document.getElementById("vault-search")?.focus();
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, []);

  return (
    <div className={clsx("relative flex items-center", className)}>
      <span className="absolute left-2 text-[var(--color-muted-foreground)] pointer-events-none">
        <Icon name="search" size={12} />
      </span>
      <input
        id="vault-search"
        type="search"
        value={value}
        onChange={handleChange}
        placeholder={placeholder}
        autoFocus={autoFocus}
        className="retro-search pl-6 pr-12"
        autoComplete="off"
        spellCheck={false}
      />
      {value ? (
        <button
          type="button"
          onClick={handleClear}
          className="absolute right-2 text-[var(--color-muted-foreground)] hover:text-[var(--color-ink)] p-0.5"
          aria-label="Clear search"
        >
          <Icon name="close" size={11} />
        </button>
      ) : (
        <kbd className="absolute right-2 text-[9px] font-bold text-[var(--color-muted-foreground)] border border-[var(--color-muted)] px-1 pointer-events-none">
          /
        </kbd>
      )}
    </div>
  );
};
