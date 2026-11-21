import React from "react";
import { useLocalStorageBool } from "../utils/use_local_storage";

type Props = {
  score: number | undefined;
  horizontal: boolean | undefined;
};

const TentativeScore = (props: Props) =>
  props.score == null ? null : (
    <ShownScore score={props.score} horizontal={props.horizontal} />
  );

// The score hides while its setting says so, and follows the setting as it
// changes, in this tab or another, as the play's question does. Only the tile
// holding a score reads it: the board's other tiles have none.
const ShownScore = (props: {
  score: number;
  horizontal: boolean | undefined;
}) => {
  const [hidePool] = useLocalStorageBool("hidePool");
  if (hidePool) {
    return null;
  }
  return (
    <p
      className={`tentative-score ${
        props.horizontal
          ? "tentative-score-horizontal"
          : "tentative-score-vertical"
      }`}
    >
      {props.score}
    </p>
  );
};

export default TentativeScore;
