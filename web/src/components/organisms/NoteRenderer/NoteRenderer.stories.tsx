import type { Meta, StoryObj } from "@storybook/react";
import { NoteRenderer } from "./NoteRenderer";

const meta: Meta<typeof NoteRenderer> = {
  title: "Organisms/NoteRenderer",
  component: NoteRenderer,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof NoteRenderer>;

const baseNote = {
  slug: "stoicism",
  title: "Stoicism",
  path: "Philosophy/Stoicism.md",
  frontmatter: { author: "Marcus Aurelius", status: "evergreen" },
  tags: ["philosophy", "stoicism"],
  excerpt: "Stoicism is a school of Hellenistic philosophy…",
  wikiLinks: [],
  backlinks: ["zettelkasten-method"],
  modTime: "2024-01-15",
};

export const Default: Story = {
  args: {
    note: {
      ...baseNote,
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
    },
    allSlugs: ["stoicism", "epistemology", "zeno-of-citium"],
    onNavigate: (slug) => console.log("navigate:", slug),
  },
};

export const WithMultipleLanguages: Story = {
  args: {
    note: {
      ...baseNote,
      title: "Code Examples",
      html: `
    <p>Here are some code examples in different languages:</p>
    <h3>Go</h3>
    <pre><code class="language-go">package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}</code></pre>
    <h3>TypeScript</h3>
    <pre><code class="language-typescript">interface Note {
  slug: string;
  title: string;
}

function render(note: Note): string {
  return \`&lt;h1&gt;\${note.title}&lt;/h1&gt;\`;
}</code></pre>
    <h3>Bash</h3>
    <pre><code class="language-bash">#!/bin/bash
echo "Building project..."
go build -o bin/app ./cmd/app
./bin/app --port 8080</code></pre>
    <h3>YAML</h3>
    <pre><code class="language-yaml">server:
  port: 8080
  host: 127.0.0.1
  vault: /home/user/notes</code></pre>
  `,
    },
    allSlugs: ["code-examples"],
    onNavigate: (slug) => console.log("navigate:", slug),
  },
};

export const WithMermaid: Story = {
  args: {
    note: {
      ...baseNote,
      title: "System Architecture",
      html: `
    <p>The system is built as a single Go binary serving both the API and the embedded React frontend.</p>
    <pre><code class="language-mermaid">graph TD
    A[Browser] --> B[Go Server]
    B --> C[Vault FS]
    B --> D[Search Index]
    C --> E[Markdown Files]
    D --> F[SQLite DB]
    B --> G[Embedded SPA]
    G --> A</code></pre>
    <p>The Go binary uses goldmark for Markdown parsing and go-sqlite3 for full-text search.</p>
    <h2>Data Flow</h2>
    <pre><code class="language-mermaid">sequenceDiagram
    participant U as User
    participant S as Server
    participant V as Vault
    U->>S: GET /api/notes/slug
    S->>V: Read note
    V-->>S: Markdown content
    S->>S: Parse &amp; render
    S-->>U: JSON response</code></pre>
  `,
    },
    allSlugs: ["system-architecture"],
    onNavigate: (slug) => console.log("navigate:", slug),
  },
};

export const WithMermaidAndCode: Story = {
  args: {
    note: {
      ...baseNote,
      title: "Full Technical Note",
      html: `
    <p>This note demonstrates both mermaid diagrams and syntax highlighting.</p>
    <pre><code class="language-mermaid">graph LR
    A[Input] --> B[Parser]
    B --> C[AST]
    C --> D[Renderer]
    D --> E[HTML]</code></pre>
    <p>The parser implementation:</p>
    <pre><code class="language-go">func Parse(src []byte) (*ParsedNote, error) {
    md := goldmark.New(
        goldmark.WithExtensions(
            meta.Meta,
            extension.GFM,
        ),
    )
    var buf bytes.Buffer
    if err := md.Convert(src, &amp;buf); err != nil {
        return nil, err
    }
    return &amp;ParsedNote{HTML: buf.String()}, nil
}</code></pre>
  `,
    },
    allSlugs: ["full-technical-note"],
    onNavigate: (slug) => console.log("navigate:", slug),
  },
};
