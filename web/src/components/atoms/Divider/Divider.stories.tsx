import type { Meta, StoryObj } from "@storybook/react";
import { Divider } from "./Divider";

const meta: Meta<typeof Divider> = {
  title: "Atoms/Divider",
  component: Divider,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof Divider>;

export const Plain: Story = {};
export const WithLabel: Story = { args: { label: "Backlinks" } };
export const Vertical: Story = {
  render: () => (
    <div className="flex h-16 items-center gap-4">
      <span>Left</span>
      <Divider orientation="vertical" />
      <span>Right</span>
    </div>
  ),
};
