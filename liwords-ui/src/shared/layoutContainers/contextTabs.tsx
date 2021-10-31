import React, { ReactNode } from 'react';
import { Affix, Button } from 'antd';
import { useSideMenuContext } from './menu';

type ContextTabProps = {
  panelKey: string;
  label: string;
};

export const ContextTab = React.memo((props: ContextTabProps) => {
  const { label, panelKey } = props;
  const { activePanelKey, setActivePanelKey } = useSideMenuContext();
  return (
    <Button
      className={
        panelKey === activePanelKey ? 'context-tab selected' : 'context-tab'
      }
      shape="round"
      onClick={() => {
        setActivePanelKey(panelKey);
      }}
    >
      {label}
    </Button>
  );
});

type TabContainerProps = {
  children?: ReactNode;
  className?: string;
};
export const ContextTabs = React.memo((props: TabContainerProps) => {
  return (
    <Affix className={`mobile-tabs ${props.className || ''}`}>
      {props.children}
    </Affix>
  );
});
