import { Button } from "antd";
import React, { useState } from "react";

type Props = {
  sendReady: () => boolean;
};

export const ReadyButton = (props: Props) => {
  const [disabled, setDisabled] = useState(false);
  return (
    <Button
      className="primary"
      onClick={() => {
        const success = props.sendReady();
        if (success) {
          setDisabled(true);
        }
      }}
      disabled={disabled}
    >
      I'm ready
    </Button>
  );
};
