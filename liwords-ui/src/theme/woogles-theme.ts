import { components } from "./components";
import {
  createTheme,
  type CSSVariablesResolver,
  type MantineColorsTuple,
} from "@mantine/core";

/**
 * Mantine theme for the antd -> Mantine migration.
 *
 * The design tokens are NOT defined here. `src/theme/tokens.scss` already emits
 * the whole palette as `--woogles-*` custom properties on :root and switches
 * them per colour mode. This file points Mantine's variables at those, so both
 * libraries read from one source during the overlap and there is a single
 * switch point rather than two that can drift apart.
 */

/**
 * Shade 6 is Mantine's default filled shade, so it is pinned to #11659e -- the
 * exact blue already used for buttons and links. The rest is a lightness ramp
 * through the same hue.
 */
const wooglesBlue: MantineColorsTuple = [
  "#eff6fb",
  "#d6e8f5",
  "#abd2ed",
  "#75b8e5",
  "#3d9ee0",
  "#1a81c6",
  "#11659e",
  "#0e5382",
  "#0b4165",
  "#08304c",
];

export const wooglesTheme = createTheme({
  fontFamily: "Mulish, sans-serif",
  fontFamilyMonospace: '"Courier Prime", monospace',
  headings: { fontFamily: "Mulish, sans-serif" },

  colors: { woogles: wooglesBlue },

  components,
  primaryColor: "woogles",

  // antd's Button token sets borderRadius: 0. Match it so Mantine buttons do
  // not look out of place next to their antd neighbours during the overlap.
  defaultRadius: 0,

  // Mirrors base.scss: $screen-mobile-min 768, tablet 1024, laptop 1280,
  // desktop 1440. Kept in sync with postcss.config.cjs, which feeds the same
  // numbers to Mantine's responsive PostCSS mixins.
  breakpoints: {
    xs: "36em", // 576px -- Mantine default, no SCSS equivalent
    sm: "48em", // 768px -- $screen-mobile-min
    md: "64em", // 1024px -- $screen-tablet-min
    lg: "80em", // 1280px -- $screen-laptop-min
    xl: "90em", // 1440px -- $screen-desktop-min
  },
});

/**
 * One-directional adapter: Mantine's semantic variables read from ours.
 *
 * `light` and `dark` are intentionally empty. The `--woogles-*` values already
 * swap themselves via `html.mode--dark` in tokens.scss, so emitting per-scheme
 * values here too would create a second switch point that could disagree with
 * the first.
 */
export const wooglesCssVariablesResolver: CSSVariablesResolver = () => ({
  variables: {
    "--mantine-color-body": "var(--woogles-color-background)",
    "--mantine-color-text": "var(--woogles-color-gray-extreme)",
    "--mantine-color-dimmed": "var(--woogles-color-gray-medium)",
    "--mantine-color-anchor": "var(--woogles-color-primary-dark)",
    "--mantine-color-default": "var(--woogles-color-card-background)",
    "--mantine-color-default-hover": "var(--woogles-color-off-background)",
    "--mantine-color-default-border": "var(--woogles-color-gray-subtle)",
    "--mantine-color-error": "var(--woogles-color-timer-out-dark)",

    "--mantine-primary-color-filled": "var(--woogles-color-button)",
    "--mantine-primary-color-filled-hover":
      "var(--woogles-color-primary-midDark)",
    "--mantine-primary-color-contrast": "var(--woogles-color-button-text)",
    "--mantine-primary-color-light": "var(--woogles-color-primary-light)",

    // Mantine defaults are 100/200/300/400, all well below antd's popup stack
    // (zIndexPopupBase 1000, with 1100 and 2000 hardcoded in a few places). A
    // Mantine dialog has to be able to open on top of an antd one while both
    // coexist, since outer shells migrate before inner ones. Revert to
    // Mantine's defaults in Phase 9 once antd is gone.
    "--mantine-z-index-app": "3000",
    "--mantine-z-index-overlay": "3100",
    "--mantine-z-index-modal": "3200",
    "--mantine-z-index-popover": "3300",
  },
  light: {},
  dark: {},
});
