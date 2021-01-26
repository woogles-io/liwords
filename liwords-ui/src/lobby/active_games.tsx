import { Table, Tooltip } from 'antd';
import React, { ReactNode } from 'react';
import { useHistory } from 'react-router-dom';
import { FundOutlined } from '@ant-design/icons/lib';
import { RatingBadge } from './rating_badge';
import { challengeFormat, timeFormat } from './sought_games';
import { ActiveGame } from '../store/reducers/lobby_reducer';
import { calculateTotalTime } from '../store/constants';

type Props = {
  activeGames: ActiveGame[];
  username?: string;
  type?: 'RESUME';
};

export const ActiveGames = (props: Props) => {
  const history = useHistory();

  type ActiveGameTableData = {
    gameID: string;
    players: ReactNode;
    lexicon: string;
    time: string;
    totalTime: number;
    details?: ReactNode;
    player1: string;
    player2: string;
  };
  const formatGameData = (games: ActiveGame[]): ActiveGameTableData[] => {
    const gameData: ActiveGameTableData[] = games.map(
      (ag: ActiveGame): ActiveGameTableData => {
        const getDetails = () => {
          return (
            <>
              {challengeFormat(ag.challengeRule)}
              {ag.rated ? (
                <Tooltip title="Rated">
                  <FundOutlined />
                </Tooltip>
              ) : null}
            </>
          );
        };
        return {
          gameID: ag.gameID,
          players: (
            <>
              <RatingBadge
                rating={ag.players[0].rating}
                player={ag.players[0].displayName}
              />
              {'vs.  '}
              <RatingBadge
                rating={ag.players[1].rating}
                player={ag.players[1].displayName}
              />
            </>
          ),
          lexicon: ag.lexicon,
          time: timeFormat(
            ag.initialTimeSecs,
            ag.incrementSecs,
            ag.maxOvertimeMinutes
          ),
          totalTime: calculateTotalTime(
            ag.initialTimeSecs,
            ag.incrementSecs,
            ag.maxOvertimeMinutes
          ),
          details: getDetails(),
          player1: ag.players[0].displayName,
          player2: ag.players[1].displayName,
        };
      }
    );
    return gameData;
  };
  const columns = [
    {
      title: 'Players',
      className: 'players',
      dataIndex: 'players',
      key: 'players',
    },
    {
      title: 'Words',
      className: 'lexicon',
      dataIndex: 'lexicon',
      key: 'lexicon',
      filters: [
        {
          text: 'CSW19',
          value: 'CSW19',
        },
        {
          text: 'NWL20',
          value: 'NWL20',
        },
        {
          text: 'NWL18',
          value: 'NWL18',
        },
        {
          text: 'ECWL',
          value: 'ECWL',
        },
      ],
      filterMultiple: false,
      onFilter: (
        value: string | number | boolean,
        record: ActiveGameTableData
      ) => record.lexicon.indexOf(value.toString()) === 0,
    },
    {
      title: 'Time',
      className: 'time',
      dataIndex: 'time',
      key: 'time',
      sorter: (a: ActiveGameTableData, b: ActiveGameTableData) =>
        a.totalTime - b.totalTime,
    },
    {
      title: 'Details',
      className: 'details',
      dataIndex: 'details',
      key: 'details',
    },
  ];

  return (
    <>
      <h4>{props.type === 'RESUME' ? 'Resume' : 'Games Live Now'}</h4>
      <Table
        className="games observe"
        dataSource={formatGameData(props.activeGames)}
        columns={columns}
        pagination={false}
        rowKey="gameID"
        showSorterTooltip={false}
        onRow={(record) => {
          return {
            onClick: (event) => {
              if (event.ctrlKey || event.altKey || event.metaKey) {
                window.open(`/game/${encodeURIComponent(record.gameID)}`);
              } else {
                history.replace(`/game/${encodeURIComponent(record.gameID)}`);
                console.log('redirecting to', record.gameID);
              }
            },
            onAuxClick: (event) => {
              if (event.button === 1) {
                // middle-click
                window.open(`/game/${encodeURIComponent(record.gameID)}`);
              }
            },
          };
        }}
        rowClassName={(record) => {
          if (
            props.username &&
            (record.player1 === props.username ||
              record.player2 === props.username)
          ) {
            return 'game-listing resume';
          }
          return 'game-listing';
        }}
      />
    </>
  );
};
