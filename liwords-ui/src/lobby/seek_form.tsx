import React from 'react';
import { Form, Radio, InputNumber } from 'antd';

import { Store } from 'antd/lib/form/interface';
import { ChallengeRule } from '../gen/macondo/api/proto/macondo/macondo_pb';

export type seekPropVals = { [val: string]: string | number };

type Props = {
  vals: seekPropVals;
  onChange: (arg0: seekPropVals) => void;
};

export const SeekForm = (props: Props) => {
  const onFormChange = (val: Store, allvals: Store) => {
    props.onChange(allvals);
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
    </Form>
  );
};
