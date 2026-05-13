import type { Meta, StoryObj } from "@storybook/react";
import { Icon, ICON_MAP } from "./Icon";

const meta: Meta<typeof Icon> = {
  title: "Atoms/Icon",
  component: Icon,
  tags: ["autodocs"],
  argTypes: {
    name: { control: "select", options: Object.keys(ICON_MAP) },
    size: { control: { type: "range", min: 10, max: 32, step: 2 } },
  },
};
export default meta;

type Story = StoryObj<typeof Icon>;

export const Default: Story = { args: { name: "file", size: 16 } };

export const AllIcons: Story = {
  render: () => (
    <div className="flex flex-wrap gap-4">
      {(Object.keys(ICON_MAP) as Array<keyof typeof ICON_MAP>).map((name) => (
        <div key={name} className="flex flex-col items-center gap-1">
          <Icon name={name} size={16} />
          <span className="text-[9px] text-[var(--color-muted-foreground)]">{name}</span>
        </div>
      ))}
    </div>
  ),
};
