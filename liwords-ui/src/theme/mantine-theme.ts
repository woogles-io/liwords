/**
 * Mantine theme configuration
 *
 * This file creates the Mantine theme using our shared tokens.
 * It maps our design tokens to Mantine's theme system.
 */

import { createTheme, MantineColorsTuple, rem } from '@mantine/core';
import { tokens, spacing, typography, borderRadius, zIndex, type ThemeMode } from './tokens';

/**
 * Brand color palette (primary blue)
 * Mantine requires a 10-shade palette for each color
 */
const brandColors: MantineColorsTuple = [
  '#e2f8ff', // 0 - lightest
  '#c9f0ff', // 1
  '#91d5ff', // 2
  '#69c0ff', // 3
  '#40a9ff', // 4
  '#2786cb', // 5 - primary mid
  '#11659e', // 6 - primary dark (main brand color)
  '#0050b3', // 7
  '#003a8c', // 8
  '#002766', // 9 - darkest
];

/**
 * Secondary color palette (purple)
 */
const secondaryColors: MantineColorsTuple = [
  '#f8f2fc', // 0 - lightest
  '#d5cad6', // 1
  '#cfb7d1', // 2
  '#955f9a', // 3
  '#6b268b', // 4 - main secondary
  '#4d116a', // 5
  '#3d0d54', // 6
  '#2f0a41', // 7
  '#21072e', // 8
  '#13041b', // 9 - darkest
];

/**
 * Gray color palette
 */
const grayColors: MantineColorsTuple = [
  '#ffffff', // 0 - white
  '#eeeeee', // 1
  '#cccccc', // 2
  '#bebebe', // 3
  '#999999', // 4
  '#777777', // 5
  '#515151', // 6
  '#414141', // 7 - black
  '#313131', // 8
  '#282828', // 9 - darkest
];

/**
 * Success color palette (green - for timers, wins)
 */
const successColors: MantineColorsTuple = [
  '#e5ffdf', // 0 - lightest
  '#b2e49b', // 1
  '#7dcb6d', // 2
  '#52b23f', // 3
  '#449e2d', // 4 - main success
  '#3a8525', // 5
  '#306c1e', // 6
  '#265317', // 7
  '#24542d', // 8 - dark mode
  '#1a3a20', // 9 - darkest
];

/**
 * Warning color palette (yellow/orange - for low time, ties)
 */
const warningColors: MantineColorsTuple = [
  '#fbe5ae', // 0 - lightest
  '#f9d978', // 1
  '#f7cd42', // 2
  '#f5c10c', // 3
  '#f4b000', // 4 - main warning
  '#c69000', // 5
  '#987000', // 6
  '#6a5000', // 7
  '#3c3000', // 8
  '#0e1000', // 9 - darkest
];

/**
 * Error color palette (red - for time out, losses)
 */
const errorColors: MantineColorsTuple = [
  '#ffeaea', // 0 - lightest
  '#ffc9c9', // 1
  '#ffa8a8', // 2
  '#ff8787', // 3
  '#ff6b6b', // 4
  '#fa5252', // 5
  '#f03e3e', // 6
  '#e03131', // 7
  '#c92a2a', // 8
  '#a92e2e', // 9 - main error
];

/**
 * Main Mantine theme
 */
export const mantineTheme = createTheme({
  // Color scheme
  primaryColor: 'brand',
  colors: {
    brand: brandColors,
    secondary: secondaryColors,
    gray: grayColors,
    success: successColors,
    warning: warningColors,
    error: errorColors,
  },

  // Typography
  fontFamily: typography.fontFamily.default,
  fontFamilyMonospace: typography.fontFamily.monospace,
  headings: {
    fontFamily: typography.fontFamily.default,
    fontWeight: '700',
    sizes: {
      h1: { fontSize: rem(typography.fontSize.xxl * 1.5), lineHeight: '1.2' },
      h2: { fontSize: rem(typography.fontSize.xxl), lineHeight: '1.3' },
      h3: { fontSize: rem(typography.fontSize.xl), lineHeight: '1.4' },
      h4: { fontSize: rem(typography.fontSize.lg), lineHeight: '1.4' },
      h5: { fontSize: rem(typography.fontSize.md), lineHeight: '1.5' },
      h6: { fontSize: rem(typography.fontSize.sm), lineHeight: '1.5' },
    },
  },

  // Font sizes
  fontSizes: {
    xs: rem(typography.fontSize.xs),
    sm: rem(typography.fontSize.sm),
    md: rem(typography.fontSize.md),
    lg: rem(typography.fontSize.lg),
    xl: rem(typography.fontSize.xl),
  },

  // Spacing
  spacing: {
    xs: rem(spacing.xs),
    sm: rem(spacing.sm),
    md: rem(spacing.md),
    lg: rem(spacing.lg),
    xl: rem(spacing.xl),
  },

  // Border radius
  radius: {
    xs: rem(borderRadius.sm),
    sm: rem(borderRadius.md),
    md: rem(borderRadius.lg),
    lg: rem(borderRadius.xl),
    xl: rem(borderRadius.xl * 1.5),
  },

  // Default radius for components (0 to match current Ant Design)
  defaultRadius: 'xs',

  // Shadows
  shadows: {
    xs: '0 1px 3px rgba(0, 0, 0, 0.05)',
    sm: '0 1px 3px rgba(0, 0, 0, 0.12), 0 1px 2px rgba(0, 0, 0, 0.24)',
    md: '0px 0px 6px rgba(0, 0, 0, 0.15)',
    lg: '0px 0px 12px rgba(0, 0, 0, 0.15)',
    xl: '0 10px 40px rgba(0, 0, 0, 0.2)',
  },

  // Breakpoints
  breakpoints: {
    xs: '36em', // 576px
    sm: '48em', // 768px - mobile
    md: '64em', // 1024px - tablet
    lg: '80em', // 1280px - laptop
    xl: '90em', // 1440px - desktop
  },

  // Component-specific defaults
  components: {
    Button: {
      defaultProps: {
        radius: 0, // Match current design
      },
    },
    Input: {
      defaultProps: {
        radius: 'xs',
      },
    },
    Card: {
      defaultProps: {
        shadow: 'md',
        padding: 'md',
        radius: 'sm',
      },
    },
    Modal: {
      defaultProps: {
        radius: 'sm',
        zIndex: zIndex.modal,
      },
    },
    Popover: {
      defaultProps: {
        radius: 'sm',
        shadow: 'md',
        zIndex: zIndex.popover,
      },
    },
    Notification: {
      defaultProps: {
        radius: 'sm',
        zIndex: zIndex.notification,
      },
    },
  },

  // Custom properties accessible via theme.other
  other: {
    // Board and game-specific colors
    board: {
      dls: tokens.boardDls,
      dws: tokens.boardDws,
      tws: tokens.boardTws,
      tls: tokens.boardTls,
      qws: tokens.boardQws,
      qls: tokens.boardQls,
      empty: tokens.boardEmpty,
    },
    tile: {
      background: tokens.tileBackground,
      backgroundSecondary: tokens.tileBackgroundSecondary,
      backgroundTertiary: tokens.tileBackgroundTertiary,
      backgroundQuaternary: tokens.tileBackgroundQuaternary,
      blankText: tokens.tileBlankText,
      text: tokens.tileText,
      lastBackground: tokens.tileLastBackground,
      lastText: tokens.tileLastText,
      lastBlank: tokens.tileLastBlank,
    },
    timer: {
      light: tokens.timerLight,
      dark: tokens.timerDark,
      lowLight: tokens.timerLowLight,
      lowDark: tokens.timerLowDark,
      outLight: tokens.timerOutLight,
      outDark: tokens.timerOutDark,
    },
    // Z-index values
    zIndex,
  },
});

/**
 * Get theme-aware colors for a specific mode
 * Use this helper when you need mode-specific colors in components
 */
export const getThemeColor = (mode: ThemeMode) => {
  return {
    background: tokens.background[mode],
    cardBackground: tokens.cardBackground[mode],
    text: mode === 'light' ? tokens.black : tokens.white,
    primaryDark: tokens.primaryDark[mode],
    secondary: tokens.secondary[mode],
    // Add more as needed
  };
};
