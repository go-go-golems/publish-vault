import { createSlice, PayloadAction } from "@reduxjs/toolkit";

interface UIState {
  sidebarOpen: boolean;
  rightPanelOpen: boolean;
  searchQuery: string;
  activeNoteSlug: string | null;
  graphVisible: boolean;
}

const initialState: UIState = {
  sidebarOpen: true,
  rightPanelOpen: true,
  searchQuery: "",
  activeNoteSlug: null,
  graphVisible: false,
};

const uiSlice = createSlice({
  name: "ui",
  initialState,
  reducers: {
    toggleSidebar(state) {
      state.sidebarOpen = !state.sidebarOpen;
    },
    setSidebarOpen(state, action: PayloadAction<boolean>) {
      state.sidebarOpen = action.payload;
    },
    toggleRightPanel(state) {
      state.rightPanelOpen = !state.rightPanelOpen;
    },
    setRightPanelOpen(state, action: PayloadAction<boolean>) {
      state.rightPanelOpen = action.payload;
    },
    setSearchQuery(state, action: PayloadAction<string>) {
      state.searchQuery = action.payload;
    },
    setActiveNote(state, action: PayloadAction<string | null>) {
      state.activeNoteSlug = action.payload;
    },
    toggleGraph(state) {
      state.graphVisible = !state.graphVisible;
    },
  },
});

export const {
  toggleSidebar,
  setSidebarOpen,
  toggleRightPanel,
  setRightPanelOpen,
  setSearchQuery,
  setActiveNote,
  toggleGraph,
} = uiSlice.actions;

export default uiSlice.reducer;
