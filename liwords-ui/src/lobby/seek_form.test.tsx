import { create } from "@bufbuild/protobuf";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, render, screen } from "@testing-library/react";
import { App } from "antd";
import React, { useEffect, useState } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, expect, it } from "vitest";
import { ActionType } from "../actions/actions";
import {
  GameRequestSchema,
  GameRulesSchema,
  RatingMode,
} from "../gen/api/proto/ipc/omgwords_pb";
import { TournamentMetadataSchema } from "../gen/api/proto/tournament_service/tournament_service_pb";
import { ChallengeRule } from "../gen/api/proto/vendored/macondo/macondo_pb";
import { Store, useTournamentStoreContext } from "../store/store";
import { SeekForm } from "./seek_form";

// What the seek form starts with: unrated games, unless the player picked
// rated before and the form saved that. In a club room whose director lets
// players pick, the choices saved by the form are kept, unrated and zero
// overtime included.

type FormProps = React.ComponentProps<typeof SeekForm>;

const lobbySeek: FormProps = {
  id: "open-seek",
  loggedIn: true,
  onFormSubmit: () => {},
  showFriendInput: false,
};

afterEach(() => {
  cleanup();
  localStorage.clear();
  formProps = lobbySeek;
});

let dispatchTournament: ReturnType<
  typeof useTournamentStoreContext
>["dispatchTournamentContext"] = () => {};
let showForm: (show: boolean) => void = () => {};
let formProps = lobbySeek;

// Mounts the form only when told to, because it reads the store and the
// saved choices once, when it mounts.
const Probe = () => {
  const { dispatchTournamentContext } = useTournamentStoreContext();
  const [show, setShow] = useState(false);
  useEffect(() => {
    dispatchTournament = dispatchTournamentContext;
    showForm = setShow;
  }, [dispatchTournamentContext]);
  return show ? <SeekForm {...formProps} /> : null;
};

const renderStore = () =>
  render(
    <MemoryRouter initialEntries={["/"]}>
      <Store>
        <QueryClientProvider client={new QueryClient()}>
          <App>
            <Probe />
          </App>
        </QueryClientProvider>
      </Store>
    </MemoryRouter>,
  );

const openForm = () => {
  act(() => {
    showForm(true);
  });
};

const shownRatingType = () =>
  screen
    .getByLabelText("Rating type")
    .closest(".ant-select")
    ?.querySelector(".ant-select-selection-item")?.textContent;

it("starts on unrated games", () => {
  renderStore();
  openForm();
  expect(shownRatingType()).toBe("Unrated");
});

it("keeps rated when the player picked it before", () => {
  localStorage.setItem("lastSeekForm", JSON.stringify({ rated: true }));
  renderStore();
  openForm();
  expect(shownRatingType()).toBe("Rated");
});

it("keeps saved unrated, shortest time and no overtime in a club", () => {
  localStorage.setItem(
    "lastMatchForm",
    JSON.stringify({
      extratime: 0,
      incOrOT: "overtime",
      initialtimeslider: 0,
      rated: false,
    }),
  );
  renderStore();
  act(() => {
    dispatchTournament({
      actionType: ActionType.SetTourneyMetadata,
      payload: {
        directors: [],
        metadata: create(TournamentMetadataSchema, {
          id: "club",
          defaultClubSettings: create(GameRequestSchema, {
            challengeRule: ChallengeRule.FIVE_POINT,
            initialTimeSeconds: 1200,
            lexicon: "CSW24",
            maxOvertimeMinutes: 1,
            ratingMode: RatingMode.RATED,
            rules: create(GameRulesSchema, { variantName: "classic" }),
          }),
          freeformClubSettingFields: ["rating_mode", "time"],
        }),
      },
    });
  });
  formProps = {
    ...lobbySeek,
    id: "match-seek",
    showFriendInput: true,
    tournamentID: "club",
  };
  openForm();
  expect(shownRatingType()).toBe("Unrated");
  expect(screen.getByLabelText("Overtime")).toHaveValue("0");
  expect(screen.getByRole("slider")).toHaveAttribute("aria-valuenow", "0");
});
