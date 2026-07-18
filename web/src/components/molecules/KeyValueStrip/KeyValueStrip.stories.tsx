import type { Meta, StoryObj } from "@storybook/react";
import { KeyValueStrip } from "./KeyValueStrip";

const meta: Meta<typeof KeyValueStrip> = {
  title: "Molecules/KeyValueStrip",
  component: KeyValueStrip,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof KeyValueStrip>;

export const Default: Story = {
  args: {
    items: [
      { key: "notes", label: "Notes", value: "482" },
      { key: "broken", label: "Broken links", value: "17" },
      { key: "orphans", label: "Orphans", value: "31" },
    ],
  },
};
