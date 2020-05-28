import { Card } from 'antd';
import React from 'react';
import {
  ChallengeRuleMap,
  ChallengeRule,
} from '../gen/api/proto/game_service_pb';
import { useLobbyContext } from '../store/lobby_store';

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

export const SoughtGames = () => {
  const { soughtGames } = useLobbyContext();
  console.log('rendering soughtgames', soughtGames);

  const soughtGameEls = soughtGames.map((game, idx) => (
    // eslint-disable-next-line jsx-a11y/no-noninteractive-element-interactions
    // eslint-disable-next-line jsx-a11y/click-events-have-key-events
    <li
      key={`game${game.seeker}`}
      style={{ paddingTop: 20, cursor: 'pointer' }}
      // onClick={(event: React.MouseEvent) => newGame(game.seeker, username)}
    >
      {game.seeker} wants to play {game.lexicon} (
      {`${game.initialTimeSecs / 60} min`}) {challRuleToStr(game.challengeRule)}
      )
    </li>
  ));

  return (
    <Card title="Join a game">
      <ul style={{ listStyleType: 'none' }}>{soughtGameEls}</ul>
    </Card>
  );
};
