import { act, cleanup, render } from "@testing-library/react";
import { afterEach, expect, it } from "vitest";
import { HookSquare, publishLabHooks } from "./lab_hooks";
import { LabHooksOverlay } from "./lab_hooks_overlay";

const publisher = Symbol("test");

afterEach(() => {
  act(() => publishLabHooks(publisher, null));
  cleanup();
});

it("draws every square in its place, across on top and down below", () => {
  const squares = new Array<HookSquare | undefined>(9);
  // a 3 by 3 board, hooks on the middle square of the top row and on the
  // bottom right one, the rest left out, as the analyzer publishes them;
  // nothing measures a square here, so each line is drawn whole
  squares[1] = { across: [..."ABDEFHJKLMNPTYZ"], down: ["A", "S"] };
  squares[8] = { across: null, down: [] };
  act(() => publishLabHooks(publisher, { dim: 3, squares }));
  const { container } = render(<LabHooksOverlay gridDim={3} />);
  const cells = [...container.querySelectorAll(".lab-hook")];
  expect(cells.length).toBe(9);
  expect(cells.map((c) => c.textContent)).toEqual([
    "",
    "ABDEFHJKLMNPTYZAS",
    "",
    "",
    "",
    "",
    "",
    "",
    "-",
  ]);
  expect(cells[1].querySelector(".lab-hook-across")?.textContent).toBe(
    "ABDEFHJKLMNPTYZ",
  );
  expect(cells[1].querySelector(".lab-hook-down")?.textContent).toBe("AS");
  // no letter fits the last one down: marked, the square left as it is
  expect(cells[8].querySelector(".lab-hook-down")?.textContent).toBe("-");
  expect(cells[8].className).toBe("lab-hook");
});

it("draws nothing for hooks of another board size", () => {
  act(() =>
    publishLabHooks(publisher, {
      dim: 15,
      squares: new Array(225).fill(undefined),
    }),
  );
  const { container } = render(<LabHooksOverlay gridDim={21} />);
  expect(container.querySelector(".lab-hooks")).toBeNull();
});
