import type { Meta, StoryObj } from "@storybook/react";
import { AdvancedSearchPanel } from "./AdvancedSearchPanel";
import type { SearchRequest } from "../../../types";

const current: SearchRequest = {
  query: "memory",
  tags: ["go", "performance"],
  tagMode: "all",
  pathPrefixes: ["research/kb/"],
  dateField: "display",
  dateFrom: { year: 2024, month: 1, day: 1 },
  dateTo: { year: 2024, month: 12, day: 31 },
  sort: "newest",
  limit: 30,
  offset: 0,
};

const meta: Meta<typeof AdvancedSearchPanel> = {
  title: "Molecules/AdvancedSearchPanel",
  component: AdvancedSearchPanel,
  tags: ["autodocs"],
  args: { open: true, current, onApply: () => {}, onOpenChange: () => {} },
};
export default meta;

type Story = StoryObj<typeof AdvancedSearchPanel>;

export const Open: Story = {};
