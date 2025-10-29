import React, { useState, useEffect } from "react";

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
import { useSearchParams } from "react-router";
import { OwnScoreEnterer } from "./enter_own_scores";
import { ConfigProvider } from "antd";
import { useQuery } from "@connectrpc/connect-query";
import { getSelfRoles } from "../gen/api/proto/user_service/user_service-AuthorizationService_connectquery";
import { useTournamentCompetitorState } from "../hooks/use_tournament_competitor_state";
import { readyForTournamentGame } from "./ready";
import { MonitoringModal } from "./monitoring/monitoring_modal";
import { DirectorDashboardModal } from "./monitoring/director_dashboard_modal";
import { MonitoringWidget } from "./monitoring/monitoring_widget";
import { TournamentService } from "../gen/api/proto/tournament_service/tournament_service_pb";
import { flashError, useClient } from "../utils/hooks/connect";
import { ActionType } from "../actions/actions";
import { MonitoringData } from "./monitoring/types";

type Props = {
  sendSocketMsg: (msg: Uint8Array) => void;
  sendChat: (msg: string, chan: string) => void;
};

export const TournamentRoom = (props: Props) => {
  const [searchParams, setSearchParams] = useSearchParams();

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
  const tClient = useClient(TournamentService);

  // Modal visibility from URL parameters
  const monitoringModalVisible = searchParams.get("monitoring") === "true";
  const directorDashboardModalVisible =
    searchParams.get("director-dashboard") === "true";

  // Modal close handlers - remove URL parameter
  const closeMonitoringModal = useCallback(() => {
    const newParams = new URLSearchParams(searchParams);
    newParams.delete("monitoring");
    setSearchParams(newParams);
  }, [searchParams, setSearchParams]);

  const closeDirectorDashboardModal = useCallback(() => {
    const newParams = new URLSearchParams(searchParams);
    newParams.delete("director-dashboard");
    setSearchParams(newParams);
  }, [searchParams, setSearchParams]);

  const { data: selfRoles } = useQuery(
    getSelfRoles,
    {},
    { enabled: loginState.loggedIn },
  );

  useTourneyMetadata(
    path,
    "",
    dispatchTournamentContext,
    loginState,
    setBadTournament,
  );

  const tournamentID = useMemo(() => {
    return tournamentContext.metadata.id;
  }, [tournamentContext.metadata]);

  // Fetch monitoring data on initial load if tournament requires monitoring
  useEffect(() => {
    if (
      !tournamentContext.metadata.id ||
      !loginState.loggedIn ||
      !tournamentContext.metadata.monitored
    ) {
      return;
    }

    const fetchMonitoringData = async () => {
      try {
        const response = await tClient.getTournamentMonitoring({
          tournamentId: tournamentContext.metadata.id,
        });

        // Convert to frontend format
        const data: MonitoringData[] = response.participants.map((p) => ({
          userId: p.userId,
          username: p.username,
          cameraKey: p.cameraKey,
          screenshotKey: p.screenshotKey,
          cameraStatus: p.cameraStatus,
          cameraTimestamp: p.cameraTimestamp
            ? new Date(
                Number(p.cameraTimestamp.seconds) * 1000 +
                  Number(p.cameraTimestamp.nanos) / 1000000,
              )
            : null,
          screenshotStatus: p.screenshotStatus,
          screenshotTimestamp: p.screenshotTimestamp
            ? new Date(
                Number(p.screenshotTimestamp.seconds) * 1000 +
                  Number(p.screenshotTimestamp.nanos) / 1000000,
              )
            : null,
        }));

        // Update tournament context with monitoring data
        dispatchTournamentContext({
          actionType: ActionType.SetMonitoringData,
          payload: data.reduce(
            (acc, d) => {
              acc[d.userId] = d;
              return acc;
            },
            {} as { [userId: string]: MonitoringData },
          ),
        });
      } catch (e) {
        flashError(e);
      }
    };

    fetchMonitoringData();
  }, [
    tournamentContext.metadata.id,
    tournamentContext.metadata.monitored,
    loginState.loggedIn,
    tClient,
    dispatchTournamentContext,
  ]);

  // Should be more like "amdirector"
  const isDirector = useMemo(() => {
    // HACK: Check for both exact match and :readonly suffix
    // TODO: Replace with proper permissions field when backend schema is updated
    return tournamentContext.directors.some(
      (director) =>
        director === username || director === `${username}:readonly`,
    );
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
              sendReady={() => {
                return readyForTournamentGame(
                  sendSocketMsg,
                  tournamentContext.metadata.id,
                  competitorState,
                );
              }}
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

      {/* Monitoring widget - shows status and opens modal when clicked */}
      <MonitoringWidget />

      {/* Monitoring modals */}
      <MonitoringModal
        visible={monitoringModalVisible}
        onClose={closeMonitoringModal}
      />
      <DirectorDashboardModal
        visible={directorDashboardModalVisible}
        onClose={closeDirectorDashboardModal}
      />
    </>
  );
};
