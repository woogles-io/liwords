import { App, ConfigProvider } from "antd";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import * as path from "node:path";
import * as sass from "sass";
import { useEffect } from "react";
import { beforeAll, describe, expect, it, vi } from "vitest";
import { liwordsDarkTheme, liwordsDefaultTheme } from "../themes";

// Covers the "Computer analysis ready" toast in socket_handlers.ts. Both halves
// of it broke once already (#1917) and neither is visible to tsc or eslint, so
// they are pinned here.

const srcDir = path.join(__dirname, "..");

beforeAll(() => {
  const { css } = sass.compile(path.join(srcDir, "App.scss"), {
    loadPaths: [srcDir],
  });
  const style = document.createElement("style");
  style.textContent = css;
  document.head.appendChild(style);
});

const toRgb = (hex: string) => {
  const n = parseInt(hex.replace("#", ""), 16);
  return `rgb(${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255})`;
};

type Handlers = {
  onClick?: (event?: React.MouseEvent<HTMLDivElement>) => void;
  onAuxClick?: (event: React.MouseEvent<HTMLDivElement>) => void;
};

const Toast = (props: Handlers) => {
  const { notification } = App.useApp();
  useEffect(() => {
    notification.success({
      message: "Computer analysis ready",
      description: "Click to view the analysis",
      duration: 0,
      onClick: props.onClick,
      props: { onAuxClick: props.onAuxClick },
    });
  }, [notification, props]);
  return null;
};

const showToast = async (mode: "dark" | "default", handlers: Handlers = {}) => {
  const theme = mode === "dark" ? liwordsDarkTheme : liwordsDefaultTheme;
  document.body.className = `mode--${mode}`;
  render(
    <ConfigProvider theme={theme}>
      <App>
        <Toast {...handlers} />
      </App>
    </ConfigProvider>,
  );
  return {
    message: await screen.findByText("Computer analysis ready"),
    description: screen.getByText("Click to view the analysis"),
    background: toRgb(theme.components.Notification.colorBgElevated),
  };
};

// Toasts render in a portal at body level, where the global `a` rule in
// App.scss resolves to $primary-dark -- which is exactly the notice background
// in both modes. So toast text rendered as a link is invisible, which is what
// #1917 shipped. antd paints the notice background via cssinjs `:where()`
// selectors that jsdom cannot resolve, so the background is derived from the
// token that produces it rather than read back off the element.
describe.each(["dark", "default"] as const)(
  "analysis toast in %s mode",
  (mode) => {
    it("does not paint the toast text the color of the toast itself", async () => {
      const { message, description, background } = await showToast(mode);

      expect(getComputedStyle(message).color).not.toBe(background);
      expect(getComputedStyle(description).color).not.toBe(background);
    });
  },
);

// The toast opens the analysis on click, and a new tab on modifier/middle
// click. Both rely on antd behavior its types do not promise.
describe("analysis toast click wiring", () => {
  it("hands a real mouse event to antd's onClick, which antd types as zero-arg", async () => {
    const onClick = vi.fn();
    const { message } = await showToast("dark", { onClick });

    await userEvent.click(message);

    expect(onClick).toHaveBeenCalledTimes(1);
    // The optional-parameter signature is only sound if the event is real.
    expect(onClick.mock.calls[0][0]?.ctrlKey).toBe(false);
  });

  it("reports modifier keys, so ctrl-click can open a new tab", async () => {
    const onClick = vi.fn();
    const { message } = await showToast("dark", { onClick });

    const user = userEvent.setup();
    await user.keyboard("{Control>}");
    await user.click(message);

    expect(onClick.mock.calls[0][0].ctrlKey).toBe(true);
  });

  it("delivers onAuxClick through `props`, which antd does not clobber", async () => {
    const onAuxClick = vi.fn();
    const { message } = await showToast("dark", { onAuxClick });

    await userEvent.pointer({ target: message, keys: "[MouseMiddle]" });

    expect(onAuxClick).toHaveBeenCalledTimes(1);
    expect(onAuxClick.mock.calls[0][0].button).toBe(1);
  });

  it("does not fire the toast's onClick when the close button is clicked", async () => {
    const onClick = vi.fn();
    await showToast("dark", { onClick });

    await userEvent.click(screen.getByLabelText("Close"));

    expect(onClick).not.toHaveBeenCalled();
  });
});
