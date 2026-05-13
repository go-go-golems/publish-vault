import type { Meta, StoryObj } from "@storybook/react";
import { ScrollArea } from "./ScrollArea";

const meta: Meta<typeof ScrollArea> = {
  title: "Atoms/ScrollArea",
  component: ScrollArea,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof ScrollArea>;

export const Default: Story = {
  render: () => (
    <ScrollArea maxHeight="120px" className="border border-[var(--color-ink)] p-2">
      {Array.from({ length: 20 }, (_, i) => (
        <div key={i} className="text-xs py-0.5">Line {i + 1} — some content here</div>
      ))}
    </ScrollArea>
  ),
};
