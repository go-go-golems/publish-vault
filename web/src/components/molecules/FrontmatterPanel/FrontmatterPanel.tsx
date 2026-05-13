/**
 * MOLECULE: FrontmatterPanel
 * Design: Retro System 1 — key/value table, monospace keys, collapsible.
 * Renders YAML frontmatter fields except title and tags (shown elsewhere).
 */
import React, { useState } from "react";
import { clsx } from "clsx";
import { Icon } from "../../atoms/Icon/Icon";
import { Tag } from "../../atoms/Tag/Tag";
import { Divider } from "../../atoms/Divider/Divider";

const EXCLUDED_KEYS = new Set(["title", "tags"]);

export interface FrontmatterPanelProps {
  frontmatter: Record<string, unknown>;
  tags?: string[];
  modTime?: string;
  className?: string;
  onTagClick?: (tag: string) => void;
}

export const FrontmatterPanel: React.FC<FrontmatterPanelProps> = ({
  frontmatter,
  tags,
  modTime,
  className,
  onTagClick,
}) => {
  const [collapsed, setCollapsed] = useState(false);

  const entries = Object.entries(frontmatter).filter(
    ([k]) => !EXCLUDED_KEYS.has(k)
  );

  return (
    <div className={clsx("text-[11px]", className)}>
      <button
        type="button"
        className="flex items-center gap-1 font-bold uppercase tracking-widest text-[10px] text-[var(--color-muted-foreground)] mb-1 hover:text-[var(--color-ink)]"
        onClick={() => setCollapsed((v) => !v)}
      >
        <Icon name={collapsed ? "chevron-right" : "chevron-down"} size={10} />
        Properties
      </button>

      {!collapsed && (
        <div className="retro-inset p-2 space-y-1">
          {modTime && (
            <div className="flex gap-2">
              <span className="font-mono text-[var(--color-muted-foreground)] w-20 shrink-0">modified</span>
              <span className="flex items-center gap-1 text-[var(--color-ink)]">
                <Icon name="clock" size={10} />
                {formatValue(modTime)}
              </span>
            </div>
          )}

          {tags && tags.length > 0 && (
            <div className="flex gap-2 items-start">
              <span className="font-mono text-[var(--color-muted-foreground)] w-20 shrink-0">tags</span>
              <div className="flex flex-wrap gap-1">
                {tags.map((t) => (
                  <Tag key={t} label={t} onClick={onTagClick ? () => onTagClick(t) : undefined} />
                ))}
              </div>
            </div>
          )}

          {entries.map(([key, value]) => (
            <div key={key} className="flex gap-2">
              <span className="font-mono text-[var(--color-muted-foreground)] w-20 shrink-0 truncate">
                {key}
              </span>
              <span className="text-[var(--color-ink)] break-all">
                {formatValue(value)}
              </span>
            </div>
          ))}

          {entries.length === 0 && !tags?.length && !modTime && (
            <span className="text-[var(--color-muted-foreground)] italic">No properties</span>
          )}
        </div>
      )}
    </div>
  );
};

function formatValue(v: unknown): string {
  if (v === null || v === undefined) return "—";
  if (Array.isArray(v)) return v.join(", ");
  // Handle Date objects (js-yaml parses YAML dates as JS Date)
  if (v instanceof Date) {
    return v.toLocaleDateString("en-US", { year: "numeric", month: "short", day: "numeric" });
  }
  if (typeof v === "object") return JSON.stringify(v);
  const s = String(v);
  // Format ISO date strings to a readable short form
  if (/^\d{4}-\d{2}-\d{2}(T|$)/.test(s)) {
    try {
      return new Date(s).toLocaleDateString("en-US", {
        year: "numeric",
        month: "short",
        day: "numeric",
      });
    } catch {
      return s.slice(0, 10);
    }
  }
  return s;
}
