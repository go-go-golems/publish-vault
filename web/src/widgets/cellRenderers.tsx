/**
 * DataTable cell renderers — interpret defunctionalized CellSpec data.
 * Ported from rag-evaluation-system packages/rag-evaluation-site
 * src/widgets/cellRenderers.tsx (v0.1.7), restyled with publish-vault's
 * retro components/classes instead of the rag foundation set.
 */
import type { ReactNode } from "react";
import type { ActionSpec, CellSpec, JsonObject, RenderableValue, RowKeySpec, WidgetNode } from "./ir";
import { isWidgetNode } from "./ir";

export type RenderWidgetNode = (node: WidgetNode) => ReactNode;

const STATUS_TONES: Record<string, string> = {
  ready: "text-[var(--color-tag)]",
  published: "text-[var(--color-tag)]",
  done: "text-[var(--color-tag)]",
  draft: "text-[var(--color-muted-foreground)]",
  pending: "text-[var(--color-muted-foreground)]",
  review: "text-[#cc7700]",
  error: "text-[var(--color-destructive-accent)]",
  failed: "text-[var(--color-destructive-accent)]",
};

const CAPTION_TONES: Record<string, string> = {
  default: "text-[var(--color-ink)]",
  muted: "text-[var(--color-muted-foreground)]",
  danger: "text-[var(--color-destructive-accent)]",
  success: "text-[var(--color-tag)]",
};

export function renderCell(
  spec: CellSpec,
  row: JsonObject,
  renderWidgetNode: RenderWidgetNode,
  dispatchAction?: (action: ActionSpec, context: JsonObject) => void,
  rowKeySpec?: RowKeySpec
): ReactNode {
  switch (spec.kind) {
    case "field":
      return stringify(getPath(row, spec.field), spec.fallback);
    case "number":
      return formatNumber(getPath(row, spec.field), spec);
    case "status": {
      const status = stringify(getPath(row, spec.field), spec.fallback ?? "pending");
      return (
        <span
          className={`text-[11px] font-bold uppercase tracking-wider ${STATUS_TONES[status] ?? "text-[var(--color-ink)]"}`}
        >
          {status}
        </span>
      );
    }
    case "caption":
      return (
        <span className={`text-[11px] ${CAPTION_TONES[spec.tone ?? "muted"]}`}>
          {stringify(getPath(row, spec.field), spec.fallback)}
        </span>
      );
    case "template":
      return renderTemplate(spec.template, row);
    case "link": {
      const href = stringify(getPath(row, spec.hrefField), "#");
      const label = stringify(getPath(row, spec.labelField), spec.fallbackLabel ?? href);
      return (
        <a href={href} target={spec.target} rel={spec.target === "_blank" ? "noreferrer" : undefined}>
          {label}
        </a>
      );
    }
    case "actionButton":
      return (
        <button
          type="button"
          className="retro-btn-flat"
          disabled={spec.disabled}
          onClick={event => {
            event.stopPropagation();
            dispatchAction?.(spec.action, {
              row,
              rowKey: rowKey(row, rowKeySpec ?? "id"),
              componentType: "DataTableCell",
            });
          }}
        >
          {renderRenderable(spec.label, renderWidgetNode)}
        </button>
      );
    case "constant":
      return renderRenderable(spec.value, renderWidgetNode);
    default:
      return <span className="text-[var(--color-destructive-accent)] text-[11px]">Unsupported cell</span>;
  }
}

export function renderRenderable(
  value: RenderableValue | undefined,
  renderWidgetNode: RenderWidgetNode
): ReactNode {
  if (value === undefined || value === null || value === false) return null;
  if (isWidgetNode(value)) return renderWidgetNode(value);
  return String(value);
}

export function rowKey(row: JsonObject, spec: RowKeySpec): string {
  if (typeof spec === "string") return stringify(getPath(row, spec), "");
  if ("field" in spec) return stringify(getPath(row, spec.field), "");
  return renderTemplate(spec.template, row);
}

function getPath(row: JsonObject, path: string): unknown {
  const parts = path.split(".").filter(Boolean);
  let current: unknown = row;
  for (const part of parts) {
    if (!current || typeof current !== "object") return undefined;
    current = (current as Record<string, unknown>)[part];
  }
  return current;
}

function stringify(value: unknown, fallback = ""): string {
  if (value === undefined || value === null) return fallback;
  return String(value);
}

function formatNumber(value: unknown, spec: Extract<CellSpec, { kind: "number" }>): string {
  const n = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(n)) return spec.fallback ?? "";
  if (spec.format === "fixed") return n.toFixed(spec.digits ?? 2);
  return Math.round(n).toLocaleString();
}

function renderTemplate(template: string, row: JsonObject): string {
  return template.replace(
    /\$\{([^}]+)\}|\$([A-Za-z0-9_.-]+)/g,
    (_match, braced: string | undefined, bare: string | undefined) => {
      const path = braced ?? bare ?? "";
      return stringify(getPath(row, path), "");
    }
  );
}
