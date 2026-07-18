import type { Meta, StoryObj } from "@storybook/react";
import { Inline } from "./Inline";

const meta: Meta<typeof Inline> = {
  title: "Layout/Inline",
  component: Inline,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof Inline>;

export const Default: Story = {
  render: () => (
    <Inline gap="sm">
      <button className="retro-btn">One</button>
      <button className="retro-btn">Two</button>
      <span className="retro-badge">Badge</span>
    </Inline>
  ),
};
