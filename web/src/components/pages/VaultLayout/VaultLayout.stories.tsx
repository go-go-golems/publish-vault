import type { Meta, StoryObj } from "@storybook/react";
import { VaultLayout } from "./VaultLayout";
import { Provider } from "react-redux";
import { configureStore } from "@reduxjs/toolkit";
import uiReducer from "../../../store/uiSlice";
import { vaultApi } from "../../../store/vaultApi";

function mockStore(overrides: Record<string, unknown> = {}) {
  return configureStore({
    reducer: {
      ui: uiReducer,
      [vaultApi.reducerPath]: () => ({}),
    },
    preloadedState: {
      ui: {
        sidebarOpen: true,
        rightPanelOpen: true,
        searchQuery: "",
        activeNoteSlug: null,
        ...overrides,
      },
      [vaultApi.reducerPath]: {},
    },
  });
}

const meta: Meta<typeof VaultLayout> = {
  title: "Pages/VaultLayout",
  component: VaultLayout,
  tags: ["autodocs"],
  decorators: [
    (Story, context) => {
      const store = mockStore(context.parameters.uiState ?? {});
      return (
        <Provider store={store}>
          <div style={{ height: "600px", width: "100%" }}>
            <Story />
          </div>
        </Provider>
      );
    },
  ],
};
export default meta;

type Story = StoryObj<typeof VaultLayout>;

export const Default: Story = {
  args: {
    children: (
      <div className="p-6 note-prose">
        <h1>Welcome to the Vault</h1>
        <p>Select a note from the sidebar to begin reading.</p>
      </div>
    ),
    vaultName: "My Vault",
  },
};

export const SidebarCollapsed: Story = {
  args: { ...Default.args },
  parameters: {
    uiState: { sidebarOpen: false },
  },
};

export const NarrowSidebar: Story = {
  args: { ...Default.args },
  parameters: {
    uiState: { sidebarOpen: true },
  },
};
