/* eslint-disable jsx-a11y/click-events-have-key-events */
/* eslint-disable jsx-a11y/no-noninteractive-element-interactions */
import { Tag, Badge, Table } from 'antd';
import React, { ReactNode } from 'react';
import { timeCtrlToDisplayName, challRuleToStr } from '../store/constants';
import { RatingBadge } from './rating_badge';
import { SoughtGame } from '../store/reducers/lobby_reducer';

type SoughtGameProps = {
  game: SoughtGame;
  newGame: (seekID: string) => void;
  userID?: string;
  username?: string;
  requests: SoughtGame[];
};

/* const SoughtGameItem = (props: SoughtGameProps) => {
  const { game, userID, username } = props;
  const [tt, tc] = timeCtrlToDisplayName(
    game.initialTimeSecs,
    game.incrementSecs,
    game.maxOvertimeMinutes
  );
  let innerel = (
    // eslint-disable-next-line jsx-a11y/no-static-element-interactions
    <div
      className="game"
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
*/

type Props = {
  isMatch?: boolean;
  newGame: (seekID: string) => void;
  userID?: string;
  username?: string;
  requests: Array<SoughtGame>;
};

export const SoughtGames = (props: Props) => {
  const columns = [
    {
      title: 'Player',
      className: 'seeker',
      dataIndex: 'seeker',
      key: 'seeker',
    },
    {
      title: 'Rating',
      className: 'rating',
      dataIndex: 'rating',
      key: 'rating',
    },
    {
      title: 'Words',
      className: 'lexicon',
      dataIndex: 'lexicon',
      key: 'lexicon',
    },
    {
      title: 'Time',
      className: 'time',
      dataIndex: 'time',
      key: 'time',
    },
    {
      title: 'Challenge',
      className: 'details',
      dataIndex: 'challengeRule',
      key: 'details',
    },
  ];

  type SoughtGameTableData = {
    seeker: string;
    rating: string;
    lexicon: string;
    time: string;
    challengeRule?: ReactNode;
    seekID: string;
  };

  const formatGameData = (games: SoughtGame[]): SoughtGameTableData[] => {
    const gameData: SoughtGameTableData[] = games.map(
      (sg: SoughtGame): SoughtGameTableData => {
        const challengeFormat = (cr: number) => {
          return (
            <span className={`mode_${challRuleToStr(cr)}`}>
              {challRuleToStr(cr)}
            </span>
          );
        };
        const timeFormat = (
          initialTimeSecs: number,
          incrementSecs: number
        ): string => {
          // TODO: Pull in from time control in seek window
          const label =
            initialTimeSecs > 900
              ? 'Regular'
              : initialTimeSecs > 300
              ? 'Rapid'
              : 'Blitz';
          return `${label} ${initialTimeSecs / 60}/${incrementSecs}`;
        };
        return {
          seeker: sg.seeker,
          rating: sg.userRating,
          lexicon: sg.lexicon,
          time: timeFormat(sg.initialTimeSecs, sg.incrementSecs),
          challengeRule: challengeFormat(sg.challengeRule),
          seekID: sg.seekID,
        };
      }
    );
    return gameData;
  };

  return (
    <>
      {props.isMatch ? <h4>Incoming Requests</h4> : <h4>Available Games</h4>}
      <Table
        className={`games ${props.isMatch ? 'match' : 'seek'}`}
        dataSource={formatGameData(props.requests)}
        columns={columns}
        pagination={{
          hideOnSinglePage: true,
        }}
        onRow={(record, rowIndex) => {
          return {
            onClick: (event) => {
              props.newGame(record.seekID);
            },
          };
        }}
        rowClassName="game-listing"
      />
    </>
  );
};
