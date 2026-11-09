import React from "react";
import { Tag } from "antd";
import { titleStyle } from "./title_styles";

type TitleTagProps = {
  // the title's abbreviation, which picks its colors
  abbreviation: string;
  children: React.ReactNode;
  style?: React.CSSProperties;
};

// A title as a tag, in the fill and border of the circle by a username. The
// text keeps the color App.scss gives every tag, dark in either mode, which
// reads on every title color (the circle's yellow text on its pale yellow
// would not).
export const TitleTag = (props: TitleTagProps) => {
  const styles = titleStyle(props.abbreviation);
  return (
    <Tag
      style={{
        backgroundColor: styles.background,
        borderColor: styles.border,
        ...props.style,
      }}
    >
      {props.children}
    </Tag>
  );
};
