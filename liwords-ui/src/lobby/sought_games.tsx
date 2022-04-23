import { Popconfirm, Table, Tooltip } from 'antd';
import {
  FilterValue,
  SorterResult,
  TableCurrentDataSource,
  TablePaginationConfig,
} from 'antd/lib/table/interface';
import React, { ReactNode, useCallback } from 'react';
import { FundOutlined, ExportOutlined } from '@ant-design/icons/lib';
import {
  calculateTotalTime,
  challRuleToStr,
  timeCtrlToDisplayName,
  timeToString,
} from '../store/constants';
import {
  SoughtGame,
  matchesRatingFormula,
} from '../store/reducers/lobby_reducer';
import { useMountedState } from '../utils/mounted';
import { PlayerAvatar } from '../shared/player_avatar';
import { DisplayUserFlag } from '../shared/display_flag';
import { RatingBadge } from './rating_badge';
import { VariantIcon } from '../shared/variant_icons';
import { MatchLexiconDisplay } from '../shared/lexicon_display';
import { ProfileUpdate } from '../gen/api/proto/ipc/users_pb';

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

type PlayerProps = {
  userID?: string;
  username: string;
};

export const PlayerDisplay = (props: PlayerProps) => {
  return (
    <div className="player-display">
      <PlayerAvatar
        player={{
          user_id: props.userID,
          nickname: props.username,
        }}
      />
      <span className="player-name">{props.username}</span>
      <DisplayUserFlag uuid={props.userID} />
    </div>
  );
};

type Props = {
  isMatch?: boolean;
  newGame: (seekID: string) => void;
  userID?: string;
  username?: string;
  requests: Array<SoughtGame>;
  ratings?: { [key: string]: ProfileUpdate.Rating };
};
export const SoughtGames = (props: Props) => {
  const { useState } = useMountedState();
  const [cancelVisible, setCancelVisible] = useState(false);
  const [sortByLexicon, setSortByLexicon] = useState(
    localStorage.getItem('sortByLexicon')
  );
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
      dataIndex: 'ratingBadge',
      key: 'rating',
      sorter: (a: SoughtGameTableData, b: SoughtGameTableData) =>
        parseInt(a.rating) - parseInt(b.rating),
    },
    {
      title: 'Words',
      className: 'lexicon',
      dataIndex: 'lexicon',
      key: 'lexicon',
      filters: ['CSW21', 'NWL20', 'ECWL', 'RD28', 'FRA20', 'NSF21'].map(
        (l) => ({
          text: <MatchLexiconDisplay lexiconCode={l} />,
          value: l,
        })
      ),
      defaultFilteredValue: sortByLexicon ? [sortByLexicon] : [],
      filterMultiple: false,
      onFilter: (
        value: string | number | boolean,
        record: SoughtGameTableData
      ) => record.lexiconCode.indexOf(value.toString()) === 0,
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

  const handleChange = useCallback(
    (
      _pagination: TablePaginationConfig,
      filters: Record<string, FilterValue | null>,
      _sorter:
        | SorterResult<SoughtGameTableData>
        | SorterResult<SoughtGameTableData>[],
      extra: TableCurrentDataSource<SoughtGameTableData>
    ) => {
      if (extra.action === 'filter') {
        if (filters.lexicon) {
          if (filters.lexicon.length === 1) {
            const lexicon = filters.lexicon[0] as string;
            if (lexicon !== sortByLexicon) {
              setSortByLexicon(lexicon);
              localStorage.setItem('sortByLexicon', lexicon);
            }
          }
        } else {
          // filter is reset, remove lexicon
          setSortByLexicon(null);
          localStorage.removeItem('sortByLexicon');
        }
      }
    },
    []
  );

  type SoughtGameTableData = {
    seeker: string | ReactNode;
    rating: string;
    ratingBadge?: ReactNode;
    lexicon: ReactNode;
    lexiconCode: string;
    time: string;
    totalTime: number;
    details?: ReactNode;
    outgoing: boolean;
    seekID: string;
  };

  const formatGameData = (games: SoughtGame[]): SoughtGameTableData[] => {
    const gameData: SoughtGameTableData[] = games
      .filter((sg: SoughtGame) => {
        if (sg.seeker === props.username || sg.receiverIsPermanent) {
          // If we are the seeker, or if it's a match request, always show it.
          return true;
        }
        if (props.ratings && matchesRatingFormula(sg, props.ratings)) {
          return true;
        }
        return false;
      })
      .map((sg: SoughtGame): SoughtGameTableData => {
        const getDetails = () => {
          return (
            <>
              <VariantIcon vcode={sg.variant} />{' '}
              {challengeFormat(sg.challengeRule)}
              {sg.rated ? (
                <Tooltip title="Rated">
                  <FundOutlined />
                </Tooltip>
              ) : null}
            </>
          );
        };
        const outgoing = sg.seeker === props.username;
        return {
          seeker: outgoing ? (
            <Popconfirm
              title="Do you want to cancel this game?"
              onConfirm={() => {
                props.newGame(sg.seekID);
                setCancelVisible(false);
              }}
              okText="Yes"
              cancelText="No"
              onCancel={() => {
                console.log('trying', setCancelVisible, cancelVisible);
                setCancelVisible(false);
              }}
              onVisibleChange={(visible) => {
                setCancelVisible(visible);
              }}
              visible={cancelVisible}
            >
              <div>
                <ExportOutlined />
                {` ${sg.receiver?.getDisplayName() || 'Seeking...'}`}
              </div>
            </Popconfirm>
          ) : (
            <PlayerDisplay userID={sg.seekerID} username={sg.seeker} />
          ),
          rating: outgoing ? '' : sg.userRating,
          ratingBadge: outgoing ? null : <RatingBadge rating={sg.userRating} />,
          lexicon: <MatchLexiconDisplay lexiconCode={sg.lexicon} />,
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
          lexiconCode: sg.lexicon,
        };
      });
    return gameData;
  };

  return (
    <>
      {props.isMatch ? <h4>Match requests</h4> : <h4>Available games</h4>}
      <Table
        className={`games ${props.isMatch ? 'match' : 'seek'}`}
        dataSource={formatGameData(props.requests)}
        columns={columns}
        pagination={false}
        rowKey="seekID"
        showSorterTooltip={false}
        onRow={(record) => {
          return {
            onClick: () => {
              if (!record.outgoing) {
                props.newGame(record.seekID);
              } else if (!cancelVisible) {
                setCancelVisible(true);
              }
            },
          };
        }}
        rowClassName={(record) => {
          if (record.outgoing) {
            return 'game-listing outgoing';
          }
          return 'game-listing';
        }}
        onChange={handleChange}
      />
    </>
  );
};
