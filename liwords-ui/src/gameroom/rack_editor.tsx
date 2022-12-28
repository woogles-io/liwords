import { Form } from 'antd';
import { Store } from 'antd/lib/form/interface';
import Input from 'rc-input';
import React from 'react';
import { useMountedState } from '../utils/mounted';

type Props = {
  rackCallback: (rack: string) => void;
};

export const RackEditor = (props: Props) => {
  const { useState } = useMountedState();
  const [currentRack, setCurrentRack] = useState('');

  const onFinishedEditingRack = (vals: Store) => {
    props.rackCallback(currentRack);
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
    <Form onFinish={onFinishedEditingRack}>
      <Form.Item name="rack">
        <Input
          placeholder="Enter rack. Use ? for blank"
          // antd doesn't seem to have controlled forms, so this
          // doesn't work as expected??
          value={currentRack}
          onChange={handleRackEditChange}
          autoFocus
        />
      </Form.Item>
    </Form>
  );
};
