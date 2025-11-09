import React, { useMemo } from "react";
import { Card, Empty, Tooltip } from "antd";
import { ClockCircleOutlined } from "@ant-design/icons";
import { useNavigate } from "react-router";
import { ActiveGame } from "../store/reducers/lobby_reducer";
import { useLobbyStoreContext } from "../store/store";
import { useLoginStateStoreContext } from "../store/store";

type LeagueCorrespondenceGamesProps = {
  leagueSlug: string;
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

  const handleGameClick = (gameID: string) => {
    navigate(`/game/${encodeURIComponent(gameID)}`);
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

          // Get opponent name
          const userPlayerIndex = game.players.findIndex(
            (p) => p.uuid === userID,
          );
          const opponentIndex = userPlayerIndex === 0 ? 1 : 0;
          const opponentName = game.players[opponentIndex]?.displayName || "?";

          return (
            <div
              key={game.gameID}
              className={`league-game-item compact ${isUserTurn ? "user-turn" : ""}`}
              onClick={() => handleGameClick(game.gameID)}
              style={{ cursor: "pointer" }}
            >
              <div className="game-info-compact">
                <div
                  className="opponent-name-compact"
                  style={{
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap",
                  }}
                >
                  vs {opponentName}
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
