import type { Meta, StoryObj } from "@storybook/react";
import { BacklinksPanel } from "./BacklinksPanel";

const meta: Meta<typeof BacklinksPanel> = {
  title: "Organisms/BacklinksPanel",
  component: BacklinksPanel,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof BacklinksPanel>;

export const Default: Story = {
  args: {
    backlinks: [
      { slug: "zettelkasten-method", title: "The Zettelkasten Method", excerpt: "…atomic notes linked together…" },
      { slug: "reading-notes", title: "Reading Notes" },
      { slug: "philosophy-index", title: "Philosophy Index", excerpt: "…Stoicism is one of the major schools…" },
    ],
    onNavigate: (slug) => console.log("navigate:", slug),
  },
};

export const Empty: Story = {
  args: { backlinks: [], onNavigate: () => {} },
};
