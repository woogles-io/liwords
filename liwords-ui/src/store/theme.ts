import { createSlice } from "@reduxjs/toolkit";
import type { PayloadAction } from "@reduxjs/toolkit";

export interface Theme {
  darkMode: boolean;
}

const initialState: Theme = {
  darkMode: localStorage?.getItem("darkMode") === "true",
};

export const themeSlice = createSlice({
  name: "theme",
  initialState,
  reducers: {
    setDarkMode: (state, action: PayloadAction<boolean>) => {
      state.darkMode = action.payload;
    },
  },
});

// Action creators are generated for each case reducer function
export const { setDarkMode } = themeSlice.actions;

export default themeSlice.reducer;
