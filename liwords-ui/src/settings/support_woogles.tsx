import React from "react";
import woogles from "../assets/woogles.png";
import { Donate } from "../donate";

type Props = {
  handleContribute?: () => void;
};

export const Support = React.memo((props: Props) => {
  return (
    <>
      <h3>Help us keep Woogles.io going!</h3>
      <div className="support-woogles">
        <img src={woogles} className="woogles" alt="Woogles" />
        <div className="right-column">
          <Donate />
        </div>
      </div>
    </>
  );
});
