import type { Preview } from "@storybook/react";
import "../client/src/index.css";

const preview: Preview = {
  parameters: {
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i,
      },
    },
    backgrounds: {
      default: "retro-light",
      values: [
        { name: "retro-light", value: "#f0ede8" },
        { name: "retro-dark", value: "#1a1a1a" },
      ],
    },
  },
};

export default preview;
