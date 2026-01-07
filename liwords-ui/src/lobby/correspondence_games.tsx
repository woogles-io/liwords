import { Table, Tag, Tooltip } from "antd";
import {
  FundOutlined,
  ClockCircleOutlined,
  TrophyOutlined,
} from "@ant-design/icons";
import React, {
  ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useNavigate } from "react-router";
import { RatingBadge } from "./rating_badge";
import { challengeFormat, PlayerDisplay, SoughtGames } from "./sought_games";
import { ActiveGame, SoughtGame } from "../store/reducers/lobby_reducer";
import { VariantIcon } from "../shared/variant_icons";
import { lexiconOrder, MatchLexiconDisplay } from "../shared/lexicon_display";
import { useLoginStateStoreContext } from "../store/store";
import { normalizeVariant, VariantSectionHeader } from "./variant_utils";
import { ProfileUpdate_Rating } from "../gen/api/proto/ipc/users_pb";
import {
  GameEndReason,
  GameInfoResponse,
} from "../gen/api/proto/ipc/omgwords_pb";
import { useClient } from "../utils/hooks/connect";
import { GameMetadataService } from "../gen/api/proto/game_service/game_service_pb";

type Props = {
  correspondenceGames: ActiveGame[];
  correspondenceSeeks: SoughtGame[];
  username?: string;
  userID?: string;
  newGame: (seekID: string) => void;
  declineGame?: (seekID: string) => void;
  ratings?: { [key: string]: ProfileUpdate_Rating };
};

export const CorrespondenceGames = (props: Props) => {
  const navigate = useNavigate();
  const {
    loginState: { userID },
  } = useLoginStateStoreContext();
  const gameMetadataClient = useClient(GameMetadataService);
  const [recentGames, setRecentGames] = useState<GameInfoResponse[]>([]);

  // Fetch recent correspondence games
  useEffect(() => {
    if (!props.username) return;

    const fetchRecentGames = async () => {
      try {
        const resp = await gameMetadataClient.getRecentCorrespondenceGames({
          username: props.username,
          numGames: 20,
        });
        setRecentGames(resp.gameInfo);
      } catch (e) {
        console.error("Failed to fetch recent correspondence games:", e);
      }
    };

    fetchRecentGames();
  }, [props.username, gameMetadataClient]);

  type CorrespondenceGameTableData = {
    gameID: string;
    players: ReactNode;
    turn: ReactNode;
    lexicon: ReactNode;
    lexiconCode: string;
    onTurn: boolean;
    details?: ReactNode;
    player1: string;
    player2: string;
    variant: string;
    timeRemaining: number; // Time remaining in seconds, or Infinity if not applicable
    finalScore?: ReactNode; // For finished games
    endReason?: ReactNode; // For finished games
    isLeagueGame?: boolean; // Whether this is a league game
    outcome?: ReactNode; // Combined outcome column for recently ended games
  };

  // Group games by variant
  const groupGamesByVariant = (
    games: CorrespondenceGameTableData[],
  ): { [variant: string]: CorrespondenceGameTableData[] } => {
    const grouped: { [variant: string]: CorrespondenceGameTableData[] } = {};
    games.forEach((game) => {
      const variant = normalizeVariant(game.variant);
      if (!grouped[variant]) {
        grouped[variant] = [];
      }
      grouped[variant].push(game);
    });
    return grouped;
  };

  // Separate league games from regular games
  const separateLeagueGames = (
    games: CorrespondenceGameTableData[],
  ): {
    leagueGames: CorrespondenceGameTableData[];
    regularGames: CorrespondenceGameTableData[];
  } => {
    const leagueGames: CorrespondenceGameTableData[] = [];
    const regularGames: CorrespondenceGameTableData[] = [];
    games.forEach((game) => {
      if (game.isLeagueGame) {
        leagueGames.push(game);
      } else {
        regularGames.push(game);
      }
    });
    return { leagueGames, regularGames };
  };

  const formatGameData = useCallback(
    (games: ActiveGame[]): CorrespondenceGameTableData[] => {
      const userGames = games.filter((ag: ActiveGame) =>
        ag.players.some((player) => player.uuid === userID),
      );
      const gameData: CorrespondenceGameTableData[] = userGames.map(
        (ag: ActiveGame) => {
          const player1rating = ag.players[0]?.rating || "1500?";
          const player2rating = ag.players[1]?.rating || "1500?";

          // Check if it's user's turn by comparing playerOnTurn with player UUID
          let onTurn = false;
          if (ag.playerOnTurn !== undefined && userID) {
            const playerIndex = ag.players.findIndex((p) => p.uuid === userID);
            onTurn = playerIndex === ag.playerOnTurn;
          }

          // Determine turn indicator text
          let turnIndicator = "";
          if (ag.playerOnTurn === 0) {
            turnIndicator =
              ag.players[0]?.uuid === userID ? "Your turn" : "Opponent";
          } else if (ag.playerOnTurn === 1) {
            turnIndicator =
              ag.players[1]?.uuid === userID ? "Your turn" : "Opponent";
          }

          // Calculate time remaining for low time warning (< 24 hours)
          const now = Date.now();
          let timeRemainingSecs = Infinity;
          let isLowTime = false;
          if (onTurn && ag.lastUpdate && ag.incrementSecs) {
            const timeElapsedSecs = (now - ag.lastUpdate) / 1000;
            timeRemainingSecs = ag.incrementSecs - timeElapsedSecs;
            isLowTime = timeRemainingSecs < 86400; // 24 hours in seconds
          }

          // Add low time indicator to turn text
          let turnDisplay: ReactNode = turnIndicator;
          if (isLowTime) {
            turnDisplay = (
              <span style={{ color: "#ff4d4f" }}>
                <Tooltip title="Less than 24 hours remaining">
                  <ClockCircleOutlined style={{ marginRight: 4 }} />
                </Tooltip>
                {turnIndicator}
              </span>
            );
          }

          return {
            gameID: ag.gameID,
            players: (
              <>
                <div>
                  <PlayerDisplay
                    username={ag.players[0]?.displayName || ""}
                    userID={ag.players[0]?.uuid}
                  />
                  <RatingBadge rating={player1rating} />
                </div>
                <div>
                  <PlayerDisplay
                    username={ag.players[1]?.displayName || ""}
                    userID={ag.players[1]?.uuid}
                  />
                  <RatingBadge rating={player2rating} />
                </div>
              </>
            ),
            turn: turnDisplay,
            player1: ag.players[0]?.displayName || "",
            player2: ag.players[1]?.displayName || "",
            lexicon: <MatchLexiconDisplay lexiconCode={ag.lexicon} />,
            lexiconCode: ag.lexicon,
            onTurn,
            variant: ag.variant || "classic",
            timeRemaining: timeRemainingSecs,
            isLeagueGame: !!ag.leagueSlug,
            details:
              ag.tournamentID !== "" ? (
                <span className="tourney-name">{ag.tournamentID}</span>
              ) : ag.leagueSlug ? (
                <span className="league-game">
                  <Tooltip title={`League Game: ${ag.leagueSlug}`}>
                    <TrophyOutlined
                      style={{ color: "#faad14", marginRight: 4 }}
                    />
                  </Tooltip>
                  League
                </span>
              ) : (
                <>
                  <VariantIcon vcode={ag.variant} />{" "}
                  {challengeFormat(ag.challengeRule)}
                  {ag.rated ? (
                    <Tooltip title="Rated">
                      <FundOutlined />
                    </Tooltip>
                  ) : null}
                </>
              ),
          };
        },
      );

      // Sort: games where it's user's turn first, then by time remaining (least to most)
      gameData.sort((a, b) => {
        // First prioritize games where it's user's turn
        if (a.onTurn && !b.onTurn) return -1;
        if (!a.onTurn && b.onTurn) return 1;
        // Within each group (user's turn vs not), sort by time remaining (ascending)
        return a.timeRemaining - b.timeRemaining;
      });

      return gameData;
    },
    [userID],
  );

  // Format recent (finished) games
  const formatRecentGameData = useCallback(
    (games: GameInfoResponse[]): CorrespondenceGameTableData[] => {
      return games.map((g: GameInfoResponse) => {
        const player1 = g.players[0];
        const player2 = g.players[1];

        // Show final scores
        const score1 = g.scores[0] ?? 0;
        const score2 = g.scores[1] ?? 0;

        // Determine user's result and opponent
        const userPlayerIndex = g.players.findIndex((p) => p.userId === userID);
        const opponentIndex = userPlayerIndex === 0 ? 1 : 0;
        const opponent = g.players[opponentIndex];
        const opponentName = opponent?.nickname || "Unknown";
        const userScore = userPlayerIndex === 0 ? score1 : score2;
        const opponentScore = userPlayerIndex === 0 ? score2 : score1;

        // Determine result badge and text
        let resultBadge: ReactNode = null;
        let resultText = "";
        const userWon = g.winner === userPlayerIndex;
        const isTie = g.winner === -1;
        const userLost = !userWon && !isTie;

        if (userWon) {
          resultBadge = <Tag color="green">Won</Tag>;
        } else if (isTie) {
          resultBadge = <Tag>Tie</Tag>;
        } else {
          resultBadge = <Tag color="red">Lost</Tag>;
        }

        // Build outcome text based on end reason
        // Format: "[Won] over yaki 235-200" or "[Lost] to foo via resignation"
        switch (g.gameEndReason) {
          case GameEndReason.TIME:
            if (userWon) {
              resultText = `${opponentName} timed out`;
            } else {
              resultText = `to ${opponentName} on timeout`;
            }
            break;
          case GameEndReason.RESIGNED:
            if (userWon) {
              resultText = `${opponentName} resigned`;
            } else {
              resultText = `to ${opponentName} via resignation`;
            }
            break;
          case GameEndReason.FORCE_FORFEIT:
            if (userWon) {
              resultText = `${opponentName} forfeited`;
            } else {
              resultText = `to ${opponentName} via forfeit`;
            }
            break;
          default:
            // Standard completion - show scores
            if (userWon) {
              resultText = `over ${opponentName} ${userScore}–${opponentScore}`;
            } else if (isTie) {
              resultText = `with ${opponentName} ${userScore}–${opponentScore}`;
            } else {
              resultText = `to ${opponentName} ${userScore}–${opponentScore}`;
            }
        }

        const outcome = (
          <span>
            {resultBadge} {resultText}
          </span>
        );

        return {
          gameID: g.gameId,
          players: (
            <>
              <div>
                <PlayerDisplay
                  username={player1?.nickname || ""}
                  userID={player1?.userId}
                />
              </div>
              <div>
                <PlayerDisplay
                  username={player2?.nickname || ""}
                  userID={player2?.userId}
                />
              </div>
            </>
          ),
          turn: outcome, // Keep for compatibility
          outcome,
          lexicon: (
            <MatchLexiconDisplay lexiconCode={g.gameRequest?.lexicon || ""} />
          ),
          lexiconCode: g.gameRequest?.lexicon || "",
          onTurn: false, // Finished games aren't anyone's turn
          isLeagueGame: !!g.leagueSlug,
          details: g.tournamentId ? (
            <span className="tourney-name">{g.tournamentId}</span>
          ) : g.leagueSlug ? (
            <span className="league-game">
              <Tooltip title={`League Game: ${g.leagueSlug}`}>
                <TrophyOutlined style={{ color: "#faad14", marginRight: 4 }} />
              </Tooltip>
              League
            </span>
          ) : (
            <>
              <VariantIcon vcode={g.gameRequest?.rules?.variantName || ""} />{" "}
              {challengeFormat(g.gameRequest?.challengeRule || 0)}
              {g.gameRequest?.ratingMode === 0 ? (
                <Tooltip title="Rated">
                  <FundOutlined />
                </Tooltip>
              ) : null}
            </>
          ),
          player1: player1?.nickname || "",
          player2: player2?.nickname || "",
          variant: g.gameRequest?.rules?.variantName || "",
          timeRemaining: Infinity, // Not relevant for finished games
        };
      });
    },
    [userID],
  );

  const data = useMemo(
    () => formatGameData(props.correspondenceGames),
    [props.correspondenceGames, formatGameData],
  );

  const recentData = useMemo(
    () => formatRecentGameData(recentGames),
    [recentGames, formatRecentGameData],
  );

  const handleRowClick = (record: CorrespondenceGameTableData) => {
    navigate(`/game/${encodeURIComponent(record.gameID)}`);
  };

  const columns = [
    {
      title: "Players",
      className: "players",
      dataIndex: "players",
      key: "players",
    },
    {
      title: "Turn",
      className: "turn",
      dataIndex: "turn",
      key: "turn",
    },
    {
      title: "Words",
      className: "lexicon",
      dataIndex: "lexicon",
      key: "lexicon",
      filters: lexiconOrder.map((l) => ({
        text: <MatchLexiconDisplay lexiconCode={l} />,
        value: l,
      })),
      filterMultiple: true,
      onFilter: (
        value: React.Key | boolean,
        record: CorrespondenceGameTableData,
      ) => typeof value === "string" && record.lexiconCode === value,
    },
    {
      title: "Details",
      className: "details",
      dataIndex: "details",
      key: "details",
    },
  ];

  const finishedGameColumns = [
    {
      title: "Outcome",
      className: "outcome",
      dataIndex: "outcome",
      key: "outcome",
    },
    {
      title: "Words",
      className: "lexicon",
      dataIndex: "lexicon",
      key: "lexicon",
      filters: lexiconOrder.map((l) => ({
        text: <MatchLexiconDisplay lexiconCode={l} />,
        value: l,
      })),
      filterMultiple: true,
      onFilter: (
        value: React.Key | boolean,
        record: CorrespondenceGameTableData,
      ) => typeof value === "string" && record.lexiconCode === value,
    },
    {
      title: "Details",
      className: "details",
      dataIndex: "details",
      key: "details",
    },
  ];

  // Separate league games from regular games
  const { leagueGames, regularGames } = separateLeagueGames(data);

  // Group regular games by variant
  const groupedGames = groupGamesByVariant(regularGames);

  // Define variant order: classic, wordsmog, classic_super
  const variantOrder = ["classic", "wordsmog", "classic_super"];
  const sortedVariants = Object.keys(groupedGames).sort((a, b) => {
    const indexA = variantOrder.indexOf(a);
    const indexB = variantOrder.indexOf(b);
    if (indexA === -1) return 1;
    if (indexB === -1) return -1;
    return indexA - indexB;
  });

  // Common row handlers for game tables
  const getRowHandlers = (record: CorrespondenceGameTableData) => ({
    onClick: (event: React.MouseEvent) => {
      if (event.ctrlKey || event.altKey || event.metaKey) {
        window.open(`/game/${encodeURIComponent(record.gameID)}`);
      } else {
        navigate(`/game/${encodeURIComponent(record.gameID)}`);
      }
    },
    onAuxClick: (event: React.MouseEvent) => {
      if (event.button === 1) {
        // middle-click
        window.open(`/game/${encodeURIComponent(record.gameID)}`);
      }
    },
  });

  const getRowClassName = (record: CorrespondenceGameTableData) => {
    const classes = ["game-listing"];
    if (record.onTurn) {
      classes.push("on-turn");
    }
    if (
      props.username &&
      (record.player1 === props.username || record.player2 === props.username)
    ) {
      classes.push("my-game");
    }
    return classes.join(" ");
  };

  return (
    <>
      {props.correspondenceSeeks.length > 0 && (
        <SoughtGames
          isMatch={true}
          userID={props.userID}
          username={props.username}
          newGame={props.newGame}
          declineGame={props.declineGame}
          requests={props.correspondenceSeeks}
          ratings={props.ratings}
        />
      )}
      <h4>My correspondence games</h4>

      {/* League games section - shown first */}
      {leagueGames.length > 0 && (
        <>
          <h4>
            <TrophyOutlined style={{ color: "#faad14", marginRight: 8 }} />
            League Games
          </h4>
          <Table
            className="games observe correspondence-games league-games"
            dataSource={leagueGames}
            columns={columns}
            pagination={false}
            rowKey="gameID"
            showSorterTooltip={false}
            onRow={getRowHandlers}
            rowClassName={getRowClassName}
            locale={{
              emptyText: "No league games",
            }}
          />
        </>
      )}

      {/* Regular games by variant */}
      {sortedVariants.map((variant) => (
        <React.Fragment key={variant}>
          <VariantSectionHeader variant={variant} />
          <Table
            className="games observe correspondence-games"
            dataSource={groupedGames[variant]}
            columns={columns}
            pagination={false}
            rowKey="gameID"
            showSorterTooltip={false}
            onRow={getRowHandlers}
            rowClassName={getRowClassName}
            locale={{
              emptyText: "No correspondence games",
            }}
          />
        </React.Fragment>
      ))}

      {recentData.length > 0 && (
        <>
          <h4 style={{ marginTop: "32px", opacity: 0.8 }}>
            Recently ended games
          </h4>
          <Table
            className="games observe correspondence-games finished-games"
            dataSource={recentData}
            columns={finishedGameColumns}
            pagination={false}
            rowKey="gameID"
            showSorterTooltip={false}
            onRow={getRowHandlers}
            rowClassName={(record) => {
              const classes = ["game-listing", "finished-game"];
              if (
                props.username &&
                (record.player1 === props.username ||
                  record.player2 === props.username)
              ) {
                classes.push("my-game");
              }
              return classes.join(" ");
            }}
            locale={{
              emptyText: "No recently ended correspondence games",
            }}
          />
        </>
      )}
    </>
  );
};
