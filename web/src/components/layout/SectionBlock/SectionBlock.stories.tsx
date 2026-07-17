import type { Meta, StoryObj } from "@storybook/react";
import { SectionBlock } from "./SectionBlock";

const meta: Meta<typeof SectionBlock> = {
  title: "Layout/SectionBlock",
  component: SectionBlock,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof SectionBlock>;

export const Default: Story = {
  args: {
    title: "Linked mentions",
    caption: "Notes that reference the current note.",
    children: <p className="text-xs m-0">Section body content.</p>,
  },
};
