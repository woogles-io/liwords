/* eslint-disable jsx-a11y/click-events-have-key-events */
/* eslint-disable jsx-a11y/no-noninteractive-element-interactions */
import { Card, Tag } from 'antd';
import React from 'react';
import { useStoreContext } from '../store/store';
import { timeCtrlToDisplayName, challRuleToStr } from '../store/constants';
import { RatingBadge } from './rating_badge';

type Props = {
  newGame: (seekID: string) => void;
};

export const SoughtGames = (props: Props) => {
  const { lobbyContext } = useStoreContext();

  const soughtGameEls = lobbyContext?.soughtGames.map((game) => {
    console.log('game', game);
    const [tt, tc] = timeCtrlToDisplayName(game.initialTimeSecs);

    return (
      <li
        key={`game${game.seeker}`}
        style={{ paddingTop: 20, cursor: 'pointer' }}
        onClick={(event: React.MouseEvent) => props.newGame(game.seekID)}
      >
        <RatingBadge rating={game.userRating} player={game.seeker} /> wants to
        play {game.lexicon} ({game.rated ? 'Rated' : 'Casual'})(
        {`${game.initialTimeSecs / 60} min`})<Tag color={tc}>{tt}</Tag>
        {challRuleToStr(game.challengeRule)}
        {` (Max OT: ${game.maxOvertimeMinutes} min.)`}
      </li>
    );
  });

  return (
    <Card title="Join a game">
      <ul style={{ listStyleType: 'none' }}>{soughtGameEls}</ul>
    </Card>
  );
};
