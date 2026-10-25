import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { ABSOLUTE_TIMES_KEY } from "./history_prefs";
import { ProfileTime } from "./profile_time";

describe("ProfileTime", () => {
  afterEach(() => {
    localStorage.removeItem(ABSOLUTE_TIMES_KEY);
  });

  it("shows how long ago, and a click turns every time into a date", () => {
    const at = new Date(Date.now() - 3 * 24 * 3600 * 1000);
    render(
      <>
        <ProfileTime at={at} />
        <ProfileTime at={at} />
      </>,
    );
    expect(screen.getAllByText("3 days ago")).toHaveLength(2);
    fireEvent.click(screen.getAllByText("3 days ago")[0]);
    expect(localStorage.getItem(ABSOLUTE_TIMES_KEY)).toBe("true");
    expect(screen.getAllByText(/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}$/)).toHaveLength(
      2,
    );
  });

  it("shows nothing without a time", () => {
    const { container } = render(<ProfileTime />);
    expect(container.textContent).toBe("");
  });
});
