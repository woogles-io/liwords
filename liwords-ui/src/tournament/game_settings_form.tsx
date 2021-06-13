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
} from 'antd';
import { Store } from 'rc-field-form/lib/interface';
import React from 'react';
import { ChallengeRule } from '../gen/macondo/api/proto/macondo/macondo_pb';
import {
  initialTimeMinutesToSlider,
  initTimeDiscreteScale,
  timeCtrlToDisplayName,
} from '../store/constants';
import { useMountedState } from '../utils/mounted';
import { ChallengeRulesFormItem } from '../lobby/challenge_rules_form_item';
import { seekPropVals } from '../lobby/fixed_seek_controls';
import { VariantIcon } from '../shared/variant_icons';
import {
  GameMode,
  GameRequest,
  GameRules,
  RatingMode,
} from '../gen/api/proto/realtime/realtime_pb';

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

type mandatoryFormValues = Partial<seekPropVals> &
  Pick<
    seekPropVals,
    | 'lexicon'
    | 'challengerule'
    | 'initialtimeslider'
    | 'rated'
    | 'extratime'
    | 'incOrOT'
    | 'variant'
  >;

const toFormValues: (gameRequest: GameRequest | null) => mandatoryFormValues = (
  gameRequest: GameRequest | null
) => {
  if (!gameRequest) {
    return {
      lexicon: 'CSW19',
      variant: 'classic',
      challengerule: ChallengeRule.FIVE_POINT,
      initialtimeslider: initialTimeMinutesToSlider(15),
      rated: true,
      extratime: 1,
      incOrOT: 'overtime',
    };
  }

  const vals: mandatoryFormValues = {
    lexicon: gameRequest.getLexicon(),
    variant: gameRequest.getRules()?.getVariantName() ?? '',
    challengerule: gameRequest.getChallengeRule(),
    rated: gameRequest.getRatingMode() === RatingMode.RATED,
    initialtimeslider: 0,
    extratime: 0,
    incOrOT: 'overtime',
  };

  const secs = gameRequest.getInitialTimeSeconds();
  const mins = secs / 60;
  if (mins >= 1) {
    vals.initialtimeslider = mins + 2; // magic slider position
  } else {
    vals.initialtimeslider = secs / 15 - 1; // magic slider position
  }
  if (gameRequest.getMaxOvertimeMinutes()) {
    vals.extratime = gameRequest.getMaxOvertimeMinutes();
    vals.incOrOT = 'overtime';
  } else if (gameRequest.getIncrementSeconds()) {
    vals.extratime = gameRequest.getIncrementSeconds();
    vals.incOrOT = 'increment';
  }
  return vals;
};

export const SettingsForm = (props: Props) => {
  const { useState } = useMountedState();
  const { gameRequest } = props;
  const initialValues = toFormValues(gameRequest);

  const [itc, itt] = timeCtrlToDisplayName(
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
    const [tc, tt] = timeCtrlToDisplayName(
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
