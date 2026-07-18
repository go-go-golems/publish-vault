/**
 * MOLECULE: SidebarNav
 * Design: Retro System 1 — sidebar navigation rendered from a widget-page
 * navigation spec (JS-declared, server-driven). Items dispatch their
 * ActionSpec through the page's action handler; the active item renders
 * inverted like the file tree's active note.
 */
import React from "react";
import { clsx } from "clsx";
import { Caption } from "../../foundation/Caption/Caption";
import type { PageNavigationSpec, PageNavigationItemSpec } from "../../../widgets/shell";

export interface SidebarNavProps {
  navigation: PageNavigationSpec;
  onItemSelect: (item: PageNavigationItemSpec) => void;
  className?: string;
}

function labelText(value: unknown): string {
  if (value == null) return "";
  if (typeof value === "object" && "kind" in (value as object)) {
    const node = value as { kind?: unknown; text?: unknown };
    return node.kind === "text" ? String(node.text ?? "") : "";
  }
  return String(value);
}

export const SidebarNav: React.FC<SidebarNavProps> = ({
  navigation,
  onItemSelect,
  className,
}) => (
  <nav
    aria-label={navigation.ariaLabel ?? "Page navigation"}
    className={clsx(
      "h-full overflow-y-auto retro-scroll border-r border-[var(--color-ink)] bg-[var(--color-panel)] flex flex-col gap-3 py-3",
      className
    )}
  >
    {navigation.brand != null && (
      <div className="px-3 text-sm font-bold text-[var(--color-ink)]">
        {labelText(navigation.brand)}
      </div>
    )}
    {navigation.sections.map(section => (
      <div key={section.id} className="flex flex-col gap-1">
        <Caption as="h3" className="px-3 m-0">
          {labelText(section.label)}
        </Caption>
        <ul className="m-0 p-0 list-none">
          {section.items.map(item => {
            const active = navigation.activeItemId === item.id;
            return (
              <li key={item.id}>
                <button
                  type="button"
                  disabled={item.disabled}
                  aria-current={active ? "page" : undefined}
                  className={clsx(
                    "retro-tree-item w-full text-left bg-transparent border-0",
                    active && "active",
                    item.disabled && "opacity-50 cursor-not-allowed"
                  )}
                  onClick={() => onItemSelect(item)}
                >
                  <span className="truncate">{labelText(item.label)}</span>
                  {item.badge != null && (
                    <span className="retro-badge ml-auto">{labelText(item.badge)}</span>
                  )}
                </button>
              </li>
            );
          })}
        </ul>
      </div>
    ))}
  </nav>
);
