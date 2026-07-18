/**
 * Page-shell resolution for widget pages.
 *
 * Interprets the v3 `page.shell` spec (kind app/none/root-owned with a
 * navigation spec) — semantics ported from rag-evaluation-system
 * packages/rag-evaluation-site src/app/App.tsx resolvePageShell (v0.1.7),
 * adapted to publish-vault's defaults: a page with no shell gets the vault
 * chrome (VaultLayout + file tree), NOT an auto-generated app shell.
 * Superseded by the shared headless package when rag-evaluation-system#28
 * ships one.
 */
import type { ActionSpec, RenderableValue } from "./ir";

export interface PageNavigationItemSpec {
  id: string;
  label: RenderableValue;
  action?: ActionSpec;
  icon?: RenderableValue;
  badge?: RenderableValue;
  disabled?: boolean;
}

export interface PageNavigationSectionSpec {
  id: string;
  label: RenderableValue;
  items: PageNavigationItemSpec[];
}

export interface PageNavigationSpec {
  placement: "top" | "sidebar";
  brand?: RenderableValue;
  ariaLabel?: string;
  activeItemId?: string;
  sidebarWidth?: number;
  sections: PageNavigationSectionSpec[];
}

export type PvPageShell =
  /** Default: VaultLayout chrome with the vault file tree. */
  | { kind: "vault" }
  /** Page declares its own navigation; sidebar placement replaces the tree. */
  | { kind: "app"; navigation: PageNavigationSpec }
  /** Full-viewport page owns its chrome (v1: rendered like "vault"). */
  | { kind: "none" };

export function resolvePageShell(shell: unknown): PvPageShell {
  if (!shell || typeof shell !== "object") return { kind: "vault" };
  const spec = shell as { kind?: unknown; navigation?: unknown };
  if (spec.kind === "none" || spec.kind === "root-owned") return { kind: "none" };
  if (spec.kind === "app" && spec.navigation && typeof spec.navigation === "object") {
    const nav = spec.navigation as Partial<PageNavigationSpec>;
    return {
      kind: "app",
      navigation: {
        placement: nav.placement === "sidebar" ? "sidebar" : "top",
        brand: nav.brand,
        ariaLabel: nav.ariaLabel,
        activeItemId: nav.activeItemId,
        sidebarWidth: nav.sidebarWidth,
        sections: Array.isArray(nav.sections) ? nav.sections : [],
      },
    };
  }
  return { kind: "vault" };
}
