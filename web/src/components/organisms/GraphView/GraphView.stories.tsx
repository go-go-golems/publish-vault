import type { Meta, StoryObj } from "@storybook/react";
import { GraphView } from "./GraphView";

const meta: Meta<typeof GraphView> = {
  title: "Organisms/GraphView",
  component: GraphView,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof GraphView>;

const sampleData = {
  nodes: [
    { id: "index", title: "Index", tags: [] },
    { id: "stoicism", title: "Stoicism", tags: ["philosophy"] },
    { id: "epistemology", title: "Epistemology", tags: ["philosophy"] },
    { id: "zettelkasten", title: "Zettelkasten", tags: ["productivity"] },
    { id: "atomic-notes", title: "Atomic Notes", tags: ["productivity"] },
    { id: "zeno", title: "Zeno of Citium", tags: ["philosophy"] },
  ],
  edges: [
    { source: "index", target: "stoicism" },
    { source: "index", target: "zettelkasten" },
    { source: "stoicism", target: "epistemology" },
    { source: "stoicism", target: "zeno" },
    { source: "zettelkasten", target: "atomic-notes" },
    { source: "atomic-notes", target: "index" },
  ],
};

export const Default: Story = {
  args: {
    data: sampleData,
    activeNodeId: "stoicism",
    onNodeClick: (id) => console.log("node click:", id),
    width: 500,
    height: 350,
  },
};
