import { Button } from "antd";
import React, { useState } from "react";

type Props = {
  sendReady: () => void;
};

export const ReadyButton = (props: Props) => {
  const [disabled, setDisabled] = useState(false);
  return (
    <Button
      className="primary"
      onClick={() => {
        props.sendReady();
        setDisabled(true);
      }}
      disabled={disabled}
    >
      I'm ready
    </Button>
  );
};
