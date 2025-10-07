import { Table, Tooltip } from "antd";
import {
  FilterValue,
  SorterResult,
  TableCurrentDataSource,
  TablePaginationConfig,
} from "antd/lib/table/interface";
import React, { ReactNode, useCallback, useMemo } from "react";
import { useNavigate } from "react-router";
import { FundOutlined } from "@ant-design/icons";
import { RatingBadge } from "./rating_badge";
import { challengeFormat, PlayerDisplay, timeFormat } from "./sought_games";
import { ActiveGame } from "../store/reducers/lobby_reducer";
import { calculateTotalTime } from "../store/constants";
import { VariantIcon } from "../shared/variant_icons";
import { lexiconOrder, MatchLexiconDisplay } from "../shared/lexicon_display";
import {
  useLoginStateStoreContext,
  useLobbyStoreContext,
} from "../store/store";
import { ActionType } from "../actions/actions";
import { normalizeVariant, VariantSectionHeader } from "./variant_utils";

type Props = {
  activeGames: ActiveGame[];
  username?: string;
  type?: "RESUME";
};

export const ActiveGames = (props: Props) => {
  const {
    lobbyContext: { lobbyFilterByLexicon },
    dispatchLobbyContext,
  } = useLobbyStoreContext();
  const lobbyFilterByLexiconArray = useMemo(
    () => lobbyFilterByLexicon?.match(/\S+/g) ?? [],
    [lobbyFilterByLexicon],
  );
  const navigate = useNavigate();
  const {
    loginState: { perms },
  } = useLoginStateStoreContext();
  const isAdmin = perms.includes("adm");

  const handleChange = useCallback(
    (
      _pagination: TablePaginationConfig,
      filters: Record<string, FilterValue | null>,
      _sorter:
        | SorterResult<ActiveGameTableData>
        | SorterResult<ActiveGameTableData>[],
      extra: TableCurrentDataSource<ActiveGameTableData>,
    ) => {
      if (extra.action === "filter") {
        if (filters.lexicon && filters.lexicon.length > 0) {
          const lexicon = filters.lexicon.join(" ");
          if (lexicon !== lobbyFilterByLexicon) {
            localStorage.setItem("lobbyFilterByLexicon", lexicon);
            dispatchLobbyContext({
              actionType: ActionType.setLobbyFilterByLexicon,
              payload: lexicon,
            });
          }
        } else {
          // filter is reset, remove lexicon
          localStorage.removeItem("lobbyFilterByLexicon");
          dispatchLobbyContext({
            actionType: ActionType.setLobbyFilterByLexicon,
            payload: null,
          });
        }
      }
    },
    [lobbyFilterByLexicon, dispatchLobbyContext],
  );

  type ActiveGameTableData = {
    gameID: string;
    players: ReactNode;
    lexicon: ReactNode;
    lexiconCode: string;
    time: string;
    totalTime: number;
    details?: ReactNode;
    player1: string;
    player2: string;
    variant: string;
  };
  // Group games by variant
  const groupGamesByVariant = (
    games: ActiveGameTableData[],
  ): { [variant: string]: ActiveGameTableData[] } => {
    const grouped: { [variant: string]: ActiveGameTableData[] } = {};
    games.forEach((game) => {
      const variant = normalizeVariant(game.variant);
      if (!grouped[variant]) {
        grouped[variant] = [];
      }
      grouped[variant].push(game);
    });
    return grouped;
  };

  const formatGameData = (games: ActiveGame[]): ActiveGameTableData[] => {
    const gameData: ActiveGameTableData[] = games
      .sort((agA: ActiveGame, agB: ActiveGame) => {
        // Default sort should be by combined rating
        const parseRating = (rating: string) => {
          return rating.endsWith("?")
            ? parseInt(rating.substring(0, rating.length - 1), 10)
            : parseInt(rating, 10);
        };
        return (
          parseRating(agB.players[0].rating) +
          parseRating(agB.players[1].rating) -
          (parseRating(agA.players[0].rating) +
            parseRating(agA.players[1].rating))
        );
      })
      .map((ag: ActiveGame): ActiveGameTableData => {
        const getDetails = () => {
          return (
            <>
              <VariantIcon vcode={ag.variant} />{" "}
              {challengeFormat(ag.challengeRule)}
              {ag.rated ? (
                <Tooltip title="Rated">
                  <FundOutlined />
                </Tooltip>
              ) : null}
            </>
          );
        };
        return {
          gameID: ag.gameID,
          players: (
            <>
              <div>
                <PlayerDisplay
                  username={ag.players[0].displayName}
                  userID={ag.players[0].uuid}
                />
                <RatingBadge rating={ag.players[0].rating} />
              </div>
              <div>
                <PlayerDisplay
                  username={ag.players[1].displayName}
                  userID={ag.players[1].uuid}
                />
                <RatingBadge rating={ag.players[1].rating} />
              </div>
            </>
          ),
          lexicon: <MatchLexiconDisplay lexiconCode={ag.lexicon} />,
          lexiconCode: ag.lexicon,
          time: timeFormat(
            ag.initialTimeSecs,
            ag.incrementSecs,
            ag.maxOvertimeMinutes,
            ag.gameMode,
          ),
          totalTime: calculateTotalTime(
            ag.initialTimeSecs,
            ag.incrementSecs,
            ag.maxOvertimeMinutes,
          ),
          details: getDetails(),
          player1: ag.players[0].displayName,
          player2: ag.players[1].displayName,
          variant: ag.variant || "classic",
        };
      });
    return gameData;
  };
  const columns = [
    {
      title: "Players",
      className: "players",
      dataIndex: "players",
      key: "players",
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
      filteredValue: lobbyFilterByLexiconArray,
      filterMultiple: true,
      onFilter: (value: React.Key | boolean, record: ActiveGameTableData) =>
        typeof value === "string" && record.lexiconCode === value,
    },
    {
      title: "Time",
      className: "time",
      dataIndex: "time",
      key: "time",
      sorter: (a: ActiveGameTableData, b: ActiveGameTableData) =>
        a.totalTime - b.totalTime,
    },
    {
      title: "Details",
      className: "details",
      dataIndex: "details",
      key: "details",
    },
  ];

  // Get formatted data
  const formattedGames = formatGameData(props.activeGames);

  let title = <>Resume</>;
  if (props.type !== "RESUME") {
    title = isAdmin ? (
      <>
        {"Games live now"}
        <span className="game-count">{props.activeGames?.length}</span>
      </>
    ) : (
      <>Games live now</>
    );
  }

  // For RESUME mode, show single table without grouping
  if (props.type === "RESUME") {
    return (
      <>
        <h4>{title}</h4>
        <Table
          className="games observe"
          dataSource={formattedGames}
          columns={columns}
          pagination={false}
          rowKey="gameID"
          showSorterTooltip={false}
          onRow={(record) => {
            return {
              onClick: (event) => {
                if (event.ctrlKey || event.altKey || event.metaKey) {
                  window.open(`/game/${encodeURIComponent(record.gameID)}`);
                } else {
                  navigate(`/game/${encodeURIComponent(record.gameID)}`);
                  console.log("redirecting to", record.gameID);
                }
              },
              onAuxClick: (event) => {
                if (event.button === 1) {
                  // middle-click
                  window.open(`/game/${encodeURIComponent(record.gameID)}`);
                }
              },
            };
          }}
          rowClassName={(record) => {
            if (
              props.username &&
              (record.player1 === props.username ||
                record.player2 === props.username)
            ) {
              return "game-listing resume";
            }
            return "game-listing";
          }}
          onChange={handleChange}
        />
      </>
    );
  }

  // For WATCH mode, group by variant
  const groupedGames = groupGamesByVariant(formattedGames);

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
      <h4>{title}</h4>

      {sortedVariants.map((variant) => (
        <React.Fragment key={variant}>
          <VariantSectionHeader variant={variant} />
          <Table
            className="games observe"
            dataSource={groupedGames[variant]}
            columns={columns}
            pagination={false}
            rowKey="gameID"
            showSorterTooltip={false}
            onRow={(record) => {
              return {
                onClick: (event) => {
                  if (event.ctrlKey || event.altKey || event.metaKey) {
                    window.open(`/game/${encodeURIComponent(record.gameID)}`);
                  } else {
                    navigate(`/game/${encodeURIComponent(record.gameID)}`);
                    console.log("redirecting to", record.gameID);
                  }
                },
                onAuxClick: (event) => {
                  if (event.button === 1) {
                    // middle-click
                    window.open(`/game/${encodeURIComponent(record.gameID)}`);
                  }
                },
              };
            }}
            rowClassName={(record) => {
              if (
                props.username &&
                (record.player1 === props.username ||
                  record.player2 === props.username)
              ) {
                return "game-listing resume";
              }
              return "game-listing";
            }}
            onChange={handleChange}
          />
        </React.Fragment>
      ))}
    </>
  );
};
