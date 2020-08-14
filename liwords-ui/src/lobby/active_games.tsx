import { Tag, Button } from 'antd';
import React from 'react';
import { useStoreContext } from '../store/store';
import { timeCtrlToDisplayName, challRuleToStr } from '../store/constants';
import { RatingBadge } from './rating_badge';

type Props = {
  username?: string;
};

export const ActiveGames = (props: Props) => {
  const { lobbyContext, setRedirGame } = useStoreContext();

  const activeGameEls = lobbyContext?.activeGames.map((game) => {
    console.log('game', game);
    const [tt, tc] = timeCtrlToDisplayName(
      game.initialTimeSecs,
      game.incrementSecs,
      game.maxOvertimeMinutes
    );

    return (
      <li key={`game${game.gameID}`} style={{ paddingTop: 20 }}>
        <Button
          onClick={(event: React.MouseEvent) => {
            setRedirGame(game.gameID);
            console.log('redirecting to', game.gameID);
          }}
          type="primary"
          style={{ marginRight: 10 }}
        >
          {props.username &&
          (game.players[0].displayName === props.username ||
            game.players[1].displayName === props.username)
            ? 'Resume Game'
            : 'Watch!'}
        </Button>
        <RatingBadge
          rating={game.players[0].rating}
          player={game.players[0].displayName}
        />
        {'vs.  '}
        <RatingBadge
          rating={game.players[1].rating}
          player={game.players[1].displayName}
        />{' '}
        ({game.lexicon}) ({game.rated ? 'Rated' : 'Casual'})(
        {`${game.initialTimeSecs / 60} min`})<Tag color={tc}>{tt}</Tag>
        {challRuleToStr(game.challengeRule)}
        {` (Max OT: ${game.maxOvertimeMinutes} min.)`}
      </li>
    );
  });

  return (
    <div>
      <ul style={{ listStyleType: 'none' }}>{activeGameEls}</ul>
    </div>
  );
};
