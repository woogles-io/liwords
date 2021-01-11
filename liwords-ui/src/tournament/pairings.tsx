import React, { ReactNode } from 'react';
import { useTournamentStoreContext } from '../store/store';
import { Table, Tag } from 'antd';
import { Division, SinglePairing } from '../store/reducers/tournament_reducer';

const usernameFromPlayerEntry = (p: string) =>
  p.split(':').length > 0 ? p.split(':')[1] : 'Unknown player';

// Parses the 0 based Round number from the key
const roundNumberFromPairingKey = (pairing: string) =>
  parseInt(pairing.split(':')[0]);

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
    const currentPairingKeys = Object.keys(division.roundInfo).filter(
      (p) => roundNumberFromPairingKey(p) === round - 1
    );
    const divisionPairings = currentPairingKeys.map(
      (key) => division.roundInfo[key]
    );
    console.log(props.selectedRound);
    // This deduping assumes every player plays only one game per round,
    // will need backend to dedup or give each actual game an id if we change that
    const pairings = divisionPairings.reduce(
      (acc: SinglePairing[], p: SinglePairing) => {
        //if acc contains an entry that contains the first player in this pairing, don't add it
        if (acc.some((acPairing) => acPairing.players.includes(p.players[0]))) {
          return acc;
        }
        return acc.concat(p);
      },
      new Array<SinglePairing>()
    );
    const pairingsData = pairings.map(
      (pairing: SinglePairing): PairingTableData => {
        const playerNames = pairing.players.map(usernameFromPlayerEntry);
        const isBye = Array.from(new Set(playerNames)).length === 1;
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
              <p>{playerName}</p>
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
