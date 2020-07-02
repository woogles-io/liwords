/* eslint-disable jsx-a11y/click-events-have-key-events */
/* eslint-disable jsx-a11y/no-noninteractive-element-interactions */
import { Card, Tag } from 'antd';
import React from 'react';
import { useStoreContext } from '../store/store';
import { ChallengeRule } from '../gen/macondo/api/proto/macondo/macondo_pb';
import { timeCtrlToDisplayName } from '../store/constants';

export const challRuleToStr = (n: number): string => {
  switch (n) {
    case ChallengeRule.DOUBLE:
      return 'Double';
    case ChallengeRule.SINGLE:
      return 'Single';
    case ChallengeRule.FIVE_POINT:
      return '5-pt';
    case ChallengeRule.TEN_POINT:
      return '10-pt';
    case ChallengeRule.VOID:
      return 'Void';
  }
  return 'Unsupported';
};

type Props = {
  newGame: (seekID: string) => void;
};

export const SoughtGames = (props: Props) => {
  const { lobbyContext } = useStoreContext();

  const soughtGameEls = lobbyContext?.soughtGames.map((game) => {
    const [tt, tc] = timeCtrlToDisplayName(game.initialTimeSecs);

    return (
      <li
        key={`game${game.seeker}`}
        style={{ paddingTop: 20, cursor: 'pointer' }}
        onClick={(event: React.MouseEvent) => props.newGame(game.seekID)}
      >
        {game.seeker} ({game.userRating}) wants to play {game.lexicon} (
        {game.rated ? 'Rated' : 'Casual'})(
        {`${game.initialTimeSecs / 60} min`})<Tag color={tc}>{tt}</Tag>
        {challRuleToStr(game.challengeRule)}
      </li>
    );
  });

  return (
    <Card title="Join a game">
      <ul style={{ listStyleType: 'none' }}>{soughtGameEls}</ul>
    </Card>
  );
};
