import { QuestionCircleOutlined } from "@ant-design/icons";
import { Tooltip } from "antd";
import React from "react";

type Props = {
  help: React.ReactElement<Element> | string;
  labelText: string;
};

export const HelptipLabel = (props: Props) => (
  <>
    {props.labelText}
    <Tooltip title={props.help} color="black" /* otherwise it's not opaque */>
      <QuestionCircleOutlined
        className="readable-text-color"
        style={{ marginLeft: 5, color: "cyan" }}
      />
    </Tooltip>
  </>
);
