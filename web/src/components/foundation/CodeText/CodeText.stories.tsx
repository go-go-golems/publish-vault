import type { Meta, StoryObj } from "@storybook/react";
import { CodeText } from "./CodeText";

const meta: Meta<typeof CodeText> = {
  title: "Foundation/CodeText",
  component: CodeText,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof CodeText>;

export const Default: Story = {
  args: { children: "vault.notes()" },
};
