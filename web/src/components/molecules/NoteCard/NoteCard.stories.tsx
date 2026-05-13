import type { Meta, StoryObj } from "@storybook/react";
import { NoteCard } from "./NoteCard";

const meta: Meta<typeof NoteCard> = {
  title: "Molecules/NoteCard",
  component: NoteCard,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof NoteCard>;

export const Default: Story = {
  args: {
    slug: "zettelkasten-method",
    title: "The Zettelkasten Method",
    excerpt: "A note-taking system developed by Niklas Luhmann that uses atomic notes linked together to form a web of knowledge.",
    tags: ["zettelkasten", "productivity"],
    modTime: "2024-01-15",
  },
};

export const Active: Story = {
  args: { ...Default.args, active: true },
};

export const NoExcerpt: Story = {
  args: { slug: "empty", title: "Empty Note", tags: ["draft"] },
};
