/**
 * UI Store - Manages UI state across the application
 *
 * This store REPLACES the Redux store and uses the same localStorage keys
 * for backwards compatibility during migration.
 *
 * Uses Zustand with manual localStorage management to maintain compatibility
 * with legacy keys.
 */

import { create } from "zustand";
import { devtools } from "zustand/middleware";

export type ThemeMode = "light" | "dark";

interface UIState {
  // Theme
  themeMode: ThemeMode;
  setThemeMode: (mode: ThemeMode) => void;
  toggleTheme: () => void;

  // Board and tile modes (from legacy localStorage)
  boardMode: string;
  setBoardMode: (mode: string) => void;

  tileMode: string;
  setTileMode: (mode: string) => void;

  // Modals
  activeModal: string | null;
  openModal: (modalId: string) => void;
  closeModal: () => void;

  // UI preferences
  showNotations: boolean;
  setShowNotations: (show: boolean) => void;

  // Game UI preferences
  showPoolModal: boolean;
  setShowPoolModal: (show: boolean) => void;
}

/**
 * Initialize theme from legacy localStorage "darkMode" key
 */
const getInitialTheme = (): ThemeMode => {
  if (typeof window === "undefined") return "light";
  const darkMode = localStorage.getItem("darkMode");
  return darkMode === "true" ? "dark" : "light";
};

/**
 * Initialize board mode from legacy localStorage "userBoard" key
 */
const getInitialBoardMode = (): string => {
  if (typeof window === "undefined") return "default";
  return localStorage.getItem("userBoard") || "default";
};

/**
 * Initialize tile mode from legacy localStorage "userTile" key
 */
const getInitialTileMode = (): string => {
  if (typeof window === "undefined") return "default";
  return localStorage.getItem("userTile") || "default";
};

/**
 * Main UI store - replaces Redux store
 */
export const useUIStore = create<UIState>()(
  devtools(
    (set, get) => ({
      // Theme - uses legacy "darkMode" localStorage key
      themeMode: getInitialTheme(),
      setThemeMode: (mode) => {
        set({ themeMode: mode });
        updateBodyClass(mode);
        // Write to legacy localStorage key
        if (typeof window !== "undefined") {
          localStorage.setItem("darkMode", mode === "dark" ? "true" : "false");
        }
      },
      toggleTheme: () => {
        const newMode = get().themeMode === "light" ? "dark" : "light";
        set({ themeMode: newMode });
        updateBodyClass(newMode);
        // Write to legacy localStorage key
        if (typeof window !== "undefined") {
          localStorage.setItem(
            "darkMode",
            newMode === "dark" ? "true" : "false",
          );
        }
      },

      // Board mode - uses legacy "userBoard" localStorage key
      boardMode: getInitialBoardMode(),
      setBoardMode: (mode) => {
        set({ boardMode: mode });
        updateBoardModeClass(mode);
        // Write to legacy localStorage key
        if (typeof window !== "undefined") {
          localStorage.setItem("userBoard", mode);
        }
      },

      // Tile mode - uses legacy "userTile" localStorage key
      tileMode: getInitialTileMode(),
      setTileMode: (mode) => {
        set({ tileMode: mode });
        updateTileModeClass(mode);
        // Write to legacy localStorage key
        if (typeof window !== "undefined") {
          localStorage.setItem("userTile", mode);
        }
      },

      // Modals - no persistence needed
      activeModal: null,
      openModal: (modalId) => set({ activeModal: modalId }),
      closeModal: () => set({ activeModal: null }),

      // UI preferences - could add localStorage later if needed
      showNotations: true,
      setShowNotations: (show) => set({ showNotations: show }),

      // Game UI - no persistence needed
      showPoolModal: false,
      setShowPoolModal: (show) => set({ showPoolModal: show }),
    }),
    { name: "UIStore" }, // DevTools name
  ),
);

/**
 * Helper to update body class for theme mode
 * This maintains compatibility with legacy SCSS that uses .mode--dark and .mode--default
 */
function updateBodyClass(mode: ThemeMode) {
  if (typeof document === "undefined") return;

  document.body.classList.remove("mode--default", "mode--dark");
  document.body.classList.add(`mode--${mode === "dark" ? "dark" : "default"}`);
}

/**
 * Helper to update body class for board mode
 * Maintains compatibility with legacy SCSS board modes
 */
function updateBoardModeClass(mode: string) {
  if (typeof document === "undefined") return;

  // Remove all existing board mode classes
  const boardModeClasses = Array.from(document.body.classList).filter((c) =>
    c.startsWith("board--"),
  );
  boardModeClasses.forEach((c) => document.body.classList.remove(c));

  // Add new board mode class
  if (mode && mode !== "default") {
    document.body.classList.add(`board--${mode}`);
  }
}

/**
 * Helper to update body class for tile mode
 * Maintains compatibility with legacy SCSS tile modes
 */
function updateTileModeClass(mode: string) {
  if (typeof document === "undefined") return;

  // Remove all existing tile mode classes
  const tileModeClasses = Array.from(document.body.classList).filter((c) =>
    c.startsWith("tile--"),
  );
  tileModeClasses.forEach((c) => document.body.classList.remove(c));

  // Add new tile mode class
  if (mode && mode !== "default") {
    document.body.classList.add(`tile--${mode}`);
  }
}

/**
 * Selectors for common use cases
 * These can be used to only re-render when specific values change
 */
export const selectThemeMode = (state: UIState) => state.themeMode;
export const selectActiveModal = (state: UIState) => state.activeModal;
export const selectBoardMode = (state: UIState) => state.boardMode;
export const selectTileMode = (state: UIState) => state.tileMode;
