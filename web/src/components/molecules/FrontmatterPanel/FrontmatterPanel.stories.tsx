import type { Meta, StoryObj } from "@storybook/react";
import { FrontmatterPanel } from "./FrontmatterPanel";

const meta: Meta<typeof FrontmatterPanel> = {
  title: "Molecules/FrontmatterPanel",
  component: FrontmatterPanel,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof FrontmatterPanel>;

export const Default: Story = {
  args: {
    frontmatter: {
      title: "Stoicism",
      author: "Marcus Aurelius",
      source: "https://example.com",
      status: "evergreen",
    },
    tags: ["philosophy", "stoicism"],
    modTime: "2024-01-15",
  },
};

export const Empty: Story = {
  args: { frontmatter: {}, tags: [], modTime: "2024-01-01" },
};
