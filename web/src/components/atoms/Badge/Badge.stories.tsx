import type { Meta, StoryObj } from "@storybook/react";
import { Badge } from "./Badge";

const meta: Meta<typeof Badge> = {
  title: "Atoms/Badge",
  component: Badge,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof Badge>;

export const Default: Story = { args: { children: "12" } };
export const Accent: Story = { args: { children: "NEW", variant: "accent" } };
export const Muted: Story = { args: { children: "draft", variant: "muted" } };
