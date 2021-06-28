import React, { useCallback, useEffect, useMemo } from 'react';
import {
  Form,
  Radio,
  InputNumber,
  Switch,
  Select,
  Tag,
  Slider,
  AutoComplete,
} from 'antd';

import axios from 'axios';

import { Store } from 'antd/lib/form/interface';
import { useMountedState } from '../utils/mounted';
import { ChallengeRule } from '../gen/macondo/api/proto/macondo/macondo_pb';
import {
  initialTimeMinutesToSlider,
  initTimeDiscreteScale,
  timeCtrlToDisplayName,
} from '../store/constants';
import { MatchUser } from '../gen/api/proto/realtime/realtime_pb';
import { SoughtGame } from '../store/reducers/lobby_reducer';
import { toAPIUrl } from '../api/api';
import { useDebounce } from '../utils/debounce';
import { fixedSettings, seekPropVals } from './fixed_seek_controls';
import { ChallengeRulesFormItem } from './challenge_rules_form_item';
import {
  useFriendsStoreContext,
  usePresenceStoreContext,
} from '../store/store';
import { VariantIcon } from '../shared/variant_icons';

const initTimeFormatter = (val?: number) => {
  return val != null ? initTimeDiscreteScale[val].label : null;
};

type user = {
  username: string;
  uuid: string;
};

type SearchResponse = {
  users: Array<user>;
};

type Props = {
  onFormSubmit: (g: SoughtGame, v?: Store) => void;
  loggedIn: boolean;
  showFriendInput: boolean;
  friendRef?: React.MutableRefObject<string>;
  vsBot?: boolean;
  id: string;
  tournamentID?: string;
  storageKey?: string;
  prefixItems?: React.ReactNode;
  username?: string;
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

export const SeekForm = (props: Props) => {
  const { useState } = useMountedState();
  const { friends } = useFriendsStoreContext();
  const { presences } = usePresenceStoreContext();
  const { tournamentID, username } = props;
  const enableAllLexicons = React.useMemo(
    () => localStorage.getItem('enableAllLexicons') === 'true',
    []
  );

  const enableCSW19X = React.useMemo(
    () => localStorage.getItem('enableCSW19X') === 'true',
    []
  );

  const enableWordSmog = React.useMemo(
    () => localStorage.getItem('enableWordSmog') === 'true' && !props.vsBot,
    [props.vsBot]
  );

  let storageKey = 'lastSeekForm';
  if (props.vsBot) {
    storageKey = 'lastBotForm';
  }
  if (props.showFriendInput) {
    storageKey = 'lastMatchForm';
  }
  if (props.storageKey) {
    storageKey = props.storageKey;
  }

  const storedValues = window.localStorage
    ? JSON.parse(window.localStorage.getItem(storageKey) || '{}')
    : {};
  const givenFriend = useMemo(() => props.friendRef?.current ?? '', [
    props.friendRef,
  ]);
  useEffect(() => {
    if (props.friendRef) {
      return () => {
        props.friendRef!.current = '';
      };
    }
  }, [props.friendRef]);
  const defaultValues: seekPropVals = {
    lexicon: 'CSW19',
    challengerule: ChallengeRule.FIVE_POINT,
    initialtimeslider: initialTimeMinutesToSlider(20),
    rated: true,
    extratime: 1,
    friend: '',
    incOrOT: 'overtime',
    vsBot: false,
    variant: 'classic',
  };
  let disableControls = false;
  let disableVariantControls = false;
  let disableLexiconControls = false;
  let disableChallengeControls = false;
  let initialValues;

  if (props.tournamentID && props.tournamentID in fixedSettings) {
    disableControls = true;
    disableVariantControls = 'variant' in fixedSettings[props.tournamentID];
    disableLexiconControls = 'lexicon' in fixedSettings[props.tournamentID];
    disableChallengeControls =
      'challengerule' in fixedSettings[props.tournamentID];
    initialValues = {
      ...fixedSettings[props.tournamentID],
      friend: givenFriend,
    };
    // This is a bit of a hack; sorry.
    if (!disableVariantControls) {
      initialValues = {
        ...initialValues,
        variant: storedValues.variant || defaultValues.variant,
      };
    }
    if (!disableLexiconControls) {
      initialValues = {
        ...initialValues,
        lexicon: storedValues.lexicon || defaultValues.lexicon,
      };
    }
    if (!disableChallengeControls) {
      initialValues = {
        ...initialValues,
        challengerule:
          storedValues.challengerule || defaultValues.challengerule,
      };
    }
  } else {
    initialValues = {
      ...defaultValues,
      ...storedValues,
      friend: givenFriend,
    };
  }
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
  const [showChallengeRule, setShowChallengeRule] = useState(
    initialValues.lexicon !== 'ECWL'
  );
  const [sliderTooltipVisible, setSliderTooltipVisible] = useState(true);
  const handleDropdownVisibleChange = useCallback((open) => {
    setSliderTooltipVisible(!open);
  }, []);
  const [usernameOptions, setUsernameOptions] = useState<Array<string>>([]);

  const onFormChange = (val: Store, allvals: Store) => {
    if (window.localStorage) {
      localStorage.setItem(
        storageKey,
        JSON.stringify({ ...allvals, friend: '' })
      );
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
      initTimeDiscreteScale[allvals.initialtimeslider].seconds,
      allvals.incOrOT === 'increment'
        ? Math.round(allvals.extratime as number)
        : 0,
      allvals.incOrOT === 'increment'
        ? 0
        : Math.round(allvals.extratime as number)
    );
    if (allvals.lexicon === 'ECWL') {
      setShowChallengeRule(false);
    } else {
      setShowChallengeRule(true);
    }
    setTimectrl(tc);
    setTtag(tt);
  };
  const defaultOptions = useMemo(() => {
    let defaultPlayers: string[] = [];
    if (tournamentID && presences.length) {
      defaultPlayers = presences
        .map((p) => p.username)
        .filter((u) => u !== username);
    } else {
      const friendsArray = friends ? Object.values(friends) : [];
      if (friendsArray.length) {
        defaultPlayers = friendsArray
          .filter((f) => f.channel && f.channel.length > 0)
          .map((f) => f.username);
      }
    }
    return defaultPlayers;
  }, [friends, presences, username, tournamentID]);
  const onUsernameSearch = useCallback(
    (searchText: string) => {
      axios
        .post<SearchResponse>(
          toAPIUrl('user_service.AutocompleteService', 'GetCompletion'),
          {
            prefix: searchText,
          }
        )
        .then((resp) => {
          console.log('resp', resp.data);

          setUsernameOptions(
            !searchText
              ? defaultOptions
              : resp.data.users.map((u) => u.username)
          );
        });
    },
    [defaultOptions]
  );

  const searchUsernameDebounced = useDebounce(onUsernameSearch, 300);

  const onFormSubmit = (val: Store) => {
    const receiver = new MatchUser();
    receiver.setDisplayName(val.friend as string);
    const obj = {
      // These items are assigned by the server:
      seeker: '',
      userRating: '',
      seekID: '',

      lexicon: val.lexicon as string,
      challengeRule:
        (val.lexicon as string) === 'ECWL'
          ? ChallengeRule.VOID
          : (val.challengerule as number),
      initialTimeSecs: initTimeDiscreteScale[val.initialtimeslider].seconds,
      incrementSecs:
        val.incOrOT === 'increment' ? Math.round(val.extratime as number) : 0,
      rated: val.rated as boolean,
      maxOvertimeMinutes:
        val.incOrOT === 'increment' ? 0 : Math.round(val.extratime as number),
      receiver,
      rematchFor: '',
      playerVsBot: props.vsBot || false,
      tournamentID: props.tournamentID || '',
      variant: val.variant as string,
      ...(props.tournamentID &&
        fixedSettings[props.tournamentID] && {
          ...(typeof fixedSettings[props.tournamentID].variant === 'string' && {
            // this is necessary, because variant may not be rendered depending on localStorage.
            variant: fixedSettings[props.tournamentID].variant as string,
          }),
        }),
    };
    props.onFormSubmit(obj, val);
  };

  const validateMessages = {
    required: 'This field is required.',
  };

  useEffect(() => {
    if (usernameOptions.length === 0 && defaultOptions.length > 0) {
      setUsernameOptions(defaultOptions);
    }
  }, [defaultOptions, usernameOptions]);

  return (
    <Form
      id={props.id}
      onValuesChange={onFormChange}
      initialValues={initialValues}
      onFinish={onFormSubmit}
      labelCol={{ span: 6 }}
      wrapperCol={{ span: 24 }}
      layout="horizontal"
      validateMessages={validateMessages}
      name="seekForm"
    >
      {props.prefixItems || null}

      {props.showFriendInput && (
        <Form.Item
          label={props.tournamentID ? 'Opponent' : 'Friend'}
          name="friend"
          rules={[
            {
              required: true,
            },
          ]}
        >
          <AutoComplete
            onSearch={searchUsernameDebounced}
            placeholder="username..."
            onClick={() => setUsernameOptions(defaultOptions)}
            filterOption={(inputValue, option) =>
              !option ||
              !option.value ||
              option.value.toUpperCase().indexOf(inputValue.toUpperCase()) !==
                -1
            }
            onDropdownVisibleChange={handleDropdownVisibleChange}
          >
            {usernameOptions.map((username) => (
              <AutoComplete.Option key={username} value={username}>
                {username}
              </AutoComplete.Option>
            ))}
          </AutoComplete>
        </Form.Item>
      )}

      {enableWordSmog && (
        <Form.Item label="Game type" name="variant">
          <Select disabled={disableVariantControls}>
            <Select.Option value="classic">Classic</Select.Option>
            <Select.Option value="wordsmog">
              <VariantIcon vcode="wordsmog" withName />
            </Select.Option>
          </Select>
        </Form.Item>
      )}

      <Form.Item
        label="Dictionary"
        name="lexicon"
        rules={[
          {
            required: true,
          },
        ]}
      >
        <Select disabled={disableLexiconControls}>
          <Select.Option value="CSW19">CSW 19 (World English)</Select.Option>
          <Select.Option value="NWL20">
            NWL 20 (North American English)
          </Select.Option>
          <Select.Option value="RD28">Deutsch (German)</Select.Option>
          <Select.Option value="NSF21">Norsk (Norwegian)</Select.Option>
          {enableAllLexicons && (
            <React.Fragment>
              <Select.Option value="NWL18">NWL 18 (Obsolete)</Select.Option>
              <Select.Option value="NSWL20">
                NSWL 20 (NASPA School Word List)
              </Select.Option>
              <Select.Option value="ECWL">
                English Common Word List
              </Select.Option>
              {enableCSW19X && (
                <Select.Option value="CSW19X">
                  CSW19X (ASCI Expurgated)
                </Select.Option>
              )}
            </React.Fragment>
          )}
        </Select>
      </Form.Item>
      {showChallengeRule && (
        <ChallengeRulesFormItem disabled={disableChallengeControls} />
      )}
      <Form.Item
        className="initial"
        label="Initial minutes"
        name="initialtimeslider"
        extra={<Tag color={ttag}>{timectrl}</Tag>}
      >
        <Slider
          disabled={disableControls}
          tipFormatter={initTimeFormatter}
          min={0}
          max={initTimeDiscreteScale.length - 1}
          tooltipVisible={sliderTooltipVisible || usernameOptions.length === 0}
        />
      </Form.Item>
      <Form.Item label="Time setting" name="incOrOT">
        <Radio.Group disabled={disableControls}>
          <Radio.Button value="overtime">Use max overtime</Radio.Button>
          <Radio.Button value="increment">Use increment</Radio.Button>
        </Radio.Group>
      </Form.Item>
      <Form.Item
        className="extra-time-setter"
        label={timeSetting}
        name="extratime"
        extra={extraTimeLabel}
      >
        <InputNumber min={0} max={maxTimeSetting} disabled={disableControls} />
      </Form.Item>
      <Form.Item label="Rated" name="rated" valuePropName="checked">
        <Switch disabled={disableControls} />
      </Form.Item>
    </Form>
  );
};
