import { Form } from 'antd';
import { Store } from 'antd/lib/form/interface';
import Input from 'rc-input';
import React from 'react';
import { useMountedState } from '../utils/mounted';

type Props = {
  rackCallback: (rack: string) => void;
  cancelCallback: () => void;
  currentRack: string;
};

export const RackEditor = (props: Props) => {
  const { useState } = useMountedState();
  const [currentRack, setCurrentRack] = useState(props.currentRack);

  const handleKeyDown = (evt: React.KeyboardEvent) => {
    if (evt.key === 'Enter') {
      props.rackCallback(currentRack);
    } else if (evt.key === 'Escape') {
      props.cancelCallback();
    }
  };

  const handleRackEditChange = (evt: React.ChangeEvent<HTMLInputElement>) => {
    // strip out any spaces, fix max length, etc.
    const raw = evt.target.value;
    let out = raw;
    if (out.length > 7) {
      out = out.substring(0, 7);
    }
    out = out.toLocaleUpperCase();
    const onlyLettersAndBlank = out.match(/[\p{Letter}\?]/gu);
    setCurrentRack(onlyLettersAndBlank?.join('') ?? '');
  };

  return (
    <Input
      placeholder="Enter rack. Use ? for blank"
      value={currentRack}
      onChange={handleRackEditChange}
      onKeyDown={handleKeyDown}
      autoFocus
    />
  );
};
