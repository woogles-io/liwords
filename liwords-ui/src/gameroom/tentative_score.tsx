import React from "react";

// TODO: Preferences need a better solution.
const prefsCache: { hidePool: boolean | undefined } = { hidePool: undefined };

const prefsCheck = (key: "hidePool") => {
  let cachedToggle = prefsCache[key];
  if (cachedToggle === undefined) {
    // localStorage is synchronous, try not to do it too often.
    // Not sure if this is helpful...
    cachedToggle = localStorage.getItem(key) === "true";
    prefsCache[key] = cachedToggle;
    setTimeout(() => {
      prefsCache[key] = undefined;
    }, 1000);
  }
  return cachedToggle;
};

type Props = {
  score: number | undefined;
  horizontal: boolean | undefined;
};

const TentativeScore = (props: Props) => {
  if (props.score != null && !prefsCheck("hidePool")) {
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
  } else {
    return null;
  }
};

export default TentativeScore;
