import { LeftOutlined, RightOutlined } from "@ant-design/icons";
import { Button } from "antd";
import React from "react";
import { FollowedSeason } from "./follow_player";

type Props = {
  username: string;
  prev?: FollowedSeason;
  next?: FollowedSeason;
  onGo: (season: FollowedSeason) => void;
  onStop: () => void;
};

export const seasonLabel = (s: FollowedSeason) =>
  s.divisionNumber > 0
    ? `S${s.seasonNumber} D${s.divisionNumber}`
    : `S${s.seasonNumber}`;

// Previous / next season of the player being tracked, skipping seasons they
// were not in. One quiet line of links; a side with nowhere to go is left out.
export const FollowBar = (props: Props) => {
  const { prev, next } = props;
  return (
    <div className="league-follow-bar">
      {prev && (
        <Button
          type="link"
          size="small"
          icon={<LeftOutlined />}
          onClick={() => props.onGo(prev)}
        >
          {seasonLabel(prev)}
        </Button>
      )}
      <span className="league-follow-name">Tracking {props.username}</span>
      {next && (
        <Button
          type="link"
          size="small"
          icon={<RightOutlined />}
          iconPosition="end"
          onClick={() => props.onGo(next)}
        >
          {seasonLabel(next)}
        </Button>
      )}
      <Button type="link" size="small" onClick={props.onStop}>
        Stop
      </Button>
    </div>
  );
};
