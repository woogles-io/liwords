import React from "react";
import { Card } from "antd";
import { PlayerAvatar } from "../shared/player_avatar";
import { useLoginStateStoreContext } from "../store/store";
import { FallOutlined, RiseOutlined } from "@ant-design/icons";

type Props = {
  userRating?: number;
  puzzleRating?: number;
  initialUserRating?: number;
};

export const RatingsCard = React.memo((props: Props) => {
  const { loginState } = useLoginStateStoreContext();
  const { username, loggedIn } = loginState;
  const { initialUserRating, userRating, puzzleRating } = props;
  if (!(loggedIn || userRating)) {
    return null;
  }
  return (
    <Card className="puzzle-rating-card">
      {loggedIn && (
        <div className="player">
          <PlayerAvatar username={username} />
          {!!userRating && userRating.toString()}
          {(userRating || 0) > (initialUserRating || 0) && <RiseOutlined />}
          {(userRating || 0) < (initialUserRating || 0) && <FallOutlined />}
        </div>
      )}
      {!!puzzleRating && (
        <div className="puzzle">
          <PlayerAvatar icon={<i className="fa-solid fa-puzzle-piece" />} />
          {!!puzzleRating && puzzleRating.toString()}
          {(userRating || 0) > (initialUserRating || 0) && <FallOutlined />}
          {(userRating || 0) < (initialUserRating || 0) && <RiseOutlined />}
        </div>
      )}
    </Card>
  );
});
