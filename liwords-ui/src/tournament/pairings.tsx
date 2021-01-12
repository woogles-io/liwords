import React, { ReactNode } from 'react';
import { useTournamentStoreContext } from '../store/store';
import { Table, Tag } from 'antd';
import { Division, SinglePairing } from '../store/reducers/tournament_reducer';
import { TournamentGameResult } from '../gen/api/proto/realtime/realtime_pb';

const usernameFromPlayerEntry = (p: string) =>
  p.split(':').length > 0 ? p.split(':')[1] : 'Unknown player';

const pairingsForRound = (
  round: number,
  division: Division
): Array<SinglePairing> => {
  const m = new Set<string>();
  const n = new Array<string>();
  const numPlayers = division.players.length;
  // round in this case is 1-indexed (it is the displayed round).
  for (let idx = (round - 1) * numPlayers; idx < round * numPlayers; idx++) {
    const key = division.roundInfo[idx];
    if (key && !m.has(key)) {
      n.push(key);
      m.add(key);
    }
  }
  return n.map((key) => division.pairingMap[key]);
};

type Props = {
  selectedDivision?: string;
  selectedRound: number;
};

type PairingTableData = {
  players: ReactNode;
  // ratings: ReactNode;
  //wl: ReactNode;
  //scores: ReactNode;
  //actions: ReactNode;
};
export const Pairings = (props: Props) => {
  const { tournamentContext } = useTournamentStoreContext();
  const { divisions } = tournamentContext;
  const formatPairingsData = (
    division: Division,
    round: number
  ): PairingTableData[] => {
    if (!division) {
      return new Array<PairingTableData>();
    }
    const pairings = pairingsForRound(props.selectedRound, division);
    const pairingsData = pairings.map(
      (pairing: SinglePairing): PairingTableData => {
        const playerNames = pairing.players.map(usernameFromPlayerEntry);
        const isBye = pairing.outcomes[0] === TournamentGameResult.BYE;
        const players = isBye ? (
          <div>
            <p>
              {playerNames[0]}
              <Tag className="ant-tag-bye">Bye</Tag>
            </p>
          </div>
        ) : (
          <div>
            {playerNames.map((playerName) => (
              <p key={playerName}>{playerName}</p>
            ))}
          </div>
        );
        return {
          players,
        };
      }
    );
    return pairingsData;
  };

  const columns = [
    {
      title: 'Players',
      dataIndex: 'players',
      key: 'players',
    },
    /*    {
      title: 'Ratings',
      dataIndex: 'ratings',
      key: 'ratings',
    },
    {
      title: 'W/L',
      dataIndex: 'wl',
      key: 'wl',
    },
    {
      title: 'Scores',
      dataIndex: 'scores',
      key: 'scores',
    },
    {
      title: '',
      dataIndex: 'actions',
      key: 'actions',
    },
    */
  ];
  if (!props.selectedDivision) {
    return null;
  }
  return (
    <Table
      className="pairings"
      columns={columns}
      pagination={false}
      rowKey="pairing"
      locale={{
        emptyText: 'The pairings are not yet available for this round.',
      }}
      dataSource={formatPairingsData(
        divisions[props.selectedDivision],
        props.selectedRound
      )}
    />
  );
};
