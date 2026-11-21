import React from "react";
import { act, cleanup, render, screen } from "@testing-library/react";
import TentativeScore from "./tentative_score";

afterEach(() => {
  cleanup();
  localStorage.clear();
});

// the setting changed in another tab: that tab writes it, this one hears of it
const setElsewhere = (value: string) =>
  act(() => {
    localStorage.setItem("hidePool", value);
    window.dispatchEvent(
      new StorageEvent("storage", { key: "hidePool", newValue: value }),
    );
  });

it("shows or hides the play's score as soon as its setting changes", () => {
  render(<TentativeScore score={28} horizontal />);
  expect(screen.getByText("28")).toBeInTheDocument();
  setElsewhere("true");
  expect(screen.queryByText("28")).toBeNull();
  setElsewhere("false");
  expect(screen.getByText("28")).toBeInTheDocument();
});

it("shows nothing for a tile with no score", () => {
  const { container } = render(<TentativeScore score={undefined} horizontal />);
  expect(container).toBeEmptyDOMElement();
});
