import React from "react";
import { scoreSide } from "../utils/score_pair";

import "./score_pair.scss";

type Props = {
  first: number;
  second: number;
};

// Two scores lined up on the hyphen between them: each in a box three digits
// wide, the first at the right of its box and the second at the left, so a
// column of them keeps its hyphens in a line.
export const AlignedScorePair = (props: Props) => (
  <span className="aligned-score-pair">
    <span className="aligned-score-pair-first">{scoreSide(props.first)}</span>
    <span className="aligned-score-pair-hyphen">-</span>
    <span className="aligned-score-pair-second">{scoreSide(props.second)}</span>
  </span>
);
