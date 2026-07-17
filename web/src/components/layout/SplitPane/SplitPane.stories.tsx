import type { Meta, StoryObj } from "@storybook/react";
import { SplitPane } from "./SplitPane";

const meta: Meta<typeof SplitPane> = {
  title: "Layout/SplitPane",
  component: SplitPane,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof SplitPane>;

export const Default: Story = {
  render: () => (
    <div style={{ height: 240 }}>
      <SplitPane
        main={<div className="p-4">Main content pane</div>}
        side={
          <div className="p-4 border-l border-[var(--color-ink)] h-full">
            Side panel
          </div>
        }
      />
    </div>
  ),
};
