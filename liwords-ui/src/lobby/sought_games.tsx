import { Popconfirm, Table, Tooltip } from "antd";
import {
  FilterValue,
  SorterResult,
  TableCurrentDataSource,
  TablePaginationConfig,
} from "antd/lib/table/interface";
import React, { ReactNode, useCallback, useMemo, useState } from "react";
import { FundOutlined, ExportOutlined } from "@ant-design/icons";
import {
  calculateTotalTime,
  challRuleToStr,
  timeCtrlToDisplayName,
  timeToString,
} from "../store/constants";
import {
  SoughtGame,
  matchesRatingFormula,
  hasEstablishedRating,
} from "../store/reducers/lobby_reducer";
import { PlayerAvatar } from "../shared/player_avatar";
import { DisplayUserFlag } from "../shared/display_flag";
import { RatingBadge } from "./rating_badge";
import { VariantIcon } from "../shared/variant_icons";
import { lexiconOrder, MatchLexiconDisplay } from "../shared/lexicon_display";
import { ProfileUpdate_Rating } from "../gen/api/proto/ipc/users_pb";
import { useLobbyStoreContext } from "../store/store";
import { ActionType } from "../actions/actions";
import { DisplayUserBadges } from "../profile/badge";
import { SeekConfirmModal } from "./seek_confirm_modal";
import { normalizeVariant, VariantSectionHeader } from "./variant_utils";

export const timeFormat = (
  initialTimeSecs: number,
  incrementSecs: number,
  maxOvertime: number,
  gameMode?: number,
): string => {
  // Check if this is a correspondence game
  if (gameMode === 1) {
    const days = Math.floor(initialTimeSecs / 86400);
    return `Correspondence ${days} day${days !== 1 ? "s" : ""} per turn`;
  }

  const label = timeCtrlToDisplayName(
    initialTimeSecs,
    incrementSecs,
    maxOvertime,
  )[0];

  return `${label} ${timeToString(
    initialTimeSecs,
    incrementSecs,
    maxOvertime,
  )}`;
};

export const challengeFormat = (cr: number) => {
  return (
    <span className={`challenge-rule mode_${challRuleToStr(cr)}`}>
      {challRuleToStr(cr)}
    </span>
  );
};

type PlayerProps = {
  userID?: string;
  username: string;
};

export const PlayerDisplay = (props: PlayerProps) => {
  return (
    <div className="player-display">
      <PlayerAvatar
        player={{
          userId: props.userID,
          nickname: props.username,
        }}
      />
      <span className="player-name">{props.username}</span>
      <DisplayUserFlag uuid={props.userID} />
      <DisplayUserBadges uuid={props.userID} />
    </div>
  );
};

type Props = {
  isMatch?: boolean;
  newGame: (seekID: string) => void;
  userID?: string;
  username?: string;
  requests: Array<SoughtGame>;
  ratings?: { [key: string]: ProfileUpdate_Rating };
};
export const SoughtGames = (props: Props) => {
  const [cancelVisible, setCancelVisible] = useState(false);
  const [confirmModalVisible, setConfirmModalVisible] = useState(false);
  const [selectedSeek, setSelectedSeek] = useState<SoughtGame | null>(null);
  const {
    lobbyContext: { lobbyFilterByLexicon, profile },
    dispatchLobbyContext,
  } = useLobbyStoreContext();
  const lobbyFilterByLexiconArray = useMemo(
    () => lobbyFilterByLexicon?.match(/\S+/g) ?? [],
    [lobbyFilterByLexicon],
  );
  const columns = [
    {
      title: "Player",
      className: "seeker",
      dataIndex: "seeker",
      key: "seeker",
    },
    {
      title: "Rating",
      className: "rating",
      dataIndex: "ratingBadge",
      key: "rating",
      sorter: (a: SoughtGameTableData, b: SoughtGameTableData) =>
        parseInt(a.rating) - parseInt(b.rating),
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
      onFilter: (value: React.Key | boolean, record: SoughtGameTableData) =>
        typeof value === "string" && record.lexiconCode === value,
    },
    {
      title: "Time",
      className: "time",
      dataIndex: "time",
      key: "time",
      sorter: (a: SoughtGameTableData, b: SoughtGameTableData) =>
        a.totalTime - b.totalTime,
    },
    {
      title: "Details",
      className: "details",
      dataIndex: "details",
      key: "details",
    },
  ];

  const handleChange = useCallback(
    (
      _pagination: TablePaginationConfig,
      filters: Record<string, FilterValue | null>,
      _sorter:
        | SorterResult<SoughtGameTableData>
        | SorterResult<SoughtGameTableData>[],
      extra: TableCurrentDataSource<SoughtGameTableData>,
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

  type SoughtGameTableData = {
    seeker: string | ReactNode;
    rating: string;
    ratingBadge?: ReactNode;
    lexicon: ReactNode;
    lexiconCode: string;
    time: string;
    totalTime: number;
    details?: ReactNode;
    outgoing: boolean;
    seekID: string;
    soughtGame: SoughtGame; // Keep reference to full object
  };

  // Group games by variant
  const groupGamesByVariant = (
    games: SoughtGameTableData[],
  ): { [variant: string]: SoughtGameTableData[] } => {
    const grouped: { [variant: string]: SoughtGameTableData[] } = {};
    games.forEach((game) => {
      const variant = normalizeVariant(game.soughtGame.variant || "classic");
      if (!grouped[variant]) {
        grouped[variant] = [];
      }
      grouped[variant].push(game);
    });
    return grouped;
  };

  const formatGameData = (games: SoughtGame[]): SoughtGameTableData[] => {
    const gameData: SoughtGameTableData[] = games
      .filter((sg: SoughtGame) => {
        if (sg.seeker === props.username || sg.receiverIsPermanent) {
          // If we are the seeker, or if it's a match request, always show it.
          return true;
        }

        // Check rating range
        if (!props.ratings || !matchesRatingFormula(sg, props.ratings)) {
          return false;
        }

        // Check established rating requirement
        if (sg.requireEstablishedRating && props.ratings) {
          if (!hasEstablishedRating(props.ratings, sg.ratingKey)) {
            return false;
          }
        }

        return true;
      })
      .map((sg: SoughtGame): SoughtGameTableData => {
        const getDetails = () => {
          return (
            <>
              <VariantIcon vcode={sg.variant} />{" "}
              {challengeFormat(sg.challengeRule)}
              {sg.rated ? (
                <Tooltip title="Rated">
                  <FundOutlined />
                </Tooltip>
              ) : null}
            </>
          );
        };
        const outgoing = sg.seeker === props.username;
        return {
          seeker: outgoing ? (
            <Popconfirm
              title="Do you want to cancel this game?"
              onConfirm={() => {
                props.newGame(sg.seekID);
                setCancelVisible(false);
              }}
              okText="Yes"
              cancelText="No"
              onCancel={() => {
                console.log("trying", setCancelVisible, cancelVisible);
                setCancelVisible(false);
              }}
              onOpenChange={(visible) => {
                setCancelVisible(visible);
              }}
              open={cancelVisible}
            >
              <div>
                <ExportOutlined />
                {` ${sg.receiver?.displayName || "Seeking..."}`}
              </div>
            </Popconfirm>
          ) : (
            <PlayerDisplay userID={sg.seekerID} username={sg.seeker} />
          ),
          rating: outgoing ? "" : sg.userRating,
          ratingBadge: outgoing ? null : <RatingBadge rating={sg.userRating} />,
          lexicon: <MatchLexiconDisplay lexiconCode={sg.lexicon} />,
          time: timeFormat(
            sg.initialTimeSecs,
            sg.incrementSecs,
            sg.maxOvertimeMinutes,
            sg.gameMode,
          ),
          totalTime: calculateTotalTime(
            sg.initialTimeSecs,
            sg.incrementSecs,
            sg.maxOvertimeMinutes,
          ),
          details: getDetails(),
          outgoing,
          seekID: sg.seekID,
          lexiconCode: sg.lexicon,
          soughtGame: sg,
        };
      });
    return gameData;
  };

  // Get formatted and grouped data
  const formattedGames = formatGameData(props.requests);
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
      {props.isMatch ? <h4>Match requests</h4> : <h4>Available games</h4>}

      {sortedVariants.map((variant) => (
        <React.Fragment key={variant}>
          <VariantSectionHeader variant={variant} />
          <Table
            className={`games ${props.isMatch ? "match" : "seek"}`}
            dataSource={groupedGames[variant]}
            columns={columns}
            pagination={false}
            rowKey="seekID"
            showSorterTooltip={false}
            onRow={(record) => {
              return {
                onClick: () => {
                  if (!record.outgoing) {
                    // Show confirmation modal for all incoming game requests
                    setSelectedSeek(record.soughtGame);
                    setConfirmModalVisible(true);
                  } else if (!cancelVisible) {
                    setCancelVisible(true);
                  }
                },
              };
            }}
            rowClassName={(record) => {
              if (record.outgoing) {
                return "game-listing outgoing";
              }
              return "game-listing";
            }}
            onChange={handleChange}
          />
        </React.Fragment>
      ))}

      <SeekConfirmModal
        open={confirmModalVisible}
        seek={selectedSeek}
        onAccept={() => {
          if (selectedSeek) {
            props.newGame(selectedSeek.seekID);
          }
          setConfirmModalVisible(false);
          setSelectedSeek(null);
        }}
        onCancel={() => {
          setConfirmModalVisible(false);
          setSelectedSeek(null);
        }}
        userRatings={profile?.ratings || {}}
      />
    </>
  );
};
