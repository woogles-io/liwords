import { Form } from 'antd';
import { Store } from 'antd/lib/form/interface';
import Input, { InputRef } from 'rc-input';
import React, { useEffect, useRef } from 'react';
import { useMountedState } from '../utils/mounted';

type Props = {
  rackCallback: (rack: string) => void;
  cancelCallback: () => void;
  currentRack: string;
};

export const RackEditor = (props: Props) => {
  const { useState } = useMountedState();
  const [currentRack, setCurrentRack] = useState(props.currentRack);
  const inputRef = useRef<InputRef>(null);
  const handleKeyDown = (evt: React.KeyboardEvent) => {
    if (evt.key === 'Enter') {
      props.rackCallback(currentRack);
    } else if (evt.key === 'Escape') {
      props.cancelCallback();
    }
  };
  useEffect(() => {
    inputRef.current?.focus({
      cursor: 'all',
    });
  }, [inputRef]);

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
      ref={inputRef}
      placeholder="Enter rack. Use ? for blank"
      className="rack"
      value={currentRack}
      onChange={handleRackEditChange}
      onKeyDown={handleKeyDown}
    />
  );
};
