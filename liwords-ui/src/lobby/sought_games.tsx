/* eslint-disable jsx-a11y/click-events-have-key-events */
/* eslint-disable jsx-a11y/no-noninteractive-element-interactions */
import { Card, Tag, Badge } from 'antd';
import React from 'react';
import { useStoreContext } from '../store/store';
import { timeCtrlToDisplayName, challRuleToStr } from '../store/constants';
import { RatingBadge } from './rating_badge';
import { SoughtGame } from '../store/reducers/lobby_reducer';

type SoughtGameProps = {
  game: SoughtGame;
  newGame: (seekID: string) => void;
  userID?: string;
  username?: string;
};

const SoughtGameItem = (props: SoughtGameProps) => {
  const { game, userID, username } = props;
  const [tt, tc] = timeCtrlToDisplayName(
    game.initialTimeSecs,
    game.incrementSecs,
    game.maxOvertimeMinutes
  );
  let innerel = (
    // eslint-disable-next-line jsx-a11y/no-static-element-interactions
    <div
      style={{ paddingTop: 20, cursor: 'pointer' }}
      onClick={(event: React.MouseEvent) => props.newGame(game.seekID)}
    >
      <RatingBadge rating={game.userRating} player={game.seeker} />
      {game.lexicon} ({game.rated ? 'Rated' : 'Casual'})(
      {`${game.initialTimeSecs / 60} / ${game.incrementSecs}`})
      <Tag color={tc}>{tt}</Tag>
      {challRuleToStr(game.challengeRule)}
      {game.incrementSecs === 0
        ? ` (Max OT: ${game.maxOvertimeMinutes} min.)`
        : ''}
    </div>
  );

  // game.seeker is a username - it is for display
  if (game.receiver && game.receiver.getUserId() !== '') {
    console.log('reciever', game.receiver);

    if (userID === game.receiver.getUserId()) {
      // This is the receiver of the match request.
      innerel = (
        <Badge.Ribbon text="Match Request" color="volcano">
          {innerel}
        </Badge.Ribbon>
      );
    } else {
      // We must be the sender of the match request.
      if (username !== game.seeker) {
        throw new Error(`unexpected seeker${username}, ${game.seeker}`);
      }
      innerel = (
        <Badge.Ribbon
          text={`Outgoing Match Request to ${game.receiver.getDisplayName()}`}
        >
          {innerel}
        </Badge.Ribbon>
      );
    }
  }

  return <li>{innerel}</li>;
};

type Props = {
  newGame: (seekID: string) => void;
  userID: string;
  username: string;
};

export const SoughtGames = (props: Props) => {
  const { lobbyContext } = useStoreContext();

  const soughtGameEls = lobbyContext?.soughtGames.map((game) => {
    console.log('game', game);
    return (
      <SoughtGameItem
        game={game}
        newGame={props.newGame}
        key={`game${game.seekID}`}
      />
    );
  });

  const matchReqEls = lobbyContext?.matchRequests.map((game) => {
    console.log('matchreq', game);
    return (
      <SoughtGameItem
        game={game}
        newGame={props.newGame}
        key={`game${game.seekID}`}
        userID={props.userID}
        username={props.username}
      />
    );
  });

  return (
    <Card title="Join a game">
      <ul style={{ listStyleType: 'none' }}>{matchReqEls}</ul>

      <ul style={{ listStyleType: 'none' }}>{soughtGameEls}</ul>
    </Card>
  );
};
