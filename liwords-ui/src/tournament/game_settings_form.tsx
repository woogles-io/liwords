// This form has a lot in common with the form in seek_form.tsx, but it is simpler.
import { FormInstance } from 'antd/lib/form';
import { Form, InputNumber, Radio, Select, Slider, Switch, Tag } from 'antd';
import { Store } from 'rc-field-form/lib/interface';
import React from 'react';
import { ChallengeRule } from '../gen/macondo/api/proto/macondo/macondo_pb';
import {
  initTimeDiscreteScale,
  timeCtrlToDisplayName,
  timeScaleToNum,
} from '../store/constants';
import { useMountedState } from '../utils/mounted';
import { ChallengeRulesFormItem } from '../lobby/challenge_rules_form_item';
import { VariantIcon } from '../shared/variant_icons';

type Props = {
  form?: FormInstance<any>;
};

const otLabel = 'Overtime';
const incLabel = 'Increment';
const otUnitLabel = (
  <>
    minutes <span className="help">(10 point penalty each extra minute)</span>
  </>
);
const incUnitLabel = (
  <>
    seconds <span className="help">(Extra seconds are awarded each turn)</span>
  </>
);

const initTimeFormatter = (val?: number) => {
  return initTimeDiscreteScale[val!];
};

export const SettingsForm = (props: Props) => {
  const { useState } = useMountedState();

  const initialValues = {
    lexicon: 'CSW19',
    variant: 'classic',
    challengerule: ChallengeRule.FIVE_POINT,
    initialtime: 22, // Note this isn't minutes, but the slider position.
    rated: true,
    extratime: 1,
    incOrOT: 'overtime',
  };

  const [itc, itt] = timeCtrlToDisplayName(
    timeScaleToNum(initTimeDiscreteScale[initialValues.initialtime]) * 60,
    initialValues.incOrOT === 'increment'
      ? Math.round(initialValues.extratime as number)
      : 0,
    initialValues.incOrOT === 'increment'
      ? 0
      : Math.round(initialValues.extratime as number)
  );
  const [timectrl, setTimectrl] = useState(itc);
  const [ttag, setTtag] = useState(itt);
  const [timeSetting, setTimeSetting] = useState(
    initialValues.incOrOT === 'overtime' ? otLabel : incLabel
  );
  const [extraTimeLabel, setExtraTimeLabel] = useState(
    initialValues.incOrOT === 'overtime' ? otUnitLabel : incUnitLabel
  );
  const [maxTimeSetting, setMaxTimeSetting] = useState(
    initialValues.incOrOT === 'overtime' ? 10 : 60
  );

  const onFormChange = (val: Store, allvals: Store) => {
    if (allvals.incOrOT === 'increment') {
      setTimeSetting(incLabel);
      setMaxTimeSetting(60);
      setExtraTimeLabel(incUnitLabel);
    } else {
      setTimeSetting(otLabel);
      setMaxTimeSetting(10);
      setExtraTimeLabel(otUnitLabel);
    }
    const [tc, tt] = timeCtrlToDisplayName(
      timeScaleToNum(initTimeDiscreteScale[allvals.initialtime]) * 60,
      allvals.incOrOT === 'increment'
        ? Math.round(allvals.extratime as number)
        : 0,
      allvals.incOrOT === 'increment'
        ? 0
        : Math.round(allvals.extratime as number)
    );
    setTimectrl(tc);
    setTtag(tt);
  };

  const validateMessages = {
    required: 'This field is required.',
  };

  return (
    <Form
      onValuesChange={onFormChange}
      initialValues={initialValues}
      labelCol={{ span: 6 }}
      wrapperCol={{ span: 24 }}
      layout="horizontal"
      validateMessages={validateMessages}
      name="gameSettingsForm"
      form={props.form}
    >
      <Form.Item
        label="Dictionary"
        name="lexicon"
        rules={[
          {
            required: true,
          },
        ]}
      >
        <Select>
          <Select.Option value="CSW19">CSW 19 (World English)</Select.Option>
          <Select.Option value="NWL20">
            NWL 20 (North American English)
          </Select.Option>
          <Select.Option value="ECWL">English Common Word List</Select.Option>
          <Select.Option value="NSWL20">
            NSWL 20 (NASPA School Word List)
          </Select.Option>
          <Select.Option value="NWL18">NWL 18 (Obsolete)</Select.Option>
        </Select>
      </Form.Item>

      <Form.Item label="Game type" name="variant" rules={[{ required: true }]}>
        <Select>
          <Select.Option value="classic">Classic</Select.Option>
          <Select.Option value="wordsmog">
            <VariantIcon vcode="wordsmog" withName />
          </Select.Option>
        </Select>
      </Form.Item>

      <ChallengeRulesFormItem disabled={false} />

      <Form.Item
        className="initial"
        label="Initial Minutes"
        name="initialtime"
        extra={<Tag color={ttag}>{timectrl}</Tag>}
      >
        <Slider
          tipFormatter={initTimeFormatter}
          min={0}
          max={initTimeDiscreteScale.length - 1}
          tooltipVisible={true}
        />
      </Form.Item>

      <Form.Item label="Time Setting" name="incOrOT">
        <Radio.Group>
          <Radio.Button value="overtime">Use Max Overtime</Radio.Button>
          <Radio.Button value="increment">Use Increment</Radio.Button>
        </Radio.Group>
      </Form.Item>

      <Form.Item
        className="extra-time-setter"
        label={timeSetting}
        name="extratime"
        extra={extraTimeLabel}
      >
        <InputNumber min={0} max={maxTimeSetting} />
      </Form.Item>

      <Form.Item label="Rated" name="rated" valuePropName="checked">
        <Switch />
      </Form.Item>
    </Form>
  );
};
