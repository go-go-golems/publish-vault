import type { Meta, StoryObj } from "@storybook/react";
import { NoteRenderer } from "./NoteRenderer";

const meta: Meta<typeof NoteRenderer> = {
  title: "Organisms/NoteRenderer",
  component: NoteRenderer,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof NoteRenderer>;

const sampleNote = {
  slug: "stoicism",
  title: "Stoicism",
  path: "Philosophy/Stoicism.md",
  frontmatter: { author: "Marcus Aurelius", status: "evergreen" },
  tags: ["philosophy", "stoicism"],
  excerpt: "Stoicism is a school of Hellenistic philosophy…",
  html: `
    <p>Stoicism is a school of Hellenistic philosophy founded by <a href="/note/zeno-of-citium" class="wiki-link" data-target="zeno-of-citium">Zeno of Citium</a> in Athens in the early 3rd century BC.</p>
    <h2>Core Principles</h2>
    <ul>
      <li>Live according to nature</li>
      <li>Virtue is the only good</li>
      <li>Distinguish what is in your control</li>
    </ul>
    <blockquote><p>You have power over your mind, not outside events. Realize this, and you will find strength.</p></blockquote>
    <p>See also: <a href="/note/epistemology" class="wiki-link" data-target="epistemology">Epistemology</a> and <a href="/note/broken-link" class="wiki-link broken" data-target="broken-link">Broken Link</a>.</p>
    <pre><code class="language-python">def virtue() -&gt; str:
    return "the only good"
</code></pre>
  `,
  wikiLinks: [],
  backlinks: ["zettelkasten-method"],
  modTime: "2024-01-15",
};

export const Default: Story = {
  args: {
    note: sampleNote,
    allSlugs: ["stoicism", "epistemology", "zeno-of-citium"],
    onNavigate: (slug) => console.log("navigate:", slug),
  },
};
