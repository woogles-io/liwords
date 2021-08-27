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
import {
  BotRequest,
  ChallengeRule,
  ChallengeRuleMap,
} from '../gen/macondo/api/proto/macondo/macondo_pb';
import {
  initialTimeMinutesToSlider,
  initialTimeSecondsToSlider,
  initTimeDiscreteScale,
  timeCtrlToDisplayName,
} from '../store/constants';
import {
  GameRequest,
  MatchUser,
  RatingMode,
} from '../gen/api/proto/realtime/realtime_pb';
import { SoughtGame } from '../store/reducers/lobby_reducer';
import { toAPIUrl } from '../api/api';
import { useDebounce } from '../utils/debounce';
import { ChallengeRulesFormItem } from './challenge_rules_form_item';
import {
  useFriendsStoreContext,
  usePresenceStoreContext,
  useTournamentStoreContext,
} from '../store/store';
import { VariantIcon } from '../shared/variant_icons';
import { excludedLexica, LexiconFormItem } from '../shared/lexicon_display';
import { AllLexica } from '../shared/lexica';

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

export type seekPropVals = {
  lexicon: string;
  challengerule: ChallengeRuleMap[keyof ChallengeRuleMap];
  initialtimeslider: number;
  rated: boolean;
  extratime: number;
  friend: string;
  incOrOT: 'overtime' | 'increment';
  vsBot: boolean;
  variant: string;
  botType: BotRequest.BotCodeMap[keyof BotRequest.BotCodeMap];
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

export const GameRequestToFormValues: (
  gameRequest: GameRequest | null
) => mandatoryFormValues = (gameRequest: GameRequest | null) => {
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
  try {
    vals.initialtimeslider = initialTimeSecondsToSlider(secs);
  } catch (e) {
    const msg = `cannot find ${secs} seconds in slider`;
    console.error(msg, e);
    alert(msg);
    vals.initialtimeslider = 0;
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
  const { tournamentContext } = useTournamentStoreContext();
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
    botType: BotRequest.BotCode.HASTY_BOT,
  };
  let disableTimeControls = false;
  let disableVariantControls = false;
  let disableLexiconControls = false;
  let disableChallengeControls = false;
  let disableRatedControls = false;
  let initialValues;

  if (
    props.tournamentID &&
    tournamentContext.metadata.getDefaultClubSettings()
  ) {
    const fixedClubSettings = tournamentContext.metadata.getDefaultClubSettings();
    const initFormValues = GameRequestToFormValues(fixedClubSettings!);
    const freeformItems =
      tournamentContext.metadata.getFreeformClubSettingFieldsList() || [];
    disableVariantControls = !freeformItems.includes('variant_name');
    disableLexiconControls = !freeformItems.includes('lexicon');
    disableChallengeControls = !freeformItems.includes('challenge_rule');
    disableTimeControls = !freeformItems.includes('time');
    disableRatedControls = !freeformItems.includes('rating_mode');
    // Pass through default values only if they are NOT disabled
    // (If they are disabled, we should use the hardcoded values)
    const valuesToPassThrough: Partial<seekPropVals> = {};
    if (!disableVariantControls) {
      valuesToPassThrough.variant = storedValues.variant;
    }
    if (!disableLexiconControls) {
      valuesToPassThrough.lexicon = storedValues.lexicon;
    }
    if (!disableChallengeControls) {
      valuesToPassThrough.challengerule = storedValues.challengerule;
    }
    if (!disableTimeControls) {
      valuesToPassThrough.initialtimeslider = storedValues.initialtimeslider;
      valuesToPassThrough.extratime = storedValues.extratime;
      valuesToPassThrough.incOrOT = storedValues.incOrOT;
    }
    if (!disableRatedControls) {
      valuesToPassThrough.rated = storedValues.rated;
    }
    initialValues = {
      ...initFormValues,
      ...valuesToPassThrough,
      friend: givenFriend,
    };
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
  const [lexiconCopyright, setLexiconCopyright] = useState(
    AllLexica[initialValues.lexicon]?.longDescription
  );

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
    setLexiconCopyright(AllLexica[allvals.lexicon].longDescription);
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
      botType: val.botType,
      tournamentID: props.tournamentID || '',
      variant: val.variant as string,
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

      {props.vsBot && (
        <Form.Item label="Select bot" name="botType">
          <Select>
            <Select.Option value={BotRequest.BotCode.HASTY_BOT}>
              HastyBot (5)
            </Select.Option>
            <Select.Option value={BotRequest.BotCode.LEVEL3_PROBABILISTIC}>
              Dumby Bot (3)
            </Select.Option>
            <Select.Option value={BotRequest.BotCode.LEVEL1_PROBABILISTIC}>
              Beginner Bot(1)
            </Select.Option>
          </Select>
        </Form.Item>
      )}

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

      {/* if variant controls are disabled it means we have hardcoded settings
      for it, so show them if not classic */}
      {(enableWordSmog ||
        (disableVariantControls && initialValues.variant !== 'classic')) && (
        <Form.Item label="Game type" name="variant">
          <Select disabled={disableVariantControls}>
            <Select.Option value="classic">Classic</Select.Option>
            <Select.Option value="wordsmog">
              <VariantIcon vcode="wordsmog" withName />
            </Select.Option>
          </Select>
        </Form.Item>
      )}

      <LexiconFormItem
        disabled={disableLexiconControls}
        excludedLexica={excludedLexica(enableAllLexicons, enableCSW19X)}
      />

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
          disabled={disableTimeControls}
          tipFormatter={initTimeFormatter}
          min={0}
          max={initTimeDiscreteScale.length - 1}
          tooltipVisible={sliderTooltipVisible || usernameOptions.length === 0}
        />
      </Form.Item>
      <Form.Item label="Time setting" name="incOrOT">
        <Radio.Group disabled={disableTimeControls}>
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
        <InputNumber
          min={0}
          max={maxTimeSetting}
          disabled={disableTimeControls}
        />
      </Form.Item>
      <Form.Item label="Rated" name="rated" valuePropName="checked">
        <Switch disabled={disableRatedControls} />
      </Form.Item>
      <small className="readable-text-color">
        {lexiconCopyright ? lexiconCopyright : ''}
      </small>
    </Form>
  );
};
