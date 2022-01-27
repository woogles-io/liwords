// This form has a lot in common with the form in seek_form.tsx, but it is simpler.
import {
  Button,
  Form,
  InputNumber,
  Radio,
  Select,
  Slider,
  Switch,
  Tag,
  Typography,
} from 'antd';
import { Store } from 'rc-field-form/lib/interface';
import React from 'react';
import {
  challRuleToStr,
  initTimeDiscreteScale,
  timeCtrlToDisplayName,
} from '../store/constants';
import { useMountedState } from '../utils/mounted';
import { ChallengeRulesFormItem } from '../lobby/challenge_rules_form_item';
import { VariantIcon } from '../shared/variant_icons';
import { LexiconFormItem } from '../shared/lexicon_display';
import { SmileOutlined } from '@ant-design/icons';
import { GameRequestToFormValues } from '../lobby/seek_form';
import {
  GameMode,
  GameRequest,
  GameRules,
  RatingMode,
} from '../gen/api/proto/ipc/omgwords_pb';

type Props = {
  setGameRequest: (gr: GameRequest) => void;
  gameRequest: GameRequest | null;
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
  return val != null ? initTimeDiscreteScale[val].label : null;
};

export const SettingsForm = (props: Props) => {
  const { useState } = useMountedState();
  const { gameRequest } = props;
  const initialValues = GameRequestToFormValues(gameRequest);

  const [itc, , itt] = timeCtrlToDisplayName(
    initTimeDiscreteScale[initialValues.initialtimeslider].seconds,
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
    const [tc, , tt] = timeCtrlToDisplayName(
      initTimeDiscreteScale[allvals.initialtimeslider].seconds,
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

  const submitGameReq = (values: Store) => {
    const gr = new GameRequest();
    const rules = new GameRules();
    rules.setBoardLayoutName('CrosswordGame');
    rules.setLetterDistributionName('English');
    rules.setVariantName(values.variant);
    gr.setRules(rules);

    gr.setLexicon(values.lexicon);
    gr.setInitialTimeSeconds(
      initTimeDiscreteScale[values.initialtimeslider].seconds
    );

    if (values.incOrOT === 'increment') {
      gr.setIncrementSeconds(values.extratime);
    } else {
      gr.setMaxOvertimeMinutes(values.extratime);
    }
    gr.setChallengeRule(values.challengerule);
    gr.setGameMode(GameMode.REAL_TIME);
    gr.setRatingMode(values.rated ? RatingMode.RATED : RatingMode.CASUAL);
    props.setGameRequest(gr);
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
      onFinish={submitGameReq}
    >
      <LexiconFormItem />

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
        name="initialtimeslider"
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

      <Form.Item>
        <Button type="primary" htmlType="submit">
          Submit
        </Button>
      </Form.Item>
    </Form>
  );
};

export const DisplayedGameSetting = (gr: GameRequest | null) => {
  return gr ? (
    <dl className="ant-form-text readable-text-color">
      <dt>Initial Time (Minutes)</dt>
      <dd>{gr.getInitialTimeSeconds() / 60}</dd>
      <dt>Variant</dt>
      <dd>{gr.getRules()?.getVariantName()}</dd>
      <dt>Lexicon</dt>
      <dd>{gr.getLexicon()}</dd>
      <dt>Max Overtime (Minutes)</dt>
      <dd>{gr.getMaxOvertimeMinutes()}</dd>
      <dt>Increment (Seconds)</dt>
      <dd>{gr.getIncrementSeconds()}</dd>
      <dt>Challenge Rule</dt>
      <dd>{challRuleToStr(gr.getChallengeRule())}</dd>
      <dt>Rated</dt>
      <dd>{gr.getRatingMode() === RatingMode.RATED ? 'Yes' : 'No'}</dd>
    </dl>
  ) : (
    <Typography.Text
      className="ant-form-text readable-text-color"
      type="secondary"
    >
      ( <SmileOutlined /> No game settings yet. )
    </Typography.Text>
  );
};
