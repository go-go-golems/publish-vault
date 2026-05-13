import type { Meta, StoryObj } from "@storybook/react";
import { Sidebar } from "./Sidebar";

const meta: Meta<typeof Sidebar> = {
  title: "Organisms/Sidebar",
  component: Sidebar,
  tags: ["autodocs"],
  decorators: [
    (Story) => (
      <div style={{ height: "500px", display: "flex" }}>
        <Story />
      </div>
    ),
  ],
};
export default meta;

type Story = StoryObj<typeof Sidebar>;

const sampleTree = {
  name: "root",
  path: "",
  isFolder: true,
  children: [
    {
      name: "Philosophy",
      path: "Philosophy",
      isFolder: true,
      children: [
        { name: "Stoicism", slug: "stoicism", path: "Philosophy/Stoicism.md", isFolder: false },
        { name: "Epistemology", slug: "epistemology", path: "Philosophy/Epistemology.md", isFolder: false },
      ],
    },
    { name: "Index", slug: "index", path: "Index.md", isFolder: false },
    { name: "Daily Notes", slug: "daily-notes", path: "Daily Notes.md", isFolder: false },
  ],
};

export const Default: Story = {
  args: {
    tree: sampleTree,
    vaultName: "My Vault",
    activeSlug: "stoicism",
    onSelectNote: (slug) => console.log("select:", slug),
    onSearch: (q) => console.log("search:", q),
  },
};

export const Loading: Story = {
  args: { ...Default.args, tree: null, isLoading: true },
};
