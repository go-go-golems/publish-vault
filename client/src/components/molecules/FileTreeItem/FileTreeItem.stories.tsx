import type { Meta, StoryObj } from "@storybook/react";
import { FileTreeItem } from "./FileTreeItem";

const meta: Meta<typeof FileTreeItem> = {
  title: "Molecules/FileTreeItem",
  component: FileTreeItem,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof FileTreeItem>;

const sampleTree = {
  name: "Notes",
  path: "Notes",
  isFolder: true,
  children: [
    { name: "Zettelkasten Method", slug: "zettelkasten-method", path: "Notes/Zettelkasten Method.md", isFolder: false },
    {
      name: "Philosophy",
      path: "Notes/Philosophy",
      isFolder: true,
      children: [
        { name: "Stoicism", slug: "stoicism", path: "Notes/Philosophy/Stoicism.md", isFolder: false },
        { name: "Epistemology", slug: "epistemology", path: "Notes/Philosophy/Epistemology.md", isFolder: false },
      ],
    },
  ],
};

export const Default: Story = {
  args: { node: sampleTree, activeSlug: "stoicism" },
};
