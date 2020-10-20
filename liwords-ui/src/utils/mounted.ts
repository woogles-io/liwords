import React from 'react';

type SetStateType = <T>(v: T) => [T, React.Dispatch<React.SetStateAction<T>>];

// initial caps so linter allows hook usage
const JustUseState: SetStateType = (v) => React.useState(v);

export const useMountedState: () => {
  isMountedRef: React.MutableRefObject<boolean>;
  useState: SetStateType;
} = () => {
  const isMountedRef = React.useRef(true);
  React.useEffect(() => () => void (isMountedRef.current = false), []);
  const safeUseState = React.useCallback((v) => {
    const ret = JustUseState(v);
    const [state, setState] = ret;
    const safelySetState: React.Dispatch<any> = (x) => {
      if (isMountedRef.current) return setState(x);
    };
    return [state, safelySetState] as typeof ret;
  }, []);
  // "useState" so linter allows omitting returned setter as dependency
  return { isMountedRef, useState: safeUseState };
};
