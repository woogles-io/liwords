import React from 'react';

type Props = {
  score?: number;
};

const TentativeScore = (props: Props) => {
  if (props.score) {
    return (
      <p className="tentative-score">
          {props.score}
      </p>
    );
  } else {
      return null;
  }

};

export default TentativeScore;
