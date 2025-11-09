import React from "react";
import { Card } from "antd";
import {
  League,
  Season,
} from "../gen/api/proto/league_service/league_service_pb";

type LeagueHeaderCompactProps = {
  league: League;
  displayedSeason: Season | undefined;
  formatSeasonDates: (start: bigint, end: bigint) => string;
};

export const LeagueHeaderCompact: React.FC<LeagueHeaderCompactProps> = ({
  league,
  displayedSeason,
  formatSeasonDates,
}) => {
  return (
    <Card className="league-header-compact" style={{ marginBottom: 16 }}>
      <h3 style={{ margin: 0, marginBottom: 8, fontSize: "18px" }}>
        {league.name}
      </h3>
      <p
        style={{
          margin: 0,
          marginBottom: 12,
          fontSize: "13px",
          lineHeight: "1.4",
        }}
      >
        {league.description}
      </p>
      {displayedSeason &&
        displayedSeason.startDate &&
        displayedSeason.endDate && (
          <div
            style={{
              padding: "8px 12px",
              borderRadius: "4px",
              fontSize: "13px",
              backgroundColor: "#f5f5f5",
            }}
          >
            <div>
              <strong>
                {displayedSeason.status === 1
                  ? "Current Season"
                  : `Season ${displayedSeason.seasonNumber}`}
                :
              </strong>{" "}
              {formatSeasonDates(
                displayedSeason.startDate,
                displayedSeason.endDate,
              )}
            </div>
          </div>
        )}
    </Card>
  );
};
