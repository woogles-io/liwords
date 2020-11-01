import React from 'react';
import {
  Form,
  Radio,
  InputNumber,
  Switch,
  Tag,
  Slider,
  AutoComplete,
} from 'antd';

import axios from 'axios';

import { Store } from 'antd/lib/form/interface';
import { useMountedState } from '../utils/mounted';
import { ChallengeRule } from '../gen/macondo/api/proto/macondo/macondo_pb';
import { timeCtrlToDisplayName } from '../store/constants';
import { MatchUser } from '../gen/api/proto/realtime/realtime_pb';
import { SoughtGame } from '../store/reducers/lobby_reducer';
import { toAPIUrl } from '../api/api';
import { debounce } from '../utils/debounce';

export type seekPropVals = { [val: string]: string | number | boolean };

const wholetimes = [];
for (let i = 1; i <= 25; i++) {
  wholetimes.push(i.toString());
}
const initTimeDiscreteScale = [
  '¼',
  '½',
  '¾',
  ...wholetimes,
  '30',
  '35',
  '40',
  '45',
  '50',
  '55',
  '60',
];

const initTimeFormatter = (val?: number) => {
  return initTimeDiscreteScale[val!];
};

const timeScaleToNum = (val: string) => {
  switch (val) {
    case '¼':
      return 0.25;
    case '½':
      return 0.5;
    case '¾':
      return 0.75;
    default:
      return parseInt(val, 10);
  }
};

type SearchResponse = {
  usernames: Array<string>;
};

type Props = {
  onFormSubmit: (g: SoughtGame) => void;
  loggedIn: boolean;
  showFriendInput: boolean;
  vsBot?: boolean;
  id: string;
  tournamentID?: string;
};

const otLabel = 'Overtime';
const incLabel = 'Increment';
const otUnitLabel = 'minutes';
const incUnitLabel = 'seconds';

export const SeekForm = (props: Props) => {
  const { useState } = useMountedState();

  let storageKey = 'lastSeekForm';
  if (props.vsBot) {
    storageKey = 'lastBotForm';
  }
  if (props.showFriendInput) {
    storageKey = 'lastMatchForm';
  }

  const storedValues = window.localStorage
    ? JSON.parse(window.localStorage.getItem(storageKey) || '{}')
    : {};
  const defaultValues: seekPropVals = {
    lexicon: 'CSW19',
    challengerule: ChallengeRule.FIVE_POINT,
    initialtime: 12, // Note this isn't minutes, but the slider position.
    rated: true,
    extratime: 1,
    friend: '',
    incOrOT: 'overtime',
    vsBot: false,
  };
  const initialValues = {
    ...defaultValues,
    ...storedValues,
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
  const [sliderTooltipVisible, setSliderTooltipVisible] = useState(true);
  const [usernameOptions, setUsernameOptions] = useState<Array<string>>([]);
  const onFormChange = (val: Store, allvals: Store) => {
    if (window.localStorage) {
      localStorage.setItem(storageKey, JSON.stringify(allvals));
    }
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

  const onUsernameSearch = (searchText: string) => {
    axios
      .post<SearchResponse>(
        toAPIUrl('user_service.AutocompleteService', 'GetCompletion'),
        {
          prefix: searchText,
        }
      )
      .then((resp) => {
        console.log('resp', resp.data);
        setUsernameOptions(!searchText ? [] : resp.data.usernames);
      });
  };

  const searchUsernameDebounced = debounce(onUsernameSearch, 1000);

  const onFormSubmit = (val: Store) => {
    const receiver = new MatchUser();
    receiver.setDisplayName(val.friend as string);
    const obj = {
      // These items are assigned by the server:
      seeker: '',
      userRating: '',
      seekID: '',

      lexicon: val.lexicon as string,
      challengeRule: val.challengerule as number,
      initialTimeSecs:
        timeScaleToNum(initTimeDiscreteScale[val.initialtime]) * 60,
      incrementSecs:
        val.incOrOT === 'increment' ? Math.round(val.extratime as number) : 0,
      rated: val.rated as boolean,
      maxOvertimeMinutes:
        val.incOrOT === 'increment' ? 0 : Math.round(val.extratime as number),
      receiver,
      rematchFor: '',
      playerVsBot: props.vsBot || false,
      tournamentID: props.tournamentID || '',
    };
    props.onFormSubmit(obj);
  };

  return (
    <Form
      id={props.id}
      onValuesChange={onFormChange}
      initialValues={initialValues}
      onFinish={onFormSubmit}
      labelCol={{ span: 6 }}
      wrapperCol={{ span: 24 }}
      layout="horizontal"
    >
      {props.showFriendInput && (
        <Form.Item label="Friend" name="friend">
          <AutoComplete
            onSearch={searchUsernameDebounced}
            placeholder="username..."
            style={{
              width: 200,
            }}
            onFocus={() => setSliderTooltipVisible(false)}
            onBlur={() => setSliderTooltipVisible(true)}
          >
            {usernameOptions.map((username) => (
              <AutoComplete.Option key={username} value={username}>
                {username}
              </AutoComplete.Option>
            ))}
          </AutoComplete>
        </Form.Item>
      )}
      <Form.Item label="Dictionary" name="lexicon">
        <Radio.Group>
          <Radio.Button value="CSW19">CSW19</Radio.Button>
          <Radio.Button value="NWL18">NWL18</Radio.Button>
        </Radio.Group>
      </Form.Item>
      <Form.Item label="Challenge rule" name="challengerule">
        <Radio.Group>
          <Radio.Button value={ChallengeRule.FIVE_POINT}>5-pt</Radio.Button>
          <Radio.Button value={ChallengeRule.TEN_POINT}>10-pt</Radio.Button>
          <Radio.Button value={ChallengeRule.DOUBLE}>Double</Radio.Button>
          <Radio.Button value={ChallengeRule.SINGLE}>Single</Radio.Button>
          <Radio.Button value={ChallengeRule.VOID}>Void</Radio.Button>
          <Radio.Button value={ChallengeRule.TRIPLE}>Triple</Radio.Button>
        </Radio.Group>
      </Form.Item>
      <Form.Item label="Initial Minutes" name="initialtime">
        <Slider
          tipFormatter={initTimeFormatter}
          min={0}
          max={initTimeDiscreteScale.length - 1}
          tooltipVisible={sliderTooltipVisible}
        />
      </Form.Item>
      <Form.Item label="Time Setting" name="incOrOT">
        <Radio.Group>
          <Radio.Button value="overtime">Use Max Overtime</Radio.Button>
          <Radio.Button value="increment">Use Increment</Radio.Button>
        </Radio.Group>
      </Form.Item>
      <Form.Item label={timeSetting} name="extratime" extra={extraTimeLabel}>
        <InputNumber min={0} max={maxTimeSetting} />
      </Form.Item>
      <Form.Item label="Rated" name="rated" valuePropName="checked">
        <Switch />
      </Form.Item>
      <Form.Item label="Time Control">
        <Tag color={ttag}>{timectrl}</Tag>
      </Form.Item>
    </Form>
  );
};
