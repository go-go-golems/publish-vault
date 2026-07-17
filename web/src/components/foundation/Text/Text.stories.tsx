import type { Meta, StoryObj } from "@storybook/react";
import { Text } from "./Text";

const meta: Meta<typeof Text> = {
  title: "Foundation/Text",
  component: Text,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof Text>;

export const Default: Story = {
  args: { children: "The quick brown fox jumps over the lazy dog." },
};

export const Sizes: Story = {
  render: () => (
    <div className="flex flex-col gap-2">
      <Text size="md">Medium — body copy</Text>
      <Text size="sm">Small — secondary copy</Text>
      <Text size="xs">Extra small — metadata</Text>
    </div>
  ),
};

export const Tones: Story = {
  render: () => (
    <div className="flex flex-col gap-2">
      <Text tone="default">Default ink</Text>
      <Text tone="muted">Muted secondary</Text>
      <Text tone="danger">Danger / destructive</Text>
      <Text tone="success">Success / tag green</Text>
      <Text bold>Bold ink</Text>
    </div>
  ),
};
