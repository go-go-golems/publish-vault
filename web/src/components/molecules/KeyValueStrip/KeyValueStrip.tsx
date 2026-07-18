/**
 * MOLECULE: KeyValueStrip
 * Design: Retro System 1 — inline metric chip: uppercase label over a bold
 * value in a sunken panel. Widget pages emit one strip per section metric.
 */
import React from "react";
import { clsx } from "clsx";

export interface KeyValueStripItem {
  key?: string;
  label: React.ReactNode;
  value: React.ReactNode;
}

export interface KeyValueStripProps {
  items: KeyValueStripItem[];
  className?: string;
}

export const KeyValueStrip: React.FC<KeyValueStripProps> = ({ items, className }) => (
  <div className={clsx("inline-flex flex-wrap gap-2", className)}>
    {items.map((item, index) => (
      <div key={item.key ?? index} className="retro-inset px-3 py-1.5 inline-flex flex-col gap-0.5">
        <span className="text-[10px] font-bold uppercase tracking-widest text-[var(--color-muted-foreground)]">
          {item.label}
        </span>
        <span className="text-sm font-bold text-[var(--color-ink)]">{item.value}</span>
      </div>
    ))}
  </div>
);
