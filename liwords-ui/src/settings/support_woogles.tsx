import React from "react";
import kofiBingo from "../assets/kofiBingo.webp";
import { Donate } from "../donate";

type Props = {
  handleContribute?: () => void;
};

export const Support = React.memo((props: Props) => {
  return (
    <>
      <h3>Help us keep KofiBingo.io going!</h3>
      <div className="support-woogles">
        <img src={kofiBingo} className="woogles" alt="Kofi Bingo" />
        <div className="right-column">
          <Donate />
        </div>
      </div>
    </>
  );
});
