import React, { ReactNode } from 'react';
import { Affix, Button } from 'antd';
import { useSideMenuContext } from './menu';

type ContextTabProps = {
  panelKey: string;
  label: string;
  desktop?: boolean;
};

export const ContextTab = React.memo((props: ContextTabProps) => {
  const { desktop, label, panelKey } = props;
  const { activePanelKey, setActivePanelKey } = useSideMenuContext();
  let calculatedClassName =
    panelKey === activePanelKey ? 'context-tab selected' : 'context-tab';
  if (desktop) {
    calculatedClassName += ' desktop';
  }
  return (
    <Button
      className={calculatedClassName}
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
    <Affix className={`context-tabs ${props.className || ''}`}>
      {props.children}
    </Affix>
  );
});
