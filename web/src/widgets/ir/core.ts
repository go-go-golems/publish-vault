/**
 * Widget IR core types — the serialized page representation produced by the
 * Go widget host (widget.dsl v3) and consumed by WidgetRenderer.
 *
 * Ported from rag-evaluation-system packages/rag-evaluation-site
 * src/widgets/ir/core.ts (v0.1.7), trimmed to publish-vault's v1 widget set.
 * Keep field names/shapes identical to the source — the wire format is owned
 * by pkg/widgetdsl/spec in rag-evaluation-system.
 */
import type { ActionSpec } from "./actions";

export type JsonPrimitive = string | number | boolean | null;
export type JsonValue = JsonPrimitive | JsonValue[] | { [key: string]: JsonValue };
export type JsonObject = { [key: string]: JsonValue };

export type WidgetNode = TextNode | ElementNode | ComponentNode;

export interface TextNode {
  kind: "text";
  text: string;
}

export interface ElementNode {
  kind: "element";
  tag: string;
  attrs?: JsonObject;
  children?: WidgetNode[];
}

/** Component types registered in publish-vault's v1 default registry. */
export type PvWidgetType =
  | "Stack"
  | "Inline"
  | "Panel"
  | "SectionBlock"
  | "DataTable"
  | "KeyValueStrip"
  | "Text"
  | "Caption"
  | "Divider"
  | "Tag";

export interface ComponentNode {
  kind: "component";
  type: PvWidgetType | string;
  props?: Record<string, unknown>;
  children?: WidgetNode[];
}

export type RenderableValue = WidgetNode | string | number | boolean | null;

export interface BaseWidgetProps {
  className?: string;
  style?: JsonObject;
  id?: string;
  action?: ActionSpec;
  [key: string]: unknown;
}

export function isWidgetNode(value: unknown): value is WidgetNode {
  if (!value || typeof value !== "object") return false;
  const kind = (value as { kind?: unknown }).kind;
  return kind === "text" || kind === "element" || kind === "component";
}
