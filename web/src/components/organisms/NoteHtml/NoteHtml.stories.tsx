import type { Meta, StoryObj } from "@storybook/react";
import { Provider } from "react-redux";
import { NoteHtml } from "./NoteHtml";
import { makeStore } from "../../../store/store";

const meta: Meta<typeof NoteHtml> = {
  title: "Organisms/NoteHtml",
  component: NoteHtml,
  tags: ["autodocs"],
  decorators: [
    Story => (
      <Provider store={makeStore()}>
        <div className="note-prose">
          <Story />
        </div>
      </Provider>
    ),
  ],
};
export default meta;

type Story = StoryObj<typeof NoteHtml>;

const html = `
  <h2 id="intro">Intro</h2>
  <p>Rendered note body with a <a href="/note/target" class="wiki-link" data-target="target">wiki link</a>.</p>
  <pre><code class="language-go">func main() { fmt.Println("hi") }</code></pre>
`;

export const Default: Story = {
  args: { html, slug: "story-note" },
};

export const EnhancementsDisabled: Story = {
  args: {
    html,
    slug: "story-note",
    mermaid: false,
    highlight: false,
    embeds: false,
    anchors: false,
    math: false,
  },
};

// The placeholders below are exactly what internal/parser emits: the TeX lives
// in the element's text content, escaped only for &, < and >. enhanceMath swaps
// each one for MathJax SVG after mount.
const mathHtml = `
  <p>The identity <span class="math math-inline">e^{i\\pi} + 1 = 0</span> ties together
  five constants, and <span class="math math-inline">a_1, a_2, \\ldots, a_n</span> keeps
  its underscores.</p>
  <div class="math math-display">
\\begin{align}
\\mathbb{E}[X]      &amp;= \\mu \\\\
\\mathrm{Var}(X)    &amp;= \\sigma^2
\\end{align}
  </div>
  <p>Prices are prose: the book costs $30 and $25 used.</p>
`;

export const Math: Story = {
  args: { html: mathHtml, slug: "math-note" },
};

/** Math left untypeset — this is what a reader without JavaScript sees. */
export const MathNotTypeset: Story = {
  args: { html: mathHtml, slug: "math-note", math: false },
};
