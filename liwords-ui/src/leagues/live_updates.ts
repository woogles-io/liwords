import { type QueryClient } from "@tanstack/react-query";
import { type LeagueGameUpdatedEvent } from "../gen/api/proto/ipc/ipc_pb";

type QueryFilter = { queryKey: Array<unknown> };

// A connect-query key, matched on its method and the given request fields.
const methodKey = (
  methodName: string,
  input: Record<string, string>,
): QueryFilter => ({ queryKey: ["connect-query", { methodName, input }] });

// Whether the league page's all-time roster is on screen. The page keeps the
// roster loaded to follow players from the standings, even while it is not
// shown, and every ending in the league changes it.
let rosterOnScreen = false;

// The roster says whether it is on screen.
export const setRosterOnScreen = (onScreen: boolean) => {
  rosterOnScreen = onScreen;
};

// What a league game's news makes stale. A move changes its division's
// cross-table (the live score, whose turn it is) and time banks, and its two
// players' season games; an ending also changes the season's standings (which
// the league chat's "Division N (rank)" reads too), the roster if it is on
// screen, and each player's record against the other (in My League Games,
// and in the roster's H2H for whichever player it is showing). Only the
// queries a page is showing refetch, so a page on another season or division
// fetches nothing, unless it shows the all-time roster.
export const leagueGameQueryFilters = (
  ev: Pick<
    LeagueGameUpdatedEvent,
    "seasonId" | "divisionId" | "playerIds" | "ended"
  >,
  rosterShown = rosterOnScreen,
): Array<QueryFilter> => {
  const filters = [
    methodKey("GetDivisionCrosstable", { divisionId: ev.divisionId }),
    methodKey("GetDivisionTimeBankWarnings", { divisionId: ev.divisionId }),
    ...ev.playerIds.map((userId) =>
      methodKey("GetPlayerSeasonGames", { userId, seasonId: ev.seasonId }),
    ),
  ];
  if (ev.ended) {
    filters.push(
      methodKey("GetAllDivisionStandings", { seasonId: ev.seasonId }),
      ...ev.playerIds.map((userId) =>
        methodKey("GetPlayerLeagueH2H", { userId }),
      ),
    );
    if (rosterShown) {
      filters.push({
        queryKey: ["connect-query", { methodName: "GetLeagueRoster" }],
      });
    }
  }
  return filters;
};

const pending = new Map<string, QueryFilter>();
let flush: ReturnType<typeof setTimeout> | undefined;

// Marks the filters' queries stale after a short wait, each once, since
// closing a season ends many games at once.
export const scheduleLeagueRefresh = (
  queryClient: QueryClient,
  filters: Array<QueryFilter>,
  delayMs = 500,
) => {
  for (const f of filters) {
    pending.set(JSON.stringify(f.queryKey), f);
  }
  if (flush !== undefined) return;
  flush = setTimeout(() => {
    flush = undefined;
    const batch = [...pending.values()];
    pending.clear();
    for (const f of batch) {
      void queryClient.invalidateQueries(f);
    }
  }, delayMs);
};
