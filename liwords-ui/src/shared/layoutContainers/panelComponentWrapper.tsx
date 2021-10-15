// Wraps panels with a key that we would like to be accessible from the menus in mobile
import React from 'react';
import { useSideMenuContext } from './menu';

type Props = {
  className?: string;
  children?: React.ReactNode;
  panelKey: string;
};

export const PanelComponentWrapper = React.memo((props: Props) => {
  const { activePanelKey } = useSideMenuContext();
  const calculatedClassName = `panel ${props.className || ''}${
    activePanelKey === props.panelKey ? ' in' : ''
  }`;
  return (
    <div className={calculatedClassName} data-panelKey={props.panelKey}>
      {props.children}
    </div>
  );
});
