/**
 * Prop interfaces for publish-vault's v1 widget set, matching the IR emitted
 * by widget.dsl v3 (see internal/widgethost golden fixtures). Structurally a
 * subset of rag-evaluation-system's src/widgets/ir/props.ts.
 */
import type { ActionSpec } from "./actions";
import type { DataTableColumnSpec, RowKeySpec } from "./cells";
import type { JsonObject, WidgetNode } from "./core";

export interface StackWidgetProps {
  gap?: "none" | "xs" | "sm" | "md" | "lg";
  className?: string;
}

export interface InlineWidgetProps {
  gap?: "none" | "xs" | "sm" | "md" | "lg";
  className?: string;
}

export interface SectionBlockWidgetProps {
  label: string;
  caption?: string;
  level?: number;
  density?: "flush" | "comfortable";
  rule?: boolean;
  anchor?: string;
  className?: string;
}

export interface DataTableWidgetProps {
  columns: DataTableColumnSpec[];
  rows: JsonObject[];
  getRowKey: RowKeySpec;
  onRowSelect?: ActionSpec;
  selectedKey?: string | number | null;
  emptyMessage?: string;
  className?: string;
}

export interface KeyValueStripItemSpec {
  key?: WidgetNode;
  label: WidgetNode;
  value: WidgetNode;
}

export interface KeyValueStripWidgetProps {
  items: KeyValueStripItemSpec[];
  className?: string;
}

export interface PanelWidgetProps {
  title?: string;
  tone?: "default" | "success" | "warning" | "danger" | "info";
  className?: string;
}

export interface TextWidgetProps {
  tone?: "default" | "muted" | "danger" | "success";
  className?: string;
}

export interface CaptionWidgetProps {
  tone?: "default" | "muted" | "danger" | "success";
  className?: string;
}

export interface TagWidgetProps {
  label?: string;
  className?: string;
}

export interface DividerWidgetProps {
  label?: string;
  className?: string;
}
