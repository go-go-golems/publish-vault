import type { Meta, StoryObj } from "@storybook/react";
import { SectionBlock } from "./SectionBlock";

const meta: Meta<typeof SectionBlock> = {
  title: "Layout/SectionBlock",
  component: SectionBlock,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof SectionBlock>;

export const Article: Story = {
  args: {
    title: "Linked mentions",
    caption: "Notes that reference the current note.",
    children: <p className="text-xs m-0">Section body content, laid out like an article.</p>,
  },
};

export const Window: Story = {
  args: {
    title: "Linked mentions",
    caption: "Retro window chrome for nested panel groupings.",
    variant: "window",
    children: <p className="text-xs m-0">Section body content.</p>,
  },
};
