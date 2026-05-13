import type { Meta, StoryObj } from "@storybook/react";
import { Input } from "./Input";

const meta: Meta<typeof Input> = {
  title: "Atoms/Input",
  component: Input,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof Input>;

export const Default: Story = { args: { placeholder: "Type here…" } };
export const WithLabel: Story = { args: { label: "Note title", placeholder: "My Note" } };
export const WithError: Story = {
  args: { label: "Slug", placeholder: "my-note", error: "Slug already exists" },
};
