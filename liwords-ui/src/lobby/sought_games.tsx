/* eslint-disable jsx-a11y/click-events-have-key-events */
/* eslint-disable jsx-a11y/no-noninteractive-element-interactions */
import { Table, Tooltip } from 'antd';
import React, { ReactNode } from 'react';
import { FundOutlined, ExportOutlined } from '@ant-design/icons/lib';
import {
  calculateTotalTime,
  challRuleToStr,
  timeCtrlToDisplayName,
  initialTimeLabel,
  timeToString,
} from '../store/constants';
import { SoughtGame } from '../store/reducers/lobby_reducer';

type SoughtGameProps = {
  game: SoughtGame;
  newGame: (seekID: string) => void;
  userID?: string;
  username?: string;
  requests: SoughtGame[];
};

export const timeFormat = (
  initialTimeSecs: number,
  incrementSecs: number,
  maxOvertime: number
): string => {
  const label = timeCtrlToDisplayName(
    initialTimeSecs,
    incrementSecs,
    maxOvertime
  )[0];

  return `${label} ${timeToString(
    initialTimeSecs,
    incrementSecs,
    maxOvertime
  )}`;
};

export const challengeFormat = (cr: number) => {
  return (
    <span className={`challenge-rule mode_${challRuleToStr(cr)}`}>
      {challRuleToStr(cr)}
    </span>
  );
};

type Props = {
  isMatch?: boolean;
  newGame: (seekID: string) => void;
  userID?: string;
  username?: string;
  requests: Array<SoughtGame>;
};

export const SoughtGames = (props: Props) => {
  // @ts-ignore
  // @ts-ignore
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
      sorter: (a: SoughtGameTableData, b: SoughtGameTableData) =>
        parseInt(a.rating) - parseInt(b.rating),
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
          text: 'NWL18',
          value: 'NWL18',
        },
      ],
      filterMultiple: false,
      onFilter: (
        value: string | number | boolean,
        record: SoughtGameTableData
      ) => record.lexicon.indexOf(value.toString()) === 0,
    },
    {
      title: 'Time',
      className: 'time',
      dataIndex: 'time',
      key: 'time',
      sorter: (a: SoughtGameTableData, b: SoughtGameTableData) =>
        a.totalTime - b.totalTime,
    },
    {
      title: 'Details',
      className: 'details',
      dataIndex: 'details',
      key: 'details',
    },
  ];

  type SoughtGameTableData = {
    seeker: string | ReactNode;
    rating: string;
    lexicon: string;
    time: string;
    totalTime: number;
    details?: ReactNode;
    outgoing: boolean;
    seekID: string;
  };

  const formatGameData = (games: SoughtGame[]): SoughtGameTableData[] => {
    const gameData: SoughtGameTableData[] = games.map(
      (sg: SoughtGame): SoughtGameTableData => {
        const getDetails = () => {
          return (
            <>
              {challengeFormat(sg.challengeRule)}
              {sg.rated ? (
                <Tooltip title="Rated">
                  <FundOutlined />
                </Tooltip>
              ) : null}
            </>
          );
        };
        let outgoing = !!sg.receiver.getUserId();
        if (sg.receiver.getUserId() === props.userID) {
          outgoing = false;
        }
        return {
          seeker: outgoing ? (
            <>
              {sg.receiver.getDisplayName()}
              <ExportOutlined />
            </>
          ) : (
            sg.seeker
          ),
          rating: sg.userRating,
          lexicon: sg.lexicon,
          time: timeFormat(
            sg.initialTimeSecs,
            sg.incrementSecs,
            sg.maxOvertimeMinutes
          ),
          totalTime: calculateTotalTime(
            sg.initialTimeSecs,
            sg.incrementSecs,
            sg.maxOvertimeMinutes
          ),
          details: getDetails(),
          outgoing,
          seekID: sg.seekID,
        };
      }
    );
    return gameData;
  };

  return (
    <>
      {props.isMatch ? <h4>Match Requests</h4> : <h4>Available Games</h4>}
      <Table
        className={`games ${props.isMatch ? 'match' : 'seek'}`}
        dataSource={formatGameData(props.requests)}
        columns={columns}
        pagination={{
          hideOnSinglePage: true,
        }}
        rowKey="seekID"
        showSorterTooltip={false}
        onRow={(record) => {
          return {
            onClick: (event) => {
              props.newGame(record.seekID);
            },
          };
        }}
        rowClassName={(record) => {
          if (record.outgoing) {
            return 'game-listing outgoing';
          }
          return 'game-listing';
        }}
      />
    </>
  );
};
