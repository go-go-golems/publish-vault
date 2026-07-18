import type { Meta, StoryObj } from "@storybook/react";
import { SidebarNav } from "./SidebarNav";

const meta: Meta<typeof SidebarNav> = {
  title: "Molecules/SidebarNav",
  component: SidebarNav,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof SidebarNav>;

export const Default: Story = {
  render: () => (
    <div style={{ height: 320, width: 240 }}>
      <SidebarNav
        navigation={{
          placement: "sidebar",
          brand: "go-go-parc",
          activeItemId: "reader",
          sections: [
            {
              id: "pages",
              label: { kind: "text", text: "Pages" },
              items: [
                { id: "reader", label: { kind: "text", text: "Reader" } },
                { id: "recent", label: { kind: "text", text: "Recently updated" } },
                {
                  id: "tags",
                  label: { kind: "text", text: "Tags" },
                  badge: { kind: "text", text: "12" },
                },
                { id: "off", label: { kind: "text", text: "Disabled" }, disabled: true },
              ],
            },
          ],
        }}
        onItemSelect={item => console.log("select:", item.id)}
      />
    </div>
  ),
};
