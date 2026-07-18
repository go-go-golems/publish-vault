import type { Meta, StoryObj } from "@storybook/react";
import { DataTable } from "./DataTable";

interface Row {
  slug: string;
  title: string;
  status: string;
}

const rows: Row[] = [
  { slug: "widget-ir", title: "Widget IR notes", status: "ready" },
  { slug: "goja-dsl", title: "Goja DSL research", status: "draft" },
  { slug: "retro-ui", title: "Retro UI checklist", status: "review" },
];

const meta: Meta<typeof DataTable<Row>> = {
  title: "Molecules/DataTable",
  component: DataTable<Row>,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof DataTable<Row>>;

const columns = [
  { id: "slug", header: "Slug", cell: (row: Row) => row.slug },
  { id: "title", header: "Title", cell: (row: Row) => row.title },
  { id: "status", header: "Status", cell: (row: Row) => row.status },
];

export const Default: Story = {
  args: {
    columns,
    rows,
    getRowKey: (row: Row) => row.slug,
  },
};

export const SelectableRows: Story = {
  args: {
    columns,
    rows,
    getRowKey: (row: Row) => row.slug,
    selectedKey: "goja-dsl",
    onRowSelect: (row: Row) => console.log("select:", row.slug),
  },
};

export const Empty: Story = {
  args: {
    columns,
    rows: [],
    getRowKey: (row: Row) => row.slug,
    emptyMessage: "No notes yet",
  },
};
