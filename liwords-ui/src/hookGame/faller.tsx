import React, { useMemo } from 'react';
import { useMountedState } from '../utils/mounted';

export const Faller = React.memo(() => {
  const { useState } = useMountedState();

  const renderUnit = useMemo(() => {
    return <div>s</div>;
  }, []);

  return <div className="faller">{renderUnit}</div>;
});
