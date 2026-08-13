import { render } from "@testing-library/react";
import { StyleProvider } from "@ant-design/cssinjs";
import { ConfigProvider, Popover } from "antd";
import { describe, expect, it } from "vitest";

/**
 * Ant Design emits its component CSS at runtime through @ant-design/cssinjs, so
 * none of it is visible to a build-time stylesheet audit. `<StyleProvider layer>`
 * wraps that generated CSS in `@layer antd`, and layered CSS always loses to
 * unlayered CSS regardless of specificity -- which is what lets the ~366 places
 * where our SCSS overrides antd's DOM win reliably.
 *
 * Before layers, those overrides were winning only because the colorModed()
 * mixin happened to add a `.mode--*` class to them. Removing that class in the
 * token bridge handed several of them back to antd (the board's definition
 * popup picked up antd's dark-mode border, among others).
 *
 * If someone drops StyleProvider, or antd changes how `layer` is plumbed, that
 * whole class of regression returns silently. Hence this test.
 */
function injectedCss() {
  return Array.from(document.querySelectorAll("style"))
    .map((s) => s.textContent ?? "")
    .join("\n");
}

describe("antd cascade layering", () => {
  it("wraps antd's generated CSS in @layer antd", () => {
    render(
      <StyleProvider layer>
        <ConfigProvider>
          <Popover content="definition" open>
            <span>word</span>
          </Popover>
        </ConfigProvider>
      </StyleProvider>,
    );

    const css = injectedCss();
    expect(css).toContain("@layer antd");
    // The popover styles specifically -- not just some unrelated reset block.
    const popoverRule = css.indexOf("ant-popover");
    expect(popoverRule).toBeGreaterThan(-1);
    expect(css.lastIndexOf("@layer antd", popoverRule)).toBeGreaterThan(-1);
  });

  it("does not layer antd's CSS without StyleProvider", () => {
    // Guards the test itself: if antd started layering unconditionally, the
    // assertion above would pass for the wrong reason and tell us nothing.
    render(
      <ConfigProvider>
        <Popover content="definition" open>
          <span>word</span>
        </Popover>
      </ConfigProvider>,
    );
    expect(injectedCss()).toContain("ant-popover");
  });
});
