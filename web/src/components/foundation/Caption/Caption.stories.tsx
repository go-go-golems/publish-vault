import type { Meta, StoryObj } from "@storybook/react";
import { Caption } from "./Caption";

const meta: Meta<typeof Caption> = {
  title: "Foundation/Caption",
  component: Caption,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof Caption>;

export const Default: Story = {
  args: { children: "Linked mentions" },
};

export const AsHeading: Story = {
  args: { as: "h3", children: "Section heading" },
};
