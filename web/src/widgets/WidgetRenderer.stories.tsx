import type { Meta, StoryObj } from "@storybook/react";
import { WidgetRenderer } from "./WidgetRenderer";
import { defaultWidgetRegistry } from "./defaultRegistry";
import type { WidgetNode } from "./ir";
import recentPageFixture from "./__fixtures__/recent-page.json";

const meta: Meta<typeof WidgetRenderer> = {
  title: "Widgets/WidgetRenderer",
  component: WidgetRenderer,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof WidgetRenderer>;

/**
 * The exact IR served by `GET /api/widget/pages/recent` against
 * vault-example (captured fixture) — the full server-driven page as the SPA
 * renders it.
 */
export const RecentNotesPage: Story = {
  args: {
    node: (recentPageFixture as { root: WidgetNode }).root,
    registry: defaultWidgetRegistry,
    onAction: (action, context) => console.log("action:", action, context),
  },
};

export const UnknownWidgetFallback: Story = {
  args: {
    node: {
      kind: "component",
      type: "NotARealWidget",
      props: {},
    },
    registry: defaultWidgetRegistry,
  },
};

export const MetricsAndCallout: Story = {
  args: {
    registry: defaultWidgetRegistry,
    node: {
      kind: "component",
      type: "Stack",
      props: { gap: "lg" },
      children: [
        {
          kind: "component",
          type: "SectionBlock",
          props: { label: "Metrics", caption: "KeyValueStrip + Panel tones" },
          children: [
            {
              kind: "component",
              type: "KeyValueStrip",
              props: {
                items: [
                  {
                    label: { kind: "text", text: "Notes" },
                    value: { kind: "text", text: "128" },
                  },
                ],
              },
            },
            {
              kind: "component",
              type: "Panel",
              props: { title: "Healthy", tone: "success" },
              children: [{ kind: "text", text: "The vault is ready." }],
            },
          ],
        },
      ],
    } as WidgetNode,
  },
};
