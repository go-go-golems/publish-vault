import type { Meta, StoryObj } from "@storybook/react";
import { BacklinkItem } from "./BacklinkItem";

const meta: Meta<typeof BacklinkItem> = {
  title: "Molecules/BacklinkItem",
  component: BacklinkItem,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof BacklinkItem>;

export const Default: Story = {
  args: {
    slug: "zettelkasten-method",
    title: "The Zettelkasten Method",
    excerpt: "…uses atomic notes linked together to form a web of knowledge…",
  },
};

export const List: Story = {
  render: () => (
    <div className="border border-[var(--color-ink)]">
      {[
        { slug: "a", title: "Atomic Notes", excerpt: "Each note should contain one idea." },
        { slug: "b", title: "Linking Notes", excerpt: "Links create the value in a Zettelkasten." },
        { slug: "c", title: "Niklas Luhmann" },
      ].map((item) => (
        <BacklinkItem key={item.slug} {...item} />
      ))}
    </div>
  ),
};
