import React, { ReactNode } from "react";
import { Card } from "antd";
import ReactMarkdown from "react-markdown";

type Props = {
  disclaimer: string;
  logoUrl?: string;
};

function LinkRenderer(props: { href?: string; children?: ReactNode }) {
  return (
    <a href={props.href} target="_blank" rel="noreferrer">
      {props.children}
    </a>
  );
}

export const Disclaimer = React.memo((props: Props) => {
  return (
    <Card className="disclaimer">
      <div>
        <ReactMarkdown components={{ a: LinkRenderer }}>
          {props.disclaimer}
        </ReactMarkdown>
      </div>
      {props.logoUrl && (
        <div className="logo-container">
          <img className="club-logo" src={props.logoUrl} alt="logo" />
        </div>
      )}
    </Card>
  );
});
