import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { fromBinary } from "@bufbuild/protobuf";
import { describe, expect, it, vi } from "vitest";
import { FullTournamentDivisionsSchema } from "../gen/api/proto/ipc/tournament_pb";
import { ActionType } from "../actions/actions";
import {
  defaultTournamentState,
  Division,
  TournamentReducer,
} from "../store/reducers/tournament_reducer";
import { ftData } from "../store/reducers/testdata/tourney_1_divisions";
import { Standings } from "./standings";

// The standings table reads whichever round the picker is on. Rounds it must
// not render are the ones that have not been opened yet: the server computes
// standings cumulatively, so it answers for a future round with a copy of the
// current one, and rendering that would repeat the current numbers under the
// wrong heading.

// The real one is a dropdown that needs the login, match and pet contexts. Its
// "View scorecard" entry becomes a button here so the wiring can still be
// clicked.
vi.mock("../shared/usernameWithContext", () => ({
  UsernameWithContext: (props: {
    username: string;
    infoText?: string;
    handleInfoText?: () => void;
    omitProfileLink?: boolean;
    omitFriend?: boolean;
  }) => (
    <span
      data-omits={[
        props.omitProfileLink ? "profile" : "",
        props.omitFriend ? "friend" : "",
      ]
        .filter(Boolean)
        .join(",")}
    >
      {props.username}
      {props.infoText && (
        <button onClick={props.handleInfoText}>
          {props.infoText} for {props.username}
        </button>
      )}
    </span>
  ),
}));

const divisions = vi.hoisted(() => ({ value: {} as Record<string, Division> }));
const metadata = vi.hoisted(() => ({ value: {} as { irlMode?: boolean } }));

vi.mock("../store/store", () => ({
  useTournamentStoreContext: () => ({
    tournamentContext: { divisions: divisions.value, metadata: metadata.value },
  }),
}));

const toArr = (s: string) => {
  const bytes = new Uint8Array(Math.ceil(s.length / 2));
  for (let i = 0; i < bytes.length; i++) {
    bytes[i] = parseInt(s.substring(i * 2, i * 2 + 2), 16);
  }
  return bytes;
};

// The NWL division of this fixture has eight rounds of standings for three
// players, josh, lola and thedirector, and it includes byes.
const nwlDivision = (): Division => {
  const state = TournamentReducer(defaultTournamentState, {
    actionType: ActionType.SetDivisionsData,
    payload: {
      fullDivisions: fromBinary(FullTournamentDivisionsSchema, toArr(ftData)),
      loginState: {
        username: "josh",
        userID: "dcHy8DxjX3dmsCAaEMR5AH",
        loggedIn: true,
        connId: "conn-123",
        connectedToSocket: true,
      },
    },
  });
  return state.divisions["NWL"];
};

const renderAt = (
  currentRound: number,
  selectedRound: number,
  irlMode = false,
) => {
  divisions.value = { NWL: { ...nwlDivision(), currentRound } };
  metadata.value = { irlMode };
  render(<Standings selectedDivision="NWL" selectedRound={selectedRound} />);
};

const omitsFor = (username: string) =>
  screen.getByText(username).closest("span")?.getAttribute("data-omits");

// A cell's text without the stand-in scorecard button.
const cellText = (cell: HTMLElement) => {
  const clone = cell.cloneNode(true) as HTMLElement;
  clone.querySelectorAll("button").forEach((button) => button.remove());
  return clone.textContent?.trim();
};

// Every row as [rank, player, W, L, spread].
const rows = () =>
  screen
    .getAllByRole("row")
    .slice(1)
    .map((row) => within(row).getAllByRole("cell").map(cellText));

describe("standings for the selected round", () => {
  it("shows a past round rather than the current one", () => {
    renderAt(7, 2);
    expect(rows()).toEqual([
      ["1", "josh", "2", "0", "+100"],
      ["2", "lola", "1", "0", "+50"],
      ["3", "thedirector", "0", "0", "+0"],
    ]);
  });

  it("shows a different past round", () => {
    renderAt(7, 5);
    expect(rows()).toEqual([
      ["1", "josh", "5", "0", "+250"],
      ["2", "lola", "1", "0", "+50"],
      ["3", "thedirector", "0", "0", "+0"],
    ]);
  });

  it("shows the round that is open, whose games may still be running", () => {
    renderAt(7, 7);
    expect(rows()).toEqual([
      ["1", "josh", "5", "0", "+250"],
      ["2", "thedirector", "2", "0", "+100"],
      ["3", "lola", "1", "0", "+50"],
    ]);
  });

  it("declines a round that has not been opened", () => {
    renderAt(5, 6);
    expect(screen.getByText("Standings are not yet available.")).toBeVisible();
  });

  it("declines every round before the tournament starts", () => {
    renderAt(-1, 0);
    expect(screen.getByText("Standings are not yet available.")).toBeVisible();
  });

  it("shares a place between players the table cannot tell apart", () => {
    // After round 2 both josh and lola are 1-0 with a spread of 50.
    renderAt(7, 1);
    expect(rows()).toEqual([
      ["1=", "lola", "1", "0", "+50"],
      ["1=", "josh", "1", "0", "+50"],
      ["3", "thedirector", "0", "0", "+0"],
    ]);
  });

  it("shares a place below the leader too", () => {
    // Round 1 gave josh a bye, and left the other two on nothing at all.
    renderAt(7, 0);
    expect(rows()).toEqual([
      ["1", "josh", "1", "0", "+50"],
      ["2=", "lola", "0", "0", "+0"],
      ["2=", "thedirector", "0", "0", "+0"],
    ]);
  });

  it("offers no profile or friend for a name that is not an account", () => {
    // An in-real-life division holds real names, and /profile/Some%20Name is a
    // 404.
    renderAt(7, 2, true);
    expect(omitsFor("josh")).toBe("profile,friend");
  });

  it("keeps both for an online division", () => {
    renderAt(7, 2);
    expect(omitsFor("josh")).toBe("");
  });

  it("opens a scorecard covering the rounds up to the one on screen", async () => {
    renderAt(7, 2);
    await userEvent.click(screen.getByText("View scorecard for lola"));
    expect(screen.getByRole("dialog")).toHaveTextContent("rounds 1 to 3");
  });
});
