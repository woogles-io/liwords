import {
  Button,
  Card,
  Input,
  InputWrapper,
  Modal,
  NumberInput,
} from "@mantine/core";
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
/** Mantine's Button defaults to the filled variant when none is given. */
const isFilled = (variant: string | undefined) =>
  variant === undefined || variant === "filled";

export const components = {
  Button: Button.extend({
    classNames: (_theme, props) => {
      const filled = isFilled(props.variant);
      const themed = props.color === undefined;
      return {
        root: [
          classes.button,
          filled ? classes.buttonFilled : classes.buttonDefault,
          filled && themed ? classes.buttonFilledThemed : "",
        ]
          .filter(Boolean)
          .join(" "),
      };
    },
    vars: (_theme, props) => {
      const filled = isFilled(props.variant);

      // An explicit `color` means the call site wants that colour, so only
      // geometry is imposed. Without this guard every <Button color="red">
      // in the app would be forced back to Woogles blue.
      const themed = props.color === undefined;

      return {
        root: {
          "--button-fz": "12px",
          "--button-height": "36px",
          "--button-padding-x": "18px",

          // Primary is a solid fill with a small radius; everything else is an
          // outline in primary-dark on the page background, square-cornered.
          "--button-radius": filled ? "3px" : "0",

          // Geometry, not colour: @mixin button gives the primary variant no
          // border at all, regardless of what colour the call site asked for.
          "--button-bd": filled
            ? "0"
            : "1px solid var(--woogles-color-primary-dark)",

          // Hover and rest are identical on purpose -- see the CSS module.
          ...(!themed
            ? {}
            : filled
              ? {
                  "--button-bg": "var(--woogles-color-button)",
                  "--button-hover": "var(--woogles-color-button)",
                  "--button-color": "var(--woogles-color-button-text)",
                  "--button-hover-color": "var(--woogles-color-button-text)",
                }
              : {
                  "--button-bg": "var(--woogles-color-background)",
                  "--button-hover": "var(--woogles-color-background)",
                  "--button-color": "var(--woogles-color-primary-dark)",
                  "--button-hover-color": "var(--woogles-color-primary-dark)",
                }),
        },
      };
    },
  }),

  Card: Card.extend({
    classNames: { root: classes.card },
    // Radius is a prop, not an author-settable variable. antd leaves Card at
    // its default 8px, and theme.defaultRadius is 0 for buttons' sake, so Card
    // has to opt back in explicitly.
    defaultProps: { radius: 8 },
  }),

  Modal: Modal.extend({
    classNames: {
      content: classes.modalContent,
      header: classes.modalHeader,
      title: classes.modalTitle,
      inner: classes.modalInner,
      overlay: classes.modalOverlay,
    },
    vars: () => ({
      root: {
        "--modal-radius": "8px",
      },
    }),
  }),

  /*
   * Theming Input covers TextInput, PasswordInput, Textarea, NumberInput and
   * Select, since all of them render Input underneath.
   */
  Input: Input.extend({
    // Colours go through the CSS module rather than `vars`: Mantine types
    // --input-bg and --input-bd as variant-resolved, not author-supplied. The
    // #b9b9b9 border is the antd Input token from themes.tsx -- not a
    // --woogles-* token, and deliberately identical in both colour modes.
    classNames: { input: classes.input },
  }),

  InputWrapper: InputWrapper.extend({
    classNames: { label: classes.inputLabel },
  }),

  NumberInput: NumberInput.extend({
    classNames: { input: classes.numberInput },
  }),
};
