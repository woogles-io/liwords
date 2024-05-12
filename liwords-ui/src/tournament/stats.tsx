import { useMemo } from 'react';
import {
  TournamentState,
  Division,
} from '../store/reducers/tournament_reducer';

type Props = {
  tournamentContext: TournamentState;
  selectedDivision: string;
};

type DivStatsProps = {
  division: Division;
};

const DivisionStats = (props: DivStatsProps) => {
  return <div>LAH</div>;
};

export const TournamentStats = (props: Props) => {
  const td = useMemo(() => {
    const division = props.tournamentContext.divisions[props.selectedDivision];
    return <DivisionStats division={division} />;
  }, [props.tournamentContext.divisions[props.selectedDivision]]);

  return <div>{td}</div>;
};
