/* eslint-disable jsx-a11y/click-events-have-key-events */
/* eslint-disable jsx-a11y/no-noninteractive-element-interactions */
import { Card } from 'antd';
import React from 'react';
import {
  ChallengeRuleMap,
  ChallengeRule,
} from '../gen/api/proto/game_service_pb';
import { useStoreContext } from '../store/store';

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
  const { soughtGames } = useStoreContext();

  const soughtGameEls = soughtGames.map((game, idx) => (
    <li
      key={`game${game.seeker}`}
      style={{ paddingTop: 20, cursor: 'pointer' }}
      onClick={(event: React.MouseEvent) => props.newGame(game.seekID)}
    >
      {game.seeker} wants to play {game.lexicon} (
      {`${game.initialTimeSecs / 60} min`}) {challRuleToStr(game.challengeRule)}
      seekID: {game.seekID})
    </li>
  ));

  return (
    <Card title="Join a game">
      <ul style={{ listStyleType: 'none' }}>{soughtGameEls}</ul>
    </Card>
  );
};
