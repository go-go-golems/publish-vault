import type { Meta, StoryObj } from "@storybook/react";
import { Checkbox } from "./Checkbox";

const meta: Meta<typeof Checkbox> = {
  title: "Atoms/Checkbox",
  component: Checkbox,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof Checkbox>;

export const Unchecked: Story = { args: { label: "Show backlinks" } };
export const Checked: Story = { args: { label: "Show backlinks", checked: true } };
export const Disabled: Story = { args: { label: "Read-only", disabled: true, checked: true } };
