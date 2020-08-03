import React, { useState } from 'react';
import { Form, Radio, InputNumber, Switch, Tag } from 'antd';

import { Store } from 'antd/lib/form/interface';
import { ChallengeRule } from '../gen/macondo/api/proto/macondo/macondo_pb';
import { timeCtrlToDisplayName } from '../store/constants';

export type seekPropVals = { [val: string]: string | number | boolean };

type Props = {
  vals: seekPropVals;
  onChange: (arg0: seekPropVals) => void;
  loggedIn: boolean;
};

export const SeekForm = (props: Props) => {
  const [timectrl, setTimectrl] = useState('Rapid');
  const [ttag, setTtag] = useState('gold');
  const onFormChange = (val: Store, allvals: Store) => {
    props.onChange(allvals);
    const [tc, tt] = timeCtrlToDisplayName(
      Math.round((allvals.initialtime as number) * 60)
    );
    setTimectrl(tc);
    setTtag(tt);
  };

  return (
    <Form onValuesChange={onFormChange} initialValues={props.vals}>
      <Form.Item label="Lexicon" name="lexicon">
        <Radio.Group>
          <Radio.Button value="CSW19">CSW19</Radio.Button>
          <Radio.Button value="NWL18">NWL18</Radio.Button>
        </Radio.Group>
      </Form.Item>
      <Form.Item label="Challenge Rule" name="challengerule">
        <Radio.Group>
          <Radio.Button value={ChallengeRule.FIVE_POINT}>5-pt</Radio.Button>
          <Radio.Button value={ChallengeRule.TEN_POINT}>10-pt</Radio.Button>
          <Radio.Button value={ChallengeRule.DOUBLE}>Double</Radio.Button>
          <Radio.Button value={ChallengeRule.SINGLE}>Single</Radio.Button>
          <Radio.Button value={ChallengeRule.VOID}>Void</Radio.Button>
        </Radio.Group>
      </Form.Item>
      <Form.Item label="Initial Time (minutes)" name="initialtime">
        <InputNumber />
      </Form.Item>
      <Form.Item label="Max Overtime (minutes)" name="maxovertime">
        <InputNumber max={5} min={0} />
      </Form.Item>
      <Form.Item label="Rated" name="rated" valuePropName="checked">
        <Switch />
      </Form.Item>
      Time Control: <Tag color={ttag}>{timectrl}</Tag>
    </Form>
  );
};
