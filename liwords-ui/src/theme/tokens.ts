/**
 * Shared theme tokens for Liwords
 *
 * This file defines all color, spacing, typography, and other design tokens
 * that can be consumed by both the legacy Ant Design theme and the new Mantine theme.
 *
 * All colors are defined per-mode (light/dark) to support theming.
 */

export type ThemeMode = 'light' | 'dark';

interface ColorToken {
  light: string;
  dark: string;
}

interface ThemeTokens {
  // Core colors
  white: string; // Always white
  black: string; // Always dark

  // Backgrounds
  background: ColorToken;
  offBackground: ColorToken;
  cardBackground: ColorToken;

  // Grays
  graySubtle: ColorToken;
  grayMedium: ColorToken;
  grayDark: ColorToken;
  grayExtreme: ColorToken;
  grid: ColorToken;

  // Primary colors (brand blue)
  primaryLight: ColorToken;
  primaryMiddle: ColorToken;
  primaryMidDark: ColorToken;
  primaryDark: ColorToken;

  // Secondary colors (purple)
  secondary: ColorToken;
  secondaryDark: ColorToken;
  secondaryLight: ColorToken;
  secondaryLighter: ColorToken;
  secondaryMedium: ColorToken;

  // UI elements
  button: ColorToken;
  buttonText: ColorToken;
  logo: ColorToken;
  logoText: ColorToken;
  header: ColorToken;
  footer: ColorToken;
  footerLogo: ColorToken;
  footerText: ColorToken;
  footerW: ColorToken;
  listHeader: ColorToken;
  listHeaderText: ColorToken;
  columnSorter: ColorToken;

  // Board colors (premium squares)
  boardDls: ColorToken; // Double letter score
  boardDws: ColorToken; // Double word score
  boardTws: ColorToken; // Triple word score
  boardTls: ColorToken; // Triple letter score
  boardQws: ColorToken; // Quadruple word score
  boardQls: ColorToken; // Quadruple letter score
  boardEmpty: ColorToken;

  // Tile colors
  tileBackground: ColorToken;
  tileBackgroundSecondary: ColorToken;
  tileBackgroundTertiary: ColorToken;
  tileBackgroundQuaternary: ColorToken;
  tileBlankText: ColorToken;
  tileText: ColorToken;
  tileLastBackground: ColorToken;
  tileLastText: ColorToken;
  tileLastBlank: ColorToken;

  // Timer colors
  timerLight: ColorToken;
  timerDark: ColorToken;
  timerLowLight: ColorToken;
  timerLowDark: ColorToken;
  timerOutLight: ColorToken;
  timerOutDark: ColorToken;

  // Profile game result colors
  profileGameWin: ColorToken;
  profileGameTie: ColorToken;
  profileGameLoss: ColorToken;

  // Shadows
  shadow: ColorToken;
  shadowLower: ColorToken;
  boardShadow: ColorToken;
}

/**
 * Core theme tokens derived from existing SCSS color system
 */
export const tokens: ThemeTokens = {
  // Always the same
  white: '#ffffff',
  black: '#414141',

  // Backgrounds
  background: { light: '#ffffff', dark: '#282828' },
  offBackground: { light: '#eeeeee', dark: '#313131' },
  cardBackground: { light: '#ffffff', dark: '#3a3a3a' },

  // Grays
  graySubtle: { light: '#bebebe', dark: '#515151' },
  grayMedium: { light: '#999999', dark: '#cccccc' },
  grayDark: { light: '#777777', dark: '#dddddd' },
  grayExtreme: { light: '#414141', dark: '#ffffff' },
  grid: { light: '#c3c3c3', dark: '#232323' },

  // Primary (blue)
  primaryLight: { light: '#e2f8ff', dark: '#135380' },
  primaryMiddle: { light: '#c9f0ff', dark: '#2786cb' },
  primaryMidDark: { light: '#2786cb', dark: '#7cc4e3' },
  primaryDark: { light: '#11659e', dark: '#c9f0ff' },

  // Secondary (purple)
  secondary: { light: '#6b268b', dark: '#d79fdd' },
  secondaryDark: { light: '#4d116a', dark: '#f4b6fb' },
  secondaryLight: { light: '#d5cad6', dark: '#6b268b' },
  secondaryLighter: { light: '#f8f2fc', dark: '#2f1b39' },
  secondaryMedium: { light: '#955f9a', dark: '#955f9a' },

  // UI elements
  button: { light: '#11659e', dark: '#7cc4e3' },
  buttonText: { light: '#ffffff', dark: '#313131' },
  logo: { light: '#11659e', dark: '#7cc4e3' },
  logoText: { light: '#c9f0ff', dark: '#135380' },
  header: { light: '#ffffff', dark: '#282828' },
  footer: { light: '#11659e', dark: '#2f2f2f' },
  footerLogo: { light: '#c9f0ff', dark: '#7cc4e3' },
  footerText: { light: '#ffffff', dark: '#ffffff' },
  footerW: { light: '#116593', dark: '#135380' },
  listHeader: { light: '#c9f0ff', dark: '#313131' },
  listHeaderText: { light: '#414141', dark: '#b2dcf0' },
  columnSorter: { light: '#414141', dark: '#bfbfbf' },

  // Board premium squares
  boardDls: { light: '#b9e7f5', dark: '#6dadc9' },
  boardDws: { light: '#f6c0c0', dark: '#a9545a' },
  boardTws: { light: '#a92e2e', dark: '#6b2125' },
  boardTls: { light: '#3b88ca', dark: '#115d92' },
  boardQws: { light: '#693072', dark: '#5f3367' },
  boardQls: { light: '#ac8bb0', dark: '#96729e' },
  boardEmpty: { light: '#ffffff', dark: '#313131' },

  // Tile colors
  tileBackground: { light: '#6b268b', dark: '#baadbb' },
  tileBackgroundSecondary: { light: '#cfb7d1', dark: '#643e69' },
  tileBackgroundTertiary: { light: '#955f9a', dark: '#b094b3' },
  tileBackgroundQuaternary: { light: '#dec5e4', dark: '#5d4269' },
  tileBlankText: { light: '#6b268b', dark: '#baadbb' },
  tileText: { light: '#ffffff', dark: '#313131' },
  tileLastBackground: { light: '#f4b000', dark: '#fbe5ae' },
  tileLastText: { light: '#414141', dark: '#414141' },
  tileLastBlank: { light: '#414141', dark: '#414141' },

  // Timer colors
  timerLight: { light: '#e5ffdf', dark: '#24542d' },
  timerDark: { light: '#449e2d', dark: '#b2e49b' },
  timerLowLight: { light: '#fbe5ae', dark: '#fbe5ae' },
  timerLowDark: { light: '#f4b000', dark: '#f4b000' },
  timerOutLight: { light: '#ffeaea', dark: '#561f22' },
  timerOutDark: { light: '#a92e2e', dark: '#ce5f66' },

  // Profile game results
  profileGameWin: { light: '#e5ffdf', dark: '#24542d' },
  profileGameTie: { light: '#fbe5ae', dark: '#f4b000' },
  profileGameLoss: { light: '#ffeaea', dark: '#561f22' },

  // Shadows
  shadow: { light: '0px 0px 12px rgba(0, 0, 0, 0.1)', dark: '0px 0px 12px rgba(0, 0, 0, 0.2)' },
  shadowLower: { light: '0px 0px 6px rgba(0, 0, 0, 0.1)', dark: '0px 0px 6px rgba(0, 0, 0, 0.2)' },
  boardShadow: { light: '0 0 6px #777777', dark: '0 0 6px #1a1a1a' },
};

/**
 * Spacing scale (in pixels)
 */
export const spacing = {
  xs: 4,
  sm: 8,
  md: 16,
  lg: 24,
  xl: 32,
  xxl: 48,
};

/**
 * Typography scale
 */
export const typography = {
  fontFamily: {
    default: '"Mulish", -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif',
    decorative: '"Fjalla One", sans-serif',
    monospace: '"Courier Prime", monospace',
    tile: '"Roboto Mono", monospace',
  },
  fontSize: {
    xs: 12,
    sm: 14,
    md: 16,
    lg: 18,
    xl: 20,
    xxl: 24,
  },
  lineHeight: {
    xs: 1.2,
    sm: 1.4,
    md: 1.5,
    lg: 1.6,
    xl: 1.8,
  },
};

/**
 * Breakpoints (in pixels)
 */
export const breakpoints = {
  mobile: 768,
  tablet: 1024,
  laptop: 1280,
  desktop: 1440,
  desktopL: 1600,
};

/**
 * Z-index scale
 */
export const zIndex = {
  base: 0,
  dropdown: 1000,
  sticky: 1100,
  modal: 2000,
  popover: 1500,
  notification: 2500,
  tooltip: 3000,
};

/**
 * Border radius scale
 */
export const borderRadius = {
  none: 0,
  sm: 2,
  md: 4,
  lg: 8,
  xl: 12,
  full: 9999,
};

/**
 * Helper function to get a color value for a specific mode
 */
export const getColor = (token: ColorToken, mode: ThemeMode): string => {
  return token[mode];
};

/**
 * Helper function to get all colors for a specific mode
 */
export const getThemeColors = (mode: ThemeMode) => {
  return Object.entries(tokens).reduce((acc, [key, value]) => {
    if (typeof value === 'object' && 'light' in value && 'dark' in value) {
      acc[key] = value[mode];
    } else {
      acc[key] = value;
    }
    return acc;
  }, {} as Record<string, string>);
};
