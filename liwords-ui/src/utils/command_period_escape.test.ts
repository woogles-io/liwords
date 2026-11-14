import { afterEach, expect, it } from "vitest";
import { commandPeriodEscape } from "./command_period_escape";

const heard: Array<KeyboardEvent> = [];
const listen = (e: Event) => heard.push(e as KeyboardEvent);

afterEach(() => {
  heard.length = 0;
  document.removeEventListener("keydown", commandPeriodEscape, true);
  document.body.removeEventListener("keydown", listen);
});

const press = (init: KeyboardEventInit) => {
  const input = document.createElement("input");
  document.body.appendChild(input);
  const e = new KeyboardEvent("keydown", {
    bubbles: true,
    cancelable: true,
    ...init,
  });
  input.dispatchEvent(e);
  input.remove();
  return e;
};

it("sends Cmd+. on as an Escape keydown where it was pressed", () => {
  document.addEventListener("keydown", commandPeriodEscape, true);
  document.body.addEventListener("keydown", listen);
  const e = press({ key: ".", metaKey: true });
  expect(e.defaultPrevented).toBe(true);
  // the Cmd+. itself stops at the document; only the Escape goes on
  expect(heard.map((h) => [h.key, h.keyCode])).toEqual([["Escape", 27]]);
});

it("leaves every other key alone, Cmd+Shift+. and plain . included", () => {
  document.addEventListener("keydown", commandPeriodEscape, true);
  document.body.addEventListener("keydown", listen);
  press({ key: ".", metaKey: true, shiftKey: true });
  press({ key: "." });
  press({ key: ",", metaKey: true });
  expect(heard.map((h) => h.key)).toEqual([".", ".", ","]);
});
