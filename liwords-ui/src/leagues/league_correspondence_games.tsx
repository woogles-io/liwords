import React, { useMemo } from "react";
import { Card, Empty, Tooltip } from "antd";
import { ClockCircleOutlined } from "@ant-design/icons";
import { useNavigate } from "react-router";
import { useQuery } from "@connectrpc/connect-query";
import { useLobbyStoreContext } from "../store/store";
import { useLoginStateStoreContext } from "../store/store";
import { getPlayerLeagueH2H } from "../gen/api/proto/league_service/league_service-LeagueService_connectquery";
import { H2HRecord } from "../gen/api/proto/league_service/league_service_pb";

type LeagueCorrespondenceGamesProps = {
  leagueSlug: string;
};

const formatH2HCompact = (record: H2HRecord | undefined) => {
  if (!record) return null;
  const { wins, losses, draws, spread } = record;
  const spreadStr = spread > 0 ? `+${spread}` : `${spread}`;
  const wld = `${wins}-${losses}${draws ? `-${draws}` : ""}`;
  return (
    <span
      style={{
        fontSize: "11px",
        opacity: 0.7,
      }}
    >
      {wld} {spreadStr}
    </span>
  );
};

export const LeagueCorrespondenceGames: React.FC<
  LeagueCorrespondenceGamesProps
> = ({ leagueSlug }) => {
  const navigate = useNavigate();
  const {
    loginState: { userID },
  } = useLoginStateStoreContext();
  const {
    lobbyContext: { correspondenceGames },
  } = useLobbyStoreContext();

  // Fetch h2h data for the logged-in user
  const { data: h2hData } = useQuery(
    getPlayerLeagueH2H,
    {
      userId: userID || "",
      leagueId: leagueSlug,
    },
    { enabled: !!userID },
  );

  // Build a map of opponent UUID -> H2HRecord
  const h2hMap = useMemo(() => {
    const map = new Map<string, H2HRecord>();
    if (h2hData?.records) {
      for (const record of h2hData.records) {
        map.set(record.opponentUserId, record);
      }
    }
    return map;
  }, [h2hData?.records]);

  // Filter games for this league and sort by user's turn
  const leagueGames = useMemo(() => {
    const filtered = correspondenceGames.filter(
      (game) => game.leagueSlug === leagueSlug,
    );

    // Sort: user's turn first, then by time remaining
    return filtered.sort((a, b) => {
      const aOnTurn =
        a.playerOnTurn !== undefined &&
        userID &&
        a.players.findIndex((p) => p.uuid === userID) === a.playerOnTurn;
      const bOnTurn =
        b.playerOnTurn !== undefined &&
        userID &&
        b.players.findIndex((p) => p.uuid === userID) === b.playerOnTurn;

      if (aOnTurn && !bOnTurn) return -1;
      if (!aOnTurn && bOnTurn) return 1;

      // Calculate time remaining for sorting
      const now = Date.now();
      const aTimeRemaining =
        a.lastUpdate && a.incrementSecs
          ? a.incrementSecs - (now - a.lastUpdate) / 1000
          : Infinity;
      const bTimeRemaining =
        b.lastUpdate && b.incrementSecs
          ? b.incrementSecs - (now - b.lastUpdate) / 1000
          : Infinity;

      return aTimeRemaining - bTimeRemaining;
    });
  }, [correspondenceGames, leagueSlug, userID]);

  const handleGameClick = (gameID: string, event: React.MouseEvent) => {
    // Same technique as lobby correspondence games:
    // - Regular click: navigate in same window
    // - Ctrl/Alt/Meta + click: open in new window
    // - Middle-click: open in new window
    if (event.ctrlKey || event.altKey || event.metaKey) {
      window.open(`/game/${encodeURIComponent(gameID)}`);
    } else {
      navigate(`/game/${encodeURIComponent(gameID)}`);
    }
  };

  if (leagueGames.length === 0) {
    return (
      <Card title="My League Games">
        <Empty
          description="No active league games"
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        />
      </Card>
    );
  }

  return (
    <Card title="My League Games">
      <div className="league-games-list">
        {leagueGames.map((game) => {
          const isUserTurn =
            game.playerOnTurn !== undefined &&
            userID &&
            game.players.findIndex((p) => p.uuid === userID) ===
              game.playerOnTurn;

          // Calculate if low time (< 24 hours)
          const now = Date.now();
          let isLowTime = false;
          if (isUserTurn && game.lastUpdate && game.incrementSecs) {
            const timeElapsedSecs = (now - game.lastUpdate) / 1000;
            const timeRemainingSecs = game.incrementSecs - timeElapsedSecs;
            isLowTime = timeRemainingSecs < 86400; // 24 hours
          }

          // Get opponent name and scores
          const userPlayerIndex = game.players.findIndex(
            (p) => p.uuid === userID,
          );
          const opponentIndex = userPlayerIndex === 0 ? 1 : 0;
          const opponentName = game.players[opponentIndex]?.displayName || "?";
          const opponentUuid = game.players[opponentIndex]?.uuid;

          // Get h2h record for this opponent
          const h2hRecord = opponentUuid ? h2hMap.get(opponentUuid) : undefined;

          // Get scores (user's score first)
          const userScore =
            game.scores && userPlayerIndex >= 0
              ? game.scores[userPlayerIndex]
              : undefined;
          const opponentScore =
            game.scores && opponentIndex >= 0
              ? game.scores[opponentIndex]
              : undefined;
          const hasScores =
            userScore !== undefined && opponentScore !== undefined;
          const spread = hasScores ? userScore - opponentScore : 0;

          return (
            <div
              key={game.gameID}
              className={`league-game-item compact ${isUserTurn ? "user-turn" : ""}`}
              onClick={(event) => handleGameClick(game.gameID, event)}
              onAuxClick={(event) => {
                if (event.button === 1) {
                  // middle-click
                  window.open(`/game/${encodeURIComponent(game.gameID)}`);
                }
              }}
              style={{ cursor: "pointer" }}
            >
              <div className="game-info-compact">
                <div
                  className="opponent-name-compact"
                  style={{
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap",
                    display: "flex",
                    justifyContent: "space-between",
                    alignItems: "center",
                    gap: "8px",
                  }}
                >
                  <span>
                    vs {opponentName}
                    {h2hRecord && (
                      <>
                        {" "}
                        <Tooltip
                          title={`Lifetime league record: ${h2hRecord.wins}W-${h2hRecord.losses}L${h2hRecord.draws ? `-${h2hRecord.draws}D` : ""} ${h2hRecord.spread > 0 ? "+" : ""}${h2hRecord.spread}`}
                        >
                          {formatH2HCompact(h2hRecord)}
                        </Tooltip>
                      </>
                    )}
                  </span>
                  {hasScores && (
                    <span
                      style={{
                        fontWeight: 500,
                        fontSize: "13px",
                        whiteSpace: "nowrap",
                      }}
                    >
                      <Tooltip
                        placement="left"
                        title={`${spread >= 0 ? "+" : ""}${spread}`}
                      >
                        {userScore}-{opponentScore}
                      </Tooltip>
                    </span>
                  )}
                </div>
                {isUserTurn && (
                  <div className="turn-indicator-compact">
                    {isLowTime && (
                      <Tooltip title="Less than 24 hours remaining">
                        <ClockCircleOutlined
                          style={{ color: "#ff4d4f", marginRight: 4 }}
                        />
                      </Tooltip>
                    )}
                    <span className={isLowTime ? "low-time" : ""}>
                      Your turn
                    </span>
                  </div>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </Card>
  );
};
