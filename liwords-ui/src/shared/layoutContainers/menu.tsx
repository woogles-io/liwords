import React, { createContext, useContext, useMemo } from 'react';
import { useMountedState } from '../../utils/mounted';

const SideMenuContext = createContext<{
  activePanelKey?: string;
  defaultActivePanelKey?: string;
  isOpen: boolean;
  setActivePanelKey: React.Dispatch<React.SetStateAction<string | undefined>>;
  setIsOpen: React.Dispatch<React.SetStateAction<boolean>>;
}>({
  isOpen: false,
  defaultActivePanelKey: undefined,
  activePanelKey: undefined,
  setActivePanelKey: ((x: string | undefined) => {}) as React.Dispatch<
    React.SetStateAction<string | undefined>
  >,
  setIsOpen: ((x: boolean) => {}) as React.Dispatch<
    React.SetStateAction<boolean>
  >,
});

export const SideMenuContextProvider = ({
  children,
  defaultActivePanelKey,
}: {
  children: React.ReactNode;
  defaultActivePanelKey?: string;
}) => {
  const { useState } = useMountedState();
  const [isOpen, setIsOpen] = useState(false);
  const [activePanelKey, setActivePanelKey] = useState<string | undefined>(
    defaultActivePanelKey
  );
  const contextValue = useMemo(
    () => ({
      isOpen,
      setIsOpen,
      activePanelKey,
      setActivePanelKey,
      defaultActivePanelKey,
    }),
    [
      isOpen,
      setIsOpen,
      activePanelKey,
      setActivePanelKey,
      defaultActivePanelKey,
    ]
  );
  return <SideMenuContext.Provider value={contextValue} children={children} />;
};

type Props = {
  children?: React.ReactNode;
  className?: string;
};

export const SideMenu = React.memo((props: Props) => {
  const { className, children } = props;
  const { isOpen } = useSideMenuContext();
  const calculatedClassName = `menu-container ${className || ''}${
    isOpen ? ' open' : ''
  }`;
  return (
    <>
      <aside className={calculatedClassName}>Menu Here</aside>
      {children}
    </>
  );
});

export const useSideMenuContext = () => {
  return useContext(SideMenuContext);
};
