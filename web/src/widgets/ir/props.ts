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

// ── Note-domain widgets (PV-VAULT-WIDGETS-016; emitted by vault.widgets) ──

export interface NoteHtmlWidgetProps {
  html: string;
  slug?: string;
  mermaid?: boolean;
  highlight?: boolean;
  embeds?: boolean;
  anchors?: boolean;
}

export interface FrontmatterPanelWidgetProps {
  frontmatter: JsonObject;
  tags?: string[];
  modTime?: string;
  onTagClick?: ActionSpec;
  className?: string;
}

export interface BreadcrumbBarWidgetProps {
  segments: { label: string; slug?: string }[];
  onSelect?: ActionSpec;
  className?: string;
}

export interface BacklinksPanelWidgetProps {
  entries: { slug: string; title: string; excerpt?: string }[];
  onSelect?: ActionSpec;
  className?: string;
}

export interface TagCloudWidgetProps {
  tags: (string | { tag: string; count?: number })[];
  onSelect?: ActionSpec;
  className?: string;
}

export interface NoteCardWidgetProps {
  slug: string;
  title: string;
  excerpt?: string;
  tags?: string[];
  onSelect?: ActionSpec;
  className?: string;
}
