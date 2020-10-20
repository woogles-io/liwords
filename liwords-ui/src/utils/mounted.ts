import React from 'react';

// Instead of React.useState, use the useState returned from useMountedState.

// This useState returns a setState that does nothing when the containing
// component is already unmounted, avoiding any "setState warnings".

// https://reactjs.org/blog/2015/12/16/ismounted-antipattern.html

// While this is normally an antipattern, it is an intended pattern in this
// application because of how we reset the whole store from time to time.

// initial caps so linter allows hook usage
const JustUseState = React.useState;

export const useMountedState: () => {
  useState: <S>(
    initialState: S | (() => S)
  ) => [S, React.Dispatch<React.SetStateAction<S>>];
} = () => {
  const isMountedRef = React.useRef(true);
  React.useEffect(() => () => void (isMountedRef.current = false), []);
  const safeUseState = React.useCallback((initialState) => {
    const ret = JustUseState(initialState);
    const [state, setState] = ret;
    const safelySetState: React.Dispatch<any> = (value) => {
      if (isMountedRef.current) return setState(value);
    };
    return [state, safelySetState] as typeof ret;
  }, []);
  // "useState" so linter allows omitting returned setter as dependency
  return { useState: safeUseState };
};
