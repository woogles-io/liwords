import React from 'react';
import { Card } from 'antd';
import ReactMarkdown from 'react-markdown';

type Props = {
  disclaimer: string;
  logoUrl?: string;
};
export const Disclaimer = React.memo((props: Props) => {
  return (
    <Card className="disclaimer">
      <div>
        <ReactMarkdown linkTarget="_blank">{props.disclaimer}</ReactMarkdown>
      </div>
      {props.logoUrl && (
        <div className="logo-container">
          <img className="club-logo" src={props.logoUrl} alt="logo" />
        </div>
      )}
    </Card>
  );
});
