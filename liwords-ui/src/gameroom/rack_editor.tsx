import Input, { InputRef } from 'rc-input';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { MachineWord } from '../utils/cwgame/common';
import {
  Alphabet,
  machineWordToRunes,
  runesToMachineWord,
} from '../constants/alphabets';

type Props = {
  rackCallback: (rack: MachineWord) => void;
  cancelCallback: () => void;
  currentRack: MachineWord;
  alphabet: Alphabet;
};

const MaxRackLength = 7;

export const RackEditor = (props: Props) => {
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
  const curRackStr = useMemo(
    () => machineWordToRunes(currentRack, props.alphabet, false),
    [currentRack, props.alphabet]
  );

  const handleRackEditChange = (evt: React.ChangeEvent<HTMLInputElement>) => {
    // strip out any spaces, fix max length, etc.
    const raw = evt.target.value;
    let out = raw;
    out = out.toLocaleUpperCase();
    const onlyLettersAndBlank = out.match(/[\p{Letter}·\?]/gu);
    let curRack = runesToMachineWord(
      onlyLettersAndBlank?.join('') ?? '',
      props.alphabet
    );
    if (curRack.length > MaxRackLength) {
      curRack = curRack.slice(0, MaxRackLength);
    }
    setCurrentRack(curRack);
  };

  return (
    <Input
      ref={inputRef}
      placeholder="Enter rack. Use ? for blank"
      className="rack"
      value={curRackStr}
      onChange={handleRackEditChange}
      onKeyDown={handleKeyDown}
    />
  );
};
