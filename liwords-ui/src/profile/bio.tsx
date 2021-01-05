import React, { useCallback, useEffect } from 'react';
import ReactMarkdown from 'react-markdown';
import { Card } from 'antd';

type BioProps = {
  bio: string;
};

export const BioCard = React.memo((props: BioProps) => {
  return (
    <Card title="Bio">
      <ReactMarkdown>{props.bio}</ReactMarkdown>
    </Card>
  );
});

