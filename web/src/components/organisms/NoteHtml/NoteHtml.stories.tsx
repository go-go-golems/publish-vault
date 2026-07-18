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
  },
};
