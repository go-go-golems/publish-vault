import type { Meta, StoryObj } from "@storybook/react";
import { Stack } from "./Stack";
import { Panel } from "../Panel/Panel";

const meta: Meta<typeof Stack> = {
  title: "Layout/Stack",
  component: Stack,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof Stack>;

export const Default: Story = {
  render: () => (
    <Stack gap="md">
      <Panel>First</Panel>
      <Panel>Second</Panel>
      <Panel>Third</Panel>
    </Stack>
  ),
};
