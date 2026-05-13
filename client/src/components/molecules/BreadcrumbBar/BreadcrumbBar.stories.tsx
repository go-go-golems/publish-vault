import type { Meta, StoryObj } from "@storybook/react";
import { BreadcrumbBar } from "./BreadcrumbBar";

const meta: Meta<typeof BreadcrumbBar> = {
  title: "Molecules/BreadcrumbBar",
  component: BreadcrumbBar,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof BreadcrumbBar>;

export const Default: Story = {
  args: {
    segments: [
      { label: "Vault", slug: "" },
      { label: "Philosophy", slug: "philosophy" },
      { label: "Stoicism" },
    ],
  },
};

export const SingleNote: Story = {
  args: { segments: [{ label: "Vault", slug: "" }, { label: "Index" }] },
};
