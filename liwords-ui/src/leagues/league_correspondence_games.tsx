import React, { useMemo } from "react";
import { Card, Empty, Tooltip } from "antd";
import { useNavigate } from "react-router";
import { useQuery } from "@connectrpc/connect-query";
import { CorrespondenceTurnIndicator } from "../shared/corres_turn_indicator";
import { useLobbyStoreContext } from "../store/store";
import { useLoginStateStoreContext } from "../store/store";
import { onTurnCountdowns } from "../utils/time_bank_calculator";
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
      {wld} ({spreadStr})
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

    // Real time-until-expiry for whoever is on turn: per-turn allowance + that
    // player's remaining bank - elapsed. League games always have a time bank,
    // so the old bank-blind `incrementSecs - elapsed` proxy sorted them wrong
    // (it degenerated to roughly updated_at order). Lower = more urgent.
    const now = Date.now();
    const expiryOf = (game: (typeof filtered)[number]) => {
      if (game.playerOnTurn === undefined || !game.lastUpdate) {
        return Infinity;
      }
      const elapsedSecs = (now - game.lastUpdate) / 1000;
      const bankSecs = (game.timeBank?.[game.playerOnTurn] ?? 0) / 1000;
      return onTurnCountdowns(game.incrementSecs, bankSecs, elapsedSecs)
        .beforeExpiry;
    };

    // Sort: user's turn first, then by the real time-until-expiry (ascending).
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

      return expiryOf(a) - expiryOf(b);
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

          // Bank-aware countdowns for the on-turn player: before-expiry is the
          // hard deadline (per-turn allowance + bank - elapsed), before-bleed
          // is the free-time window before the bank starts draining. The shared
          // CorrespondenceTurnIndicator renders these (and the low-time /
          // time-bank escalations) from the values below.
          const now = Date.now();
          let countdowns: ReturnType<typeof onTurnCountdowns> | undefined;
          let hasTimeBank = false;
          if (game.playerOnTurn !== undefined && game.lastUpdate) {
            const timeElapsedSecs = (now - game.lastUpdate) / 1000;
            const bankSecs = (game.timeBank?.[game.playerOnTurn] ?? 0) / 1000;
            hasTimeBank = bankSecs > 0;
            countdowns = onTurnCountdowns(
              game.incrementSecs,
              bankSecs,
              timeElapsedSecs,
            );
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
                {(isUserTurn || countdowns) && (
                  <div className="turn-indicator-compact">
                    <CorrespondenceTurnIndicator
                      onTurn={!!isUserTurn}
                      countdowns={countdowns}
                      hasTimeBank={hasTimeBank}
                    />
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
