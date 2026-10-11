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

// Previous / next season of the player being followed, skipping seasons they
// were not in.
export const FollowBar = (props: Props) => {
  const { prev, next } = props;
  return (
    <div className="league-follow-bar">
      <Button
        size="small"
        icon={<LeftOutlined />}
        disabled={!prev}
        onClick={() => prev && props.onGo(prev)}
      >
        {prev ? seasonLabel(prev) : null}
      </Button>
      <span className="league-follow-name">Following {props.username}</span>
      <Button
        size="small"
        icon={<RightOutlined />}
        iconPosition="end"
        disabled={!next}
        onClick={() => next && props.onGo(next)}
      >
        {next ? seasonLabel(next) : null}
      </Button>
      <Button size="small" type="link" onClick={props.onStop}>
        Stop
      </Button>
    </div>
  );
};
