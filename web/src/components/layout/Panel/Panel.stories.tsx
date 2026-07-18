import type { Meta, StoryObj } from "@storybook/react";
import { Panel } from "./Panel";

const meta: Meta<typeof Panel> = {
  title: "Layout/Panel",
  component: Panel,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof Panel>;

export const Variants: Story = {
  render: () => (
    <div className="flex flex-col gap-4 p-4">
      <Panel variant="window">Window — raised with drop shadow</Panel>
      <Panel variant="inset">Inset — sunken panel</Panel>
      <Panel variant="plain">Plain — 1px border</Panel>
    </div>
  ),
};
