import { Button } from "@mantine/core";
import classes from "./mantine_components.module.css";

/**
 * Mantine component overrides, translated from the antd reskin mixins in
 * base.scss so migrated components inherit the Woogles look instead of each
 * area re-fixing it locally.
 *
 * Compare any change against /dev/components in a dev build, in both colour
 * modes. These values are reproductions of existing behaviour, not fresh design
 * decisions -- where antd does something unusual it is copied deliberately.
 *
 * Values come through `vars` wherever Mantine exposes a CSS variable for them,
 * since those are documented API. Only what has no variable goes in the CSS
 * module.
 *
 * NOT reproduced: @mixin button puts `margin: 0 6px` on every default button
 * and `6px 3px` on every primary one. Baking margins into a component is a
 * footgun -- it fights every layout that wants to control its own spacing, and
 * Mantine's idiom is an explicit <Group gap>. Each area that migrates buttons
 * needs to add spacing at the call site; expect to notice it there.
 */
export const components = {
  Button: Button.extend({
    classNames: { root: classes.button },
    vars: (_theme, props) => {
      const filled = props.variant === undefined || props.variant === "filled";
      return {
        root: {
          "--button-fz": "12px",
          "--button-height": "36px",
          "--button-padding-x": "18px",

          // Primary is a solid fill with a small radius; everything else is an
          // outline in primary-dark on the page background, square-cornered.
          "--button-radius": filled ? "3px" : "0",

          // Hover and rest are identical on purpose -- see the CSS module.
          ...(filled
            ? {
                "--button-bg": "var(--woogles-color-button)",
                "--button-hover": "var(--woogles-color-button)",
                "--button-color": "var(--woogles-color-button-text)",
                "--button-hover-color": "var(--woogles-color-button-text)",
                "--button-bd": "0",
              }
            : {
                "--button-bg": "var(--woogles-color-background)",
                "--button-hover": "var(--woogles-color-background)",
                "--button-color": "var(--woogles-color-primary-dark)",
                "--button-hover-color": "var(--woogles-color-primary-dark)",
                "--button-bd": "1px solid var(--woogles-color-primary-dark)",
              }),
        },
      };
    },
  }),
};
