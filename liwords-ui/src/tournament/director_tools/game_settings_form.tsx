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
} from "antd";
import { Store } from "rc-field-form/lib/interface";
import React, { useState } from "react";
import {
  challRuleToStr,
  initTimeDiscreteScale,
  timeCtrlToDisplayName,
} from "../../store/constants";
import { ChallengeRulesFormItem } from "../../lobby/challenge_rules_form_item";
import { VariantIcon } from "../../shared/variant_icons";
import { LexiconFormItem } from "../../shared/lexicon_display";
import { SmileOutlined } from "@ant-design/icons";
import { GameRequestToFormValues } from "../../lobby/seek_form";
import {
  GameMode,
  GameRequest,
  GameRequestSchema,
  GameRulesSchema,
  RatingMode,
} from "../../gen/api/proto/ipc/omgwords_pb";
import { defaultLetterDistribution } from "../../lobby/sought_game_interactions";
import { create } from "@bufbuild/protobuf";

type Props = {
  setGameRequest: (gr: GameRequest) => void;
  gameRequest: GameRequest | undefined;
};

const otLabel = "Overtime";
const incLabel = "Increment";
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
  const { gameRequest } = props;
  const initialValues = GameRequestToFormValues(gameRequest);

  const [itc, , itt] = timeCtrlToDisplayName(
    initTimeDiscreteScale[initialValues.initialtimeslider].seconds,
    initialValues.incOrOT === "increment"
      ? Math.round(initialValues.extratime as number)
      : 0,
    initialValues.incOrOT === "increment"
      ? 0
      : Math.round(initialValues.extratime as number),
  );
  const [timectrl, setTimectrl] = useState(itc);
  const [ttag, setTtag] = useState(itt);
  const [timeSetting, setTimeSetting] = useState(
    initialValues.incOrOT === "overtime" ? otLabel : incLabel,
  );
  const [extraTimeLabel, setExtraTimeLabel] = useState(
    initialValues.incOrOT === "overtime" ? otUnitLabel : incUnitLabel,
  );
  const [maxTimeSetting, setMaxTimeSetting] = useState(
    initialValues.incOrOT === "overtime" ? 10 : 60,
  );

  const onFormChange = (val: Store, allvals: Store) => {
    if (allvals.incOrOT === "increment") {
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
      allvals.incOrOT === "increment"
        ? Math.round(allvals.extratime as number)
        : 0,
      allvals.incOrOT === "increment"
        ? 0
        : Math.round(allvals.extratime as number),
    );
    setTimectrl(tc);
    setTtag(tt);
  };

  const validateMessages = {
    required: "This field is required.",
  };

  const submitGameReq = (values: Store) => {
    const gr = create(GameRequestSchema, {});
    const rules = create(GameRulesSchema, {
      boardLayoutName: "CrosswordGame",
      letterDistributionName: defaultLetterDistribution(values.lexicon),
      variantName: values.variant,
    });
    gr.rules = rules;

    gr.lexicon = values.lexicon;
    gr.initialTimeSeconds =
      initTimeDiscreteScale[values.initialtimeslider].seconds;

    if (values.incOrOT === "increment") {
      gr.incrementSeconds = values.extratime;
    } else {
      gr.maxOvertimeMinutes = values.extratime;
    }
    gr.challengeRule = values.challengerule;
    gr.gameMode = GameMode.REAL_TIME; // Hardcoded to REAL_TIME for clubs
    gr.ratingMode = values.rated ? RatingMode.RATED : RatingMode.CASUAL;
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
          <Select.Option value="classic_super">
            <VariantIcon vcode="classic_super" withName />
          </Select.Option>
        </Select>
      </Form.Item>

      <ChallengeRulesFormItem disabled={false} />

      <Form.Item
        className="initial custom-tags"
        label="Initial Minutes"
        name="initialtimeslider"
        extra={<Tag color={ttag}>{timectrl}</Tag>}
      >
        <Slider
          tooltip={{
            formatter: initTimeFormatter,
            open: true,
            getPopupContainer: (triggerNode) =>
              triggerNode.parentElement ?? document.body,
          }}
          min={0}
          max={initTimeDiscreteScale.length - 1}
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
        <InputNumber inputMode="numeric" min={0} max={maxTimeSetting} />
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

export const DisplayedGameSetting = (gr: GameRequest | undefined) => {
  return gr ? (
    <dl className="ant-form-text readable-text-color">
      <dt>Initial Time (Minutes)</dt>
      <dd>{gr.initialTimeSeconds / 60}</dd>
      <dt>Variant</dt>
      <dd>{gr.rules?.variantName}</dd>
      <dt>Lexicon</dt>
      <dd>{gr.lexicon}</dd>
      <dt>Max Overtime (Minutes)</dt>
      <dd>{gr.maxOvertimeMinutes}</dd>
      <dt>Increment (Seconds)</dt>
      <dd>{gr.incrementSeconds}</dd>
      <dt>Challenge Rule</dt>
      <dd>{challRuleToStr(gr.challengeRule)}</dd>
      <dt>Rated</dt>
      <dd>{gr.ratingMode === RatingMode.RATED ? "Yes" : "No"}</dd>
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
