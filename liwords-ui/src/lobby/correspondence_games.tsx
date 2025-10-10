import { Table, Tooltip } from "antd";
import { FundOutlined } from "@ant-design/icons";
import React, { ReactNode, useCallback, useMemo } from "react";
import { useNavigate } from "react-router";
import { RatingBadge } from "./rating_badge";
import { challengeFormat, PlayerDisplay, SoughtGames } from "./sought_games";
import { ActiveGame, SoughtGame } from "../store/reducers/lobby_reducer";
import { VariantIcon } from "../shared/variant_icons";
import { lexiconOrder, MatchLexiconDisplay } from "../shared/lexicon_display";
import { useLoginStateStoreContext } from "../store/store";
import { normalizeVariant, VariantSectionHeader } from "./variant_utils";
import { ProfileUpdate_Rating } from "../gen/api/proto/ipc/users_pb";

type Props = {
  correspondenceGames: ActiveGame[];
  correspondenceSeeks: SoughtGame[];
  username?: string;
  userID?: string;
  newGame: (seekID: string) => void;
  ratings?: { [key: string]: ProfileUpdate_Rating };
};

export const CorrespondenceGames = (props: Props) => {
  const navigate = useNavigate();
  const {
    loginState: { userID },
  } = useLoginStateStoreContext();

  type CorrespondenceGameTableData = {
    gameID: string;
    players: ReactNode;
    turn: string;
    lexicon: ReactNode;
    lexiconCode: string;
    onTurn: boolean;
    details?: ReactNode;
    player1: string;
    player2: string;
    variant: string;
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

  const formatGameData = useCallback(
    (games: ActiveGame[]): CorrespondenceGameTableData[] => {
      const gameData: CorrespondenceGameTableData[] = games.map(
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
              ag.players[0]?.uuid === userID
                ? "Your turn"
                : ag.players[0]?.displayName || "";
          } else if (ag.playerOnTurn === 1) {
            turnIndicator =
              ag.players[1]?.uuid === userID
                ? "Your turn"
                : ag.players[1]?.displayName || "";
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
            turn: turnIndicator,
            player1: ag.players[0]?.displayName || "",
            player2: ag.players[1]?.displayName || "",
            lexicon: <MatchLexiconDisplay lexiconCode={ag.lexicon} />,
            lexiconCode: ag.lexicon,
            onTurn,
            variant: ag.variant || "classic",
            details:
              ag.tournamentID !== "" ? (
                <span className="tourney-name">{ag.tournamentID}</span>
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

      // Sort: games where it's user's turn first
      gameData.sort((a, b) => {
        if (a.onTurn && !b.onTurn) return -1;
        if (!a.onTurn && b.onTurn) return 1;
        return 0;
      });

      return gameData;
    },
    [userID],
  );

  const data = useMemo(
    () => formatGameData(props.correspondenceGames),
    [props.correspondenceGames, formatGameData],
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

  // Group data by variant
  const groupedGames = groupGamesByVariant(data);

  // Define variant order: classic, wordsmog, classic_super
  const variantOrder = ["classic", "wordsmog", "classic_super"];
  const sortedVariants = Object.keys(groupedGames).sort((a, b) => {
    const indexA = variantOrder.indexOf(a);
    const indexB = variantOrder.indexOf(b);
    if (indexA === -1) return 1;
    if (indexB === -1) return -1;
    return indexA - indexB;
  });

  return (
    <>
      {props.correspondenceSeeks.length > 0 && (
        <SoughtGames
          isMatch={true}
          userID={props.userID}
          username={props.username}
          newGame={props.newGame}
          requests={props.correspondenceSeeks}
          ratings={props.ratings}
        />
      )}
      <h4>My correspondence games</h4>

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
            onRow={(record) => ({
              onClick: (event) => {
                if (event.ctrlKey || event.altKey || event.metaKey) {
                  window.open(`/game/${encodeURIComponent(record.gameID)}`);
                } else {
                  navigate(`/game/${encodeURIComponent(record.gameID)}`);
                }
              },
              onAuxClick: (event) => {
                if (event.button === 1) {
                  // middle-click
                  window.open(`/game/${encodeURIComponent(record.gameID)}`);
                }
              },
            })}
            rowClassName={(record) => {
              const classes = ["game-listing"];
              if (record.onTurn) {
                classes.push("on-turn");
              }
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
              emptyText: "No correspondence games",
            }}
          />
        </React.Fragment>
      ))}
    </>
  );
};
