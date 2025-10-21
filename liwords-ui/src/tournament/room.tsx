import React, { useState } from "react";

import { useCallback, useMemo } from "react";

import {
  useLoginStateStoreContext,
  useTournamentStoreContext,
} from "../store/store";
import { TopBar } from "../navigation/topbar";
import { Chat } from "../chat/chat";
import { TournamentInfo } from "./tournament_info";
import { sendAccept, sendSeek } from "../lobby/sought_game_interactions";
import { SoughtGame } from "../store/reducers/lobby_reducer";
import { ActionsPanel } from "./actions_panel";
import { CompetitorStatus } from "./competitor_status";
import "./room.scss";
import { useTourneyMetadata } from "./utils";
import { useSearchParams, useLocation } from "react-router";
import { OwnScoreEnterer } from "./enter_own_scores";
import { ConfigProvider } from "antd";
import { useQuery } from "@connectrpc/connect-query";
import { getSelfRoles } from "../gen/api/proto/user_service/user_service-AuthorizationService_connectquery";
import { useTournamentCompetitorState } from "../hooks/use_tournament_competitor_state";
import { readyForTournamentGame } from "./ready";
import { MonitoringSetup } from "./monitoring/monitoring_setup";

type Props = {
  sendSocketMsg: (msg: Uint8Array) => void;
  sendChat: (msg: string, chan: string) => void;
};

export const TournamentRoom = (props: Props) => {
  const [searchParams] = useSearchParams();
  const location = useLocation();

  const { loginState } = useLoginStateStoreContext();
  const { tournamentContext, dispatchTournamentContext } =
    useTournamentStoreContext();
  const { loggedIn, username, userID } = loginState;
  const competitorState = useTournamentCompetitorState();
  const { isRegistered } = competitorState;
  const { sendSocketMsg } = props;
  const { path } = loginState;
  const [badTournament, setBadTournament] = useState(false);
  const [selectedGameTab, setSelectedGameTab] = useState("GAMES");

  const { data: selfRoles } = useQuery(
    getSelfRoles,
    {},
    { enabled: loginState.loggedIn },
  );

  // Strip /monitoring suffix from path if present for metadata fetch
  const metadataPath = path.endsWith("/monitoring")
    ? path.slice(0, -"/monitoring".length)
    : path;

  useTourneyMetadata(
    metadataPath,
    "",
    dispatchTournamentContext,
    loginState,
    setBadTournament,
  );

  const tournamentID = useMemo(() => {
    return tournamentContext.metadata.id;
  }, [tournamentContext.metadata]);

  // Should be more like "amdirector"
  const isDirector = useMemo(() => {
    return tournamentContext.directors.includes(username);
  }, [tournamentContext.directors, username]);

  const canManageTournaments = useMemo(() => {
    return !!(
      selfRoles?.roles.includes("Admin") ||
      selfRoles?.roles.includes("Tournament Manager")
    );
  }, [selfRoles?.roles]);

  const handleNewGame = useCallback(
    (seekID: string) => {
      sendAccept(seekID, sendSocketMsg);
    },
    [sendSocketMsg],
  );
  const onSeekSubmit = useCallback(
    (g: SoughtGame) => {
      sendSeek(g, sendSocketMsg);
    },
    [sendSocketMsg],
  );

  if (badTournament) {
    return (
      <>
        <TopBar />
        <div className="lobby">
          <h3>You tried to access a non-existing page.</h3>
        </div>
      </>
    );
  }

  // Check if we're on the monitoring page (before tournamentID check)
  if (location.pathname.endsWith("/monitoring")) {
    return (
      <>
        <TopBar />
        <MonitoringSetup />
      </>
    );
  }

  if (!tournamentID) {
    return (
      <>
        <TopBar />
      </>
    );
  }

  if (searchParams.get("es") != null) {
    return (
      <>
        <OwnScoreEnterer truncatedID={searchParams.get("es") ?? ""} />
        <div style={{ marginTop: 400 }}></div>
      </>
    );
  }

  return (
    <>
      <TopBar />
      <div className={`lobby room ${isRegistered ? " competitor" : ""}`}>
        <div className="chat-area">
          <Chat
            sendChat={props.sendChat}
            defaultChannel={`chat.tournament.${tournamentID}`}
            defaultDescription={tournamentContext.metadata.name}
            highlight={tournamentContext.directors}
            highlightText="Director"
            tournamentID={tournamentID}
          />
          {isRegistered && (
            <CompetitorStatus
              sendReady={() =>
                readyForTournamentGame(
                  sendSocketMsg,
                  tournamentContext.metadata.id,
                  competitorState,
                )
              }
            />
          )}
        </div>
        <ConfigProvider
          theme={{
            components: {
              Dropdown: {
                paddingBlock: 5,
                paddingXS: 0,
                paddingXXS: 0,
              },
            },
          }}
        >
          <ActionsPanel
            selectedGameTab={selectedGameTab}
            setSelectedGameTab={setSelectedGameTab}
            isDirector={isDirector}
            canManageTournaments={canManageTournaments}
            tournamentID={tournamentID}
            onSeekSubmit={onSeekSubmit}
            loggedIn={loggedIn}
            newGame={handleNewGame}
            username={username}
            userID={userID}
            sendReady={() =>
              readyForTournamentGame(
                sendSocketMsg,
                tournamentContext.metadata.id,
                competitorState,
              )
            }
            showFirst={tournamentContext.metadata.irlMode}
          />
        </ConfigProvider>
        <TournamentInfo
          setSelectedGameTab={setSelectedGameTab}
          sendSocketMsg={sendSocketMsg}
        />
      </div>
    </>
  );
};
