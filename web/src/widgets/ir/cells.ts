/**
 * Widget IR cell specs — defunctionalized DataTable cell renderers.
 * Ported from rag-evaluation-system packages/rag-evaluation-site
 * src/widgets/ir/cells.ts (v0.1.7); button/status variant types are local
 * because publish-vault renders cells with its own retro components.
 */
import type { ActionSpec } from "./actions";
import type { RenderableValue } from "./core";

export type CaptionTone = "default" | "muted" | "danger" | "success";

export interface DataTableColumnSpec {
  id: string;
  header: RenderableValue;
  cell: CellSpec;
  align?: "start" | "end" | "center";
  maxWidth?: number | string;
  sortable?: boolean;
  sortDirection?: "ascending" | "descending";
  onSort?: ActionSpec;
}

export type RowKeySpec = string | { field: string } | { template: string };

export type CellSpec =
  | FieldCellSpec
  | NumberCellSpec
  | StatusCellSpec
  | CaptionCellSpec
  | TemplateCellSpec
  | LinkCellSpec
  | ActionButtonCellSpec
  | ConstantCellSpec;

export interface FieldCellSpec {
  kind: "field";
  field: string;
  fallback?: string;
}

export interface NumberCellSpec {
  kind: "number";
  field: string;
  format?: "integer" | "fixed";
  digits?: number;
  fallback?: string;
}

export interface StatusCellSpec {
  kind: "status";
  field: string;
  icon?: boolean;
  fallback?: string;
}

export interface CaptionCellSpec {
  kind: "caption";
  field: string;
  tone?: CaptionTone;
  fallback?: string;
}

export interface TemplateCellSpec {
  kind: "template";
  template: string;
}

export interface LinkCellSpec {
  kind: "link";
  hrefField: string;
  labelField: string;
  target?: "_blank" | "_self" | "_parent" | "_top";
  fallbackLabel?: string;
}

export interface ActionButtonCellSpec {
  kind: "actionButton";
  label: RenderableValue;
  action: ActionSpec;
  disabled?: boolean;
}

export interface ConstantCellSpec {
  kind: "constant";
  value: RenderableValue;
}
