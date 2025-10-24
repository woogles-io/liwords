import { App, Button, Card, Select } from "antd";
import { Modal } from "../utils/focus_modal";
import React, {
  ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { LeftOutlined, RightOutlined } from "@ant-design/icons";
import { ActiveGames } from "../lobby/active_games";
import { SeekForm } from "../lobby/seek_form";
import { SoughtGames } from "../lobby/sought_games";
import { SoughtGame } from "../store/reducers/lobby_reducer";
import {
  useContextMatchContext,
  useLobbyStoreContext,
  useTournamentStoreContext,
} from "../store/store";
import { RecentTourneyGames } from "./recent_games";
import { ActionType } from "../actions/actions";
import { Pairings } from "./pairings";
import { isPairedMode, isClubType } from "../store/constants";
import { Standings } from "./standings";
import { DirectorTools } from "./director_tools/director_tools";
import { useClient } from "../utils/hooks/connect";
import { TournamentService } from "../gen/api/proto/tournament_service/tournament_service_pb";
import { flashTournamentError } from "./tournament_error";
import { useTournamentCompetitorState } from "../hooks/use_tournament_competitor_state";
import { PairingMethod } from "../gen/api/proto/ipc/tournament_pb";
// import { CheckIn } from './check_in';

const PAGE_SIZE = 30;

type Props = {
  newGame: (seekID: string) => void;
  loggedIn: boolean;
  // For logged-in users:
  userID?: string;
  username?: string;
  selectedGameTab: string;
  setSelectedGameTab: (tab: string) => void;
  tournamentID: string;
  isDirector: boolean;
  canManageTournaments: boolean;
  onSeekSubmit: (g: SoughtGame) => void;
  sendReady: () => void;
  showFirst?: boolean;
};

export const ActionsPanel = React.memo((props: Props) => {
  const [matchModalVisible, setMatchModalVisible] = useState(false);
  const [formDisabled, setFormDisabled] = useState(false);
  const {
    selectedGameTab,
    setSelectedGameTab,
    isDirector,
    canManageTournaments,
    onSeekSubmit,
    newGame,
    userID,
    username,
  } = props;
  const renderDirectorTools = () => {
    return <DirectorTools tournamentID={props.tournamentID} />;
  };
  const { dispatchTournamentContext, tournamentContext } =
    useTournamentStoreContext();

  // HACK: Check if user is a full director (not read-only)
  // TODO: Replace with proper permissions field when backend schema is updated
  const isFullDirector = useMemo(() => {
    if (!isDirector || !username) return false;
    // Check if user is in directors list WITHOUT :readonly suffix
    return tournamentContext.directors.some(
      (director) => director === username,
    );
  }, [isDirector, username, tournamentContext.directors]);
  const itIsPairedMode = useMemo(
    () => isPairedMode(tournamentContext.metadata?.type),
    [tournamentContext],
  );
  const { divisions } = tournamentContext;
  const competitorState = useTournamentCompetitorState();
  const [competitorStatusLoaded, setCompetitorStatusLoaded] = useState(
    competitorState.isRegistered,
  );
  let initialRound = 0;
  let initialDivision = "";
  if (competitorState.division) {
    initialDivision = competitorState.division;
    initialRound = competitorState.currentRound;
  }
  const [selectedRound, setSelectedRound] = useState(initialRound);
  const [selectedDivision, setSelectedDivision] = useState(initialDivision);
  const { lobbyContext } = useLobbyStoreContext();
  const tournamentID = tournamentContext.metadata?.id;

  const { addHandleContextMatch, removeHandleContextMatch } =
    useContextMatchContext();
  const friendRef = useRef("");
  const handleContextMatch = useCallback((s: string) => {
    friendRef.current = s;
    setMatchModalVisible(true);
  }, []);
  useEffect(() => {
    if (!itIsPairedMode && !matchModalVisible) {
      addHandleContextMatch(handleContextMatch);
      return () => {
        removeHandleContextMatch(handleContextMatch);
      };
    }
  }, [
    itIsPairedMode,
    matchModalVisible,
    handleContextMatch,
    addHandleContextMatch,
    removeHandleContextMatch,
  ]);

  const lobbyContextMatchRequests = lobbyContext?.matchRequests;
  const thisTournamentMatchRequests = useMemo(
    () =>
      lobbyContextMatchRequests?.filter(
        (matchRequest) => matchRequest.tournamentID === tournamentID,
      ),
    [lobbyContextMatchRequests, tournamentID],
  );

  const fetchPrev = useCallback(() => {
    dispatchTournamentContext({
      actionType: ActionType.SetTourneyGamesOffset,
      payload: Math.max(
        tournamentContext.gamesOffset - tournamentContext.gamesPageSize,
        0,
      ),
    });
  }, [
    dispatchTournamentContext,
    tournamentContext.gamesOffset,
    tournamentContext.gamesPageSize,
  ]);
  const fetchNext = useCallback(() => {
    dispatchTournamentContext({
      actionType: ActionType.SetTourneyGamesOffset,
      payload: tournamentContext.gamesOffset + tournamentContext.gamesPageSize,
    });
  }, [
    dispatchTournamentContext,
    tournamentContext.gamesOffset,
    tournamentContext.gamesPageSize,
  ]);
  const onFormSubmit = (sg: SoughtGame) => {
    setMatchModalVisible(false);
    setFormDisabled(true);
    if (!formDisabled) {
      onSeekSubmit(sg);
      setTimeout(() => {
        setFormDisabled(false);
      }, 500);
    }
  };

  const tournamentClient = useClient(TournamentService);

  const { message, notification } = App.useApp();

  useEffect(() => {
    if (!tournamentID) {
      return;
    }

    (async () => {
      const resp = await tournamentClient.recentGames({
        id: tournamentID,
        numGames: PAGE_SIZE,
        offset: tournamentContext.gamesOffset,
      });
      dispatchTournamentContext({
        actionType: ActionType.AddTourneyGameResults,
        payload: resp.games,
      });
    })();
  }, [
    tournamentID,
    tournamentClient,
    dispatchTournamentContext,
    tournamentContext.gamesOffset,
  ]);
  const renderDivisionSelector =
    Object.values(divisions).length > 1 ? (
      <Select value={selectedDivision} onChange={setSelectedDivision}>
        {Object.values(divisions).map((d) => {
          return (
            <Select.Option value={d.divisionID} key={d.divisionID}>
              {d.divisionID}
            </Select.Option>
          );
        })}
      </Select>
    ) : null;
  const renderStartRoundButton = () => {
    const division = tournamentContext.divisions[selectedDivision];
    if (!division) {
      return null;
    }
    const { currentRound } = division;
    let roundToStart: null | number = null;
    if (division) {
      roundToStart = currentRound + 1;
    }
    if (
      !isDirector ||
      !(typeof roundToStart === "number") ||
      !(roundToStart === selectedRound) ||
      roundToStart === 0 // (handle this with Start Tournament)
    ) {
      return null;
    }
    const startRound = async () => {
      try {
        await tournamentClient.startRoundCountdown({
          id: tournamentID,
          division: division.divisionID,
          round: roundToStart as number, // should already be a number.
        });
      } catch (e) {
        flashTournamentError(message, notification, e, tournamentContext);
      }
    };
    return (
      <Button className="primary open-round" onClick={startRound}>
        Open Round {roundToStart + 1}
      </Button>
    );
  };

  const pairingsTentative = useMemo(() => {
    // Only IRL mode can have tentative pairings.
    if (!tournamentContext.metadata.irlMode) {
      return false;
    }
    if (!tournamentContext.divisions[selectedDivision]) {
      return false;
    }
    if (!tournamentContext.divisions[selectedDivision].roundControls) {
      return false;
    }
    if (
      !tournamentContext.divisions[selectedDivision].roundControls[
        selectedRound
      ]
    ) {
      return false;
    }
    const pairingMethod =
      tournamentContext.divisions[selectedDivision].roundControls[selectedRound]
        .pairingMethod;
    // These pairing methods could potentially be manually edited by the director.
    // Actually, they all could be, but these are the ones that are most likely to be.
    // We don't put "MANUAL" here because that is presumed to be actually chosen
    // by the director.
    const potentiallyTentative = [
      PairingMethod.KING_OF_THE_HILL,
      PairingMethod.SWISS,
      PairingMethod.FACTOR,
    ].includes(pairingMethod);

    if (selectedRound > -1 && tournamentContext.divisions[selectedDivision]) {
      return (
        selectedRound >
          tournamentContext.divisions[selectedDivision].currentRound &&
        potentiallyTentative
      );
    }
    return false;
  }, [
    selectedRound,
    selectedDivision,
    tournamentContext.divisions,
    tournamentContext.metadata.irlMode,
  ]);

  const renderGamesTab = () => {
    if (selectedGameTab === "GAMES") {
      if (itIsPairedMode) {
        return (
          <div className="pairings-container">
            {/* <CheckIn /> */}
            <div className="round-options">
              {renderDivisionSelector}
              {renderStartRoundButton()}
            </div>
            <Pairings
              selectedRound={selectedRound}
              selectedDivision={selectedDivision}
              username={username}
              sendReady={props.sendReady}
              isDirector={isDirector}
              isFullDirector={isFullDirector}
              showFirst={props.showFirst}
              tentative={pairingsTentative}
            />
          </div>
        );
      }
      return (
        <>
          {thisTournamentMatchRequests?.length ? (
            <SoughtGames
              isMatch={true}
              isClubMatch={isClubType(tournamentContext.metadata?.type)}
              userID={userID}
              username={username}
              newGame={newGame}
              requests={thisTournamentMatchRequests}
            />
          ) : null}
          <ActiveGames
            username={username}
            activeGames={tournamentContext?.activeGames}
          />
        </>
      );
    }
    if (selectedGameTab === "RECENT") {
      return (
        <>
          <h4>Recent games</h4>
          <RecentTourneyGames
            games={tournamentContext.finishedTourneyGames}
            fetchPrev={
              tournamentContext.gamesOffset > 0 ? fetchPrev : undefined
            }
            fetchNext={
              tournamentContext.finishedTourneyGames.length < PAGE_SIZE
                ? undefined
                : fetchNext
            }
          />
        </>
      );
    }
    if (selectedGameTab === "STANDINGS") {
      return (
        <div className="standings-container">
          <div className="round-options">{renderDivisionSelector}</div>
          <Standings selectedDivision={selectedDivision} />
        </div>
      );
    }
    return null;
  };

  const matchModal = (
    <Modal
      className="seek-modal"
      title="Send match request"
      open={matchModalVisible}
      destroyOnClose
      onCancel={() => {
        setMatchModalVisible(false);
        setFormDisabled(false);
      }}
      footer={[
        <Button
          key="back"
          onClick={() => {
            setMatchModalVisible(false);
          }}
          style={{ marginBottom: 5 }}
          type="link"
        >
          Cancel
        </Button>,
        <button
          className="primary"
          key="submit"
          form="match-seek"
          type="submit"
          disabled={formDisabled}
        >
          Create game
        </button>,
      ]}
    >
      <SeekForm
        onFormSubmit={onFormSubmit}
        loggedIn={props.loggedIn}
        username={props.username}
        showFriendInput={true}
        friendRef={friendRef}
        id="match-seek"
        tournamentID={props.tournamentID}
      />
    </Modal>
  );

  const idFromPlayerEntry = useCallback((p: string) => p.split(":")[0], []);
  useEffect(() => {
    const divisionArray = Object.values(divisions);
    const foundDivision = userID
      ? divisionArray.find((d) => {
          return d.players
            .map((v) => v.id)
            .map(idFromPlayerEntry)
            .includes(userID);
        })
      : undefined;
    // look for ourselves in the division.
    if (foundDivision) {
      if (!competitorStatusLoaded) {
        setCompetitorStatusLoaded(true);
        setSelectedDivision(foundDivision.divisionID);
        setSelectedRound(foundDivision.currentRound);
      } else if (!selectedDivision) {
        setSelectedDivision(foundDivision.divisionID);
        setSelectedRound(foundDivision.currentRound);
      } else if (selectedRound === -1) {
        // If we are directing _and_ playing, this, combined with the code
        // in pairings.tsx to hide initial pairings, will show preview
        // pairings for the director.
        setSelectedRound(isDirector ? 0 : foundDivision.currentRound);
      }
    } else {
      // we are an observer
      if (divisionArray.length) {
        if (!selectedDivision) {
          setSelectedDivision(divisionArray[0].divisionID);
          setSelectedRound(
            divisionArray[0].currentRound > -1
              ? divisionArray[0].currentRound
              : 0,
          );
        }
      }
    }
  }, [
    divisions,
    idFromPlayerEntry,
    isDirector,
    selectedDivision,
    competitorStatusLoaded,
    selectedRound,
    userID,
  ]);

  const actions = useMemo(() => {
    if (selectedGameTab === "STANDINGS") {
      return [];
    }
    let matchButtonText = "Start tournament game";
    if (isClubType(tournamentContext.metadata?.type)) {
      matchButtonText = "Start club game";
    }
    const availableActions = new Array<ReactNode>();
    if (props.loggedIn && !itIsPairedMode) {
      // We are allowing free-form match requests in CLUBHOUSE mode, if desired.
      availableActions.push(
        <div
          className="match"
          onClick={() => {
            setMatchModalVisible(true);
          }}
          key="match-action"
        >
          {matchButtonText}
        </div>,
      );
    }
    if (
      itIsPairedMode &&
      selectedRound > -1 &&
      tournamentContext.divisions[selectedDivision]
    ) {
      const lastRound =
        tournamentContext.divisions[selectedDivision]?.numRounds - 1 ||
        selectedRound;
      if (selectedRound < 1) {
        availableActions.push(<div className="empty"></div>);
      } else {
        availableActions.push(
          <div
            className="round-change prev"
            onClick={() => {
              setSelectedRound(selectedRound - 1);
            }}
          >
            <LeftOutlined />
            {/* The previous, zero-indexed round converted to 1-indexed */}
            Round {selectedRound}{" "}
          </div>,
        );
      }
      availableActions.push(
        <div className="round-label">
          {/* The current zero-indexed round converted to 1-indexed */}
          Round {selectedRound + 1}
        </div>,
      );
      if (lastRound === selectedRound) {
        availableActions.push(<div className="empty"></div>);
      } else {
        availableActions.push(
          <div
            className="round-change next"
            onClick={() => {
              setSelectedRound(selectedRound + 1);
            }}
          >
            {/* The next zero-indexed round converted to 1-indexed*/}
            Round {selectedRound + 2}
            <RightOutlined />
          </div>,
        );
      }
    }
    return availableActions;
  }, [
    itIsPairedMode,
    props.loggedIn,
    tournamentContext,
    selectedDivision,
    selectedRound,
    selectedGameTab,
  ]);
  return (
    <div className="game-lists">
      <Card
        actions={actions}
        className={itIsPairedMode ? "paired-mode" : "free-form"}
      >
        <div className="tabs">
          <div
            onClick={() => {
              setSelectedGameTab("GAMES");
            }}
            className={selectedGameTab === "GAMES" ? "tab active" : "tab"}
          >
            Games
          </div>
          {!itIsPairedMode ? (
            <div
              onClick={() => {
                setSelectedGameTab("RECENT");
              }}
              className={selectedGameTab === "RECENT" ? "tab active" : "tab"}
            >
              Recent games
            </div>
          ) : (
            <div
              onClick={() => {
                setSelectedGameTab("STANDINGS");
              }}
              className={selectedGameTab === "STANDINGS" ? "tab active" : "tab"}
            >
              Standings
            </div>
          )}
          {(isDirector || canManageTournaments) && (
            <div
              onClick={() => {
                setSelectedGameTab("DIRECTOR TOOLS");
              }}
              className={
                selectedGameTab === "DIRECTOR TOOLS" ? "tab active" : "tab"
              }
            >
              Director Tools
            </div>
          )}
        </div>
        <div className="main-content">
          {(isDirector || canManageTournaments) &&
            selectedGameTab === "DIRECTOR TOOLS" &&
            renderDirectorTools()}
          {matchModal}
          {!competitorState.isRegistered &&
          tournamentContext.metadata.registrationOpen &&
          !tournamentContext.started ? (
            <>
              <center>
                <div style={{ marginTop: 80 }}>
                  Registration for {tournamentContext.metadata.name} is now
                  open.
                </div>
              </center>
              <center>
                {renderDivisionSelector ? (
                  <div style={{ marginTop: 20 }}>
                    Select division: {renderDivisionSelector}
                  </div>
                ) : null}

                <Button
                  size="large"
                  type="primary"
                  style={{ marginTop: 40 }}
                  onClick={async () => {
                    try {
                      await tournamentClient.register({
                        id: tournamentContext.metadata.id,
                        division: selectedDivision,
                      });
                    } catch (e) {
                      flashTournamentError(
                        message,
                        notification,
                        e,
                        tournamentContext,
                      );
                    }
                  }}
                >
                  Register for this tournament
                </Button>
              </center>
            </>
          ) : competitorState.isRegistered &&
            tournamentContext.metadata.checkinsOpen &&
            !competitorState.isCheckedIn &&
            !tournamentContext.started ? (
            <>
              <center>
                <div style={{ marginTop: 80, marginLeft: 20, marginRight: 20 }}>
                  Please ensure you can play the tournament in its entirety
                  before checking in.
                </div>
              </center>
              <center>
                <Button
                  size="large"
                  style={{ marginTop: 40 }}
                  onClick={async () => {
                    try {
                      await tournamentClient.checkIn({
                        id: tournamentContext.metadata.id,
                        checkin: true,
                      });
                    } catch (e) {
                      flashTournamentError(
                        message,
                        notification,
                        e,
                        tournamentContext,
                      );
                    }
                  }}
                >
                  Check into this tournament (division&nbsp;
                  {competitorState.division})
                </Button>
              </center>
            </>
          ) : (
            renderGamesTab()
          )}
        </div>
      </Card>
    </div>
  );
});
