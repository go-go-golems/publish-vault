import type { Meta, StoryObj } from "@storybook/react";
import { Tag } from "./Tag";

const meta: Meta<typeof Tag> = {
  title: "Atoms/Tag",
  component: Tag,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof Tag>;

export const Default: Story = { args: { label: "philosophy" } };
export const Active: Story = { args: { label: "active", active: true } };
export const Clickable: Story = {
  args: { label: "clickable", onClick: () => alert("tag clicked") },
};
export const Group: Story = {
  render: () => (
    <div className="flex gap-1 flex-wrap">
      {["zettelkasten", "philosophy", "reading", "projects", "daily"].map((t) => (
        <Tag key={t} label={t} />
      ))}
    </div>
  ),
};
