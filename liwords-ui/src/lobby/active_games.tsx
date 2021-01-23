import { Table, Tooltip } from 'antd';
import React, { ReactNode } from 'react';
import { useHistory } from 'react-router-dom';
import { FundOutlined } from '@ant-design/icons/lib';
import { RatingBadge } from './rating_badge';
import { challengeFormat, timeFormat } from './sought_games';
import { ActiveGame } from '../store/reducers/lobby_reducer';
import { calculateTotalTime } from '../store/constants';
import { useMountedState } from '../utils/mounted';

const SimultaneousGamesFooter = (props: { onSimultaneous?: () => void }) => {
  const { onSimultaneous } = props;
  const { useState } = useMountedState();

  // Used as a ref callback. React will call setter with null on unmount.
  const [thisElement, setThisElement] = useState<HTMLElement | null>(null);

  // This is the containing div.ant-table-footer.
  const parentElement = React.useMemo(() => thisElement?.parentElement, [
    thisElement,
  ]);

  // The class adds a pointer cursor.
  React.useEffect(() => {
    if (parentElement) {
      parentElement.classList.add('enable-simultaneous-games');
      return () => {
        parentElement.classList.remove('enable-simultaneous-games');
      };
    }
  }, [parentElement]);

  // Mouse-only just like the rest of the app.
  const handleClick = React.useCallback(() => {
    if (onSimultaneous) onSimultaneous();
  }, [onSimultaneous]);

  // Allow clicking outside the text too.
  React.useEffect(() => {
    if (parentElement) {
      parentElement.addEventListener('click', handleClick);
      return () => {
        parentElement.removeEventListener('click', handleClick);
      };
    }
  }, [handleClick, parentElement]);

  return (
    <span
      className="enable-simultaneous-games"
      onClick={handleClick}
      ref={setThisElement}
    >
      Play more games simultaneously (timers will run concurrently)
    </span>
  );
};

type Props = {
  activeGames: ActiveGame[];
  username?: string;
  type?: 'RESUME';
  onSimultaneous?: () => void;
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

  const simultaneousFooter = React.useCallback(
    () => <SimultaneousGamesFooter onSimultaneous={props.onSimultaneous} />,
    [props.onSimultaneous]
  );

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
        footer={props.onSimultaneous ? simultaneousFooter : undefined}
      />
    </>
  );
};
