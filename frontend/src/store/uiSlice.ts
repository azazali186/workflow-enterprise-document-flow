import { createSlice, type PayloadAction } from '@reduxjs/toolkit';

export interface Toast {
  id: number;
  kind: 'success' | 'error' | 'info';
  title: string;
  message?: string;
}

interface UiState {
  sidebarOpen: boolean;
  sidebarCollapsed: boolean;
  toasts: Toast[];
  mobileSidebarOpen: boolean;
}

const initialState: UiState = {
  sidebarOpen: false,
  sidebarCollapsed: false,
  toasts: [],
  mobileSidebarOpen: false,
};

let nextToastId = 1;

const uiSlice = createSlice({
  name: 'ui',
  initialState,
  reducers: {
    toggleSidebar(state) {
      state.sidebarCollapsed = !state.sidebarCollapsed;
    },
    setMobileSidebar(state, action: PayloadAction<boolean>) {
      state.mobileSidebarOpen = action.payload;
    },
    pushToast(state, action: PayloadAction<Omit<Toast, 'id'>>) {
      state.toasts.push({ ...action.payload, id: nextToastId++ });
    },
    dismissToast(state, action: PayloadAction<number>) {
      state.toasts = state.toasts.filter((t) => t.id !== action.payload);
    },
  },
});

export const { toggleSidebar, setMobileSidebar, pushToast, dismissToast } = uiSlice.actions;
export default uiSlice.reducer;
