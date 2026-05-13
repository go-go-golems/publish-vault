import type { Meta, StoryObj } from "@storybook/react";
import { SearchBar } from "./SearchBar";

const meta: Meta<typeof SearchBar> = {
  title: "Molecules/SearchBar",
  component: SearchBar,
  tags: ["autodocs"],
};
export default meta;

type Story = StoryObj<typeof SearchBar>;

export const Default: Story = {
  args: { onSearch: (q) => console.log("search:", q) },
};

export const WithInitialValue: Story = {
  args: { onSearch: (q) => console.log(q), initialValue: "zettelkasten" },
};
