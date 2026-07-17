import type { StorybookConfig } from "@storybook/react-vite";

const config: StorybookConfig = {
  stories: ["../src/**/*.stories.@(js|jsx|ts|tsx)"],
  // Storybook 10: essentials (controls, docs, actions, backgrounds, toolbars)
  // are built into core; the v8 addon packages are incompatible and removed.
  addons: [],
  framework: {
    name: "@storybook/react-vite",
    options: {},
  },
};

export default config;
