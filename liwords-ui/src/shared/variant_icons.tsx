import { CloudTwoTone } from '@ant-design/icons';
import { Tooltip } from 'antd';
import React from 'react';

type Props = {
  vcode: string;
  withName?: boolean;
};

export const VariantIcon = (props: Props) => {
  switch (props.vcode) {
    // classic has no icon yet
    case '':
    case 'classic':
      if (props.withName) {
        return <>{'Classic'}</>;
      }
      break;
    case 'wordsmog':
      if (props.withName) {
        return (
          <>
            <CloudTwoTone /> WordSmog
          </>
        );
      }
      return (
        <Tooltip title="WordSmog">
          <CloudTwoTone />
        </Tooltip>
      );
  }
  return null;
};
