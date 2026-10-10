import { render } from "@testing-library/react";
import { MantineProvider } from "@mantine/core";
import { describe, expect, it } from "vitest";
import { wooglesTheme, wooglesCssVariablesResolver } from "./woogles-theme";

/**
 * The palette lives in src/theme/tokens.scss as `--woogles-*` custom properties.
 * Mantine is wired to read from those rather than carry its own copy, so there
 * is one switch point for light/dark instead of two that can drift.
 *
 * If the resolver stops being applied -- or someone gives Mantine literal
 * colours "to make it work" -- the two libraries start disagreeing only in one
 * colour mode, which is the kind of thing nobody notices until it ships.
 */
function renderThemed() {
  return render(
    <MantineProvider
      theme={wooglesTheme}
      cssVariablesResolver={wooglesCssVariablesResolver}
      forceColorScheme="dark"
    >
      <div />
    </MantineProvider>,
  );
}

function injectedCss() {
  return Array.from(document.querySelectorAll("style"))
    .map((s) => s.textContent ?? "")
    .join("\n");
}

describe("woogles Mantine theme", () => {
  it("points Mantine's semantic colours at the --woogles-* tokens", () => {
    renderThemed();
    const css = injectedCss();
    expect(css).toContain(
      "--mantine-color-body: var(--woogles-color-background)",
    );
    expect(css).toContain(
      "--mantine-color-text: var(--woogles-color-gray-extreme)",
    );
    expect(css).toContain(
      "--mantine-primary-color-filled: var(--woogles-color-button)",
    );
  });

  it("stacks Mantine's popups above antd's while both coexist", () => {
    // antd's zIndexPopupBase is 1000, with 1100 and 2000 hardcoded in a few
    // places. Mantine ships 100-400, so without this a Mantine dialog opens
    // behind an antd one. Revert in Phase 9 when antd goes.
    renderThemed();
    const css = injectedCss();
    expect(css).toContain("--mantine-z-index-modal: 3200");
    expect(css).toContain("--mantine-z-index-popover: 3300");
  });

  it("emits no per-scheme colour values of its own", () => {
    // light/dark are deliberately empty in the resolver: tokens.scss already
    // switches the --woogles-* values via html.mode--dark. A second switch
    // point here could disagree with that one.
    const resolved = wooglesCssVariablesResolver(wooglesTheme as never);
    expect(resolved.light).toEqual({});
    expect(resolved.dark).toEqual({});
  });

  it("pins primary shade 6 to the existing Woogles blue", () => {
    // Mantine uses shade 6 for filled buttons and links by default; it must be
    // the same #11659e the SCSS has always used.
    expect(wooglesTheme.colors?.woogles?.[6]).toBe("#11659e");
  });
});
