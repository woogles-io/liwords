import { cleanup, render } from "@testing-library/react";
import { afterEach, expect, it } from "vitest";
import { AlignedScorePair } from "./score_pair";

afterEach(cleanup);

it("writes two scores around a hyphen, each in its own box", () => {
  const { container } = render(<AlignedScorePair first={428} second={86} />);
  expect(container.textContent).toBe("428-86");
  expect(
    container.querySelector(".aligned-score-pair-first")?.textContent,
  ).toBe("428");
  expect(
    container.querySelector(".aligned-score-pair-second")?.textContent,
  ).toBe("86");
});

it("brackets a negative score, as scorePair does", () => {
  const { container } = render(<AlignedScorePair first={-10} second={12} />);
  expect(container.textContent).toBe("(-10)-12");
});
