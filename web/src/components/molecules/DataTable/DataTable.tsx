/**
 * MOLECULE: DataTable
 * Design: Retro System 1 — 1px-ruled table with inverted header row.
 * Presentational: columns render precomputed ReactNodes; row selection and
 * cell behavior are injected by the caller (see DataTable.widget.tsx for the
 * widget IR adapter).
 */
import React from "react";
import { clsx } from "clsx";

export interface DataTableColumn<Row> {
  id: string;
  header: React.ReactNode;
  align?: "start" | "end" | "center";
  cell: (row: Row) => React.ReactNode;
}

export interface DataTableProps<Row> {
  columns: DataTableColumn<Row>[];
  rows: Row[];
  getRowKey: (row: Row) => string;
  onRowSelect?: (row: Row) => void;
  selectedKey?: string | null;
  emptyMessage?: React.ReactNode;
  className?: string;
}

const ALIGN_CLASSES = {
  start: "text-left",
  center: "text-center",
  end: "text-right",
} as const;

export function DataTable<Row>({
  columns,
  rows,
  getRowKey,
  onRowSelect,
  selectedKey,
  emptyMessage = "No entries",
  className,
}: DataTableProps<Row>) {
  return (
    <table
      className={clsx(
        "w-full border-collapse text-xs border border-[var(--color-ink)]",
        className
      )}
    >
      <thead>
        <tr>
          {columns.map(column => (
            <th
              key={column.id}
              className={clsx(
                "bg-[var(--color-ink)] text-[var(--color-paper)] font-bold px-2 py-1 border border-[var(--color-ink)]",
                ALIGN_CLASSES[column.align ?? "start"]
              )}
            >
              {column.header}
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {rows.length === 0 && (
          <tr>
            <td
              className="px-2 py-2 text-[var(--color-muted-foreground)] italic border border-[var(--color-ink)]"
              colSpan={columns.length}
            >
              {emptyMessage}
            </td>
          </tr>
        )}
        {rows.map(row => {
          const key = getRowKey(row);
          const selected = selectedKey != null && key === selectedKey;
          return (
            <tr
              key={key}
              aria-selected={selected || undefined}
              className={clsx(
                "even:bg-[var(--color-panel)]",
                selected && "bg-[var(--color-ink)] text-[var(--color-paper)]",
                onRowSelect &&
                  "cursor-pointer hover:bg-[var(--color-ink)] hover:text-[var(--color-paper)]"
              )}
              onClick={onRowSelect ? () => onRowSelect(row) : undefined}
            >
              {columns.map(column => (
                <td
                  key={column.id}
                  className={clsx(
                    "px-2 py-1 border border-[var(--color-ink)]",
                    ALIGN_CLASSES[column.align ?? "start"]
                  )}
                >
                  {column.cell(row)}
                </td>
              ))}
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}
