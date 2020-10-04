import React from 'react';

type Props = {
  score: number | undefined;
  horizontal: boolean | undefined;
};

const TentativeScore = (props: Props) => {
  if (props.score != null) {
    return (
      <p
        className={`tentative-score ${
          props.horizontal
            ? 'tentative-score-horizontal'
            : 'tentative-score-vertical'
        }`}
      >
          {props.score}
      </p>
    );
  } else {
      return null;
  }

};

export default TentativeScore;
