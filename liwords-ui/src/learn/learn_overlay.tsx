import React, { useContext, useEffect, useMemo } from 'react';
import { useMountedState } from '../utils/mounted';

export const LearnContext = React.createContext<{
  gridDim: number;
  setGridDim: React.Dispatch<React.SetStateAction<number>>;
  learnLayout: Array<Array<string>>;
  setLearnLayout: React.Dispatch<React.SetStateAction<Array<Array<string>>>>;
}>({
  gridDim: 0,
  setGridDim: ((g: number) => {}) as React.Dispatch<
    React.SetStateAction<number>
  >,
  learnLayout: [],
  setLearnLayout: ((l: Array<Array<string>>) => {}) as React.Dispatch<
    React.SetStateAction<Array<Array<string>>>
  >,
});

export const LearnContextProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  const { useState } = useMountedState();
  const [gridDim, setGridDim] = useState(0);
  const [learnLayout, setLearnLayout] = useState<Array<Array<string>>>([]);
  const contextValue = useMemo(
    () => ({ gridDim, setGridDim, learnLayout, setLearnLayout }),
    [gridDim, setGridDim, learnLayout, setLearnLayout]
  );
  return <LearnContext.Provider value={contextValue} children={children} />;
};

export enum LearnSpaceType {
  Normal = ' ',
  Faded = 'f',
  Highlighted = 'h',
}

const getLearnSpaceClassName = (l: LearnSpaceType) => {
  switch (l) {
    case LearnSpaceType.Faded:
      return 'faded';
    case LearnSpaceType.Highlighted:
      return 'highlighted';
  }
  return '';
};

type LearnSpaceProps = {
  value: LearnSpaceType;
};

const LearnSpace = React.memo((props: LearnSpaceProps) => {
  const { value } = props;
  return (
    <div
      className={`learn-space board-space ${getLearnSpaceClassName(value)}`}
    />
  );
});

type LearnOverlayProps = {
  gridDim: number;
};

export const generateEmptyLearnLayout = (
  size: number,
  filler: string = LearnSpaceType.Normal
) => {
  return new Array(size).fill(null).map(() => new Array(size).fill(filler));
};

export const LearnOverlay = React.memo((props: LearnOverlayProps) => {
  const { setGridDim, learnLayout, setLearnLayout } = useContext(LearnContext);

  useEffect(() => {
    setGridDim(props.gridDim);
    setLearnLayout(
      generateEmptyLearnLayout(props.gridDim, LearnSpaceType.Normal)
    );
  }, [props.gridDim, setGridDim, setLearnLayout]);

  const renderSpaces = useMemo(() => {
    const flattened = learnLayout.flat();
    return flattened.map((s: string, i: number) => (
      <LearnSpace key={`lo-${i}`} value={s as LearnSpaceType} />
    ));
  }, [learnLayout]);

  return <div className="learn-spaces board-spaces">{renderSpaces}</div>;
});
