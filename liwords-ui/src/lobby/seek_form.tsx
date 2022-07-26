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
  ChallengeRule,
  ChallengeRuleMap,
} from '../gen/macondo/api/proto/macondo/macondo_pb';
import {
  initialTimeMinutesToSlider,
  initialTimeSecondsToSlider,
  initTimeDiscreteScale,
  ratingKey,
  StartingRating,
  timeCtrlToDisplayName,
} from '../store/constants';
import { SoughtGame } from '../store/reducers/lobby_reducer';
import { toAPIUrl } from '../api/api';
import { useDebounce } from '../utils/debounce';
import { ChallengeRulesFormItem } from './challenge_rules_form_item';
import {
  useFriendsStoreContext,
  useLobbyStoreContext,
  usePresenceStoreContext,
  useTournamentStoreContext,
} from '../store/store';
import { VariantIcon } from '../shared/variant_icons';
import { excludedLexica, LexiconFormItem } from '../shared/lexicon_display';
import { AllLexica } from '../shared/lexica';
import { BotTypesEnum, BotTypesEnumProperties } from './bots';
import { GameRequest, RatingMode } from '../gen/api/proto/ipc/omgwords_pb';
import { MatchUser } from '../gen/api/proto/ipc/omgseeks_pb';
import { ProfileUpdate } from '../gen/api/proto/ipc/users_pb';

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
  botType: BotTypesEnum;
  ratingRange: number;
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
  gameRequest: GameRequest | undefined
) => mandatoryFormValues = (gameRequest: GameRequest | undefined) => {
  if (!gameRequest) {
    return {
      lexicon: 'CSW21',
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

const myDisplayRating = (
  ratings: { [k: string]: ProfileUpdate.Rating },
  secs: number,
  incrementSecs: number,
  maxOvertime: number,
  variant: string,
  lexicon: string
) => {
  const r =
    ratings[ratingKey(secs, incrementSecs, maxOvertime, variant, lexicon)];
  if (r) {
    return Math.round(r.getRating());
  }
  return `${StartingRating}?`;
};

export const SeekForm = (props: Props) => {
  const { useState } = useMountedState();
  const { friends } = useFriendsStoreContext();
  const { presences } = usePresenceStoreContext();
  const { tournamentContext } = useTournamentStoreContext();
  const { tournamentID, username } = props;
  const { lobbyContext } = useLobbyStoreContext();

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
  const givenFriend = useMemo(
    () => props.friendRef?.current ?? '',
    [props.friendRef]
  );
  useEffect(() => {
    if (props.friendRef) {
      return () => {
        // why?
        // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
        props.friendRef!.current = '';
      };
    }
  }, [props.friendRef]);
  const defaultValues: seekPropVals = {
    lexicon: 'CSW21',
    challengerule: ChallengeRule.VOID,
    initialtimeslider: initialTimeMinutesToSlider(20),
    rated: true,
    extratime: 1,
    friend: '',
    incOrOT: 'overtime',
    vsBot: false,
    variant: 'classic',
    botType: BotTypesEnum.BEGINNER,
    ratingRange: 500,
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
    const fixedClubSettings =
      tournamentContext.metadata.getDefaultClubSettings();
    const initFormValues = GameRequestToFormValues(fixedClubSettings);
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
  const [selectedSecs, selectedIncrementSecs, selectedMaxOvertime] = [
    initTimeDiscreteScale[initialValues.initialtimeslider].seconds,
    initialValues.incOrOT === 'increment'
      ? Math.round(initialValues.extratime as number)
      : 0,
    initialValues.incOrOT === 'increment'
      ? 0
      : Math.round(initialValues.extratime as number),
  ];

  const [itc, , itt] = timeCtrlToDisplayName(
    selectedSecs,
    selectedIncrementSecs,
    selectedMaxOvertime
  );
  const [timectrl, setTimectrl] = useState(itc);
  const [ttag, setTtag] = useState(itt);
  const [myRating, setMyRating] = useState(
    myDisplayRating(
      lobbyContext.profile.ratings,
      selectedSecs,
      selectedIncrementSecs,
      selectedMaxOvertime,
      initialValues.variant,
      initialValues.lexicon
    )
  );
  const [selections, setSelections] = useState<Store | null>(initialValues);
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
    setSelections(allvals);
    if (allvals.incOrOT === 'increment') {
      setTimeSetting(incLabel);
      setMaxTimeSetting(60);
      setExtraTimeLabel(incUnitLabel);
    } else {
      setTimeSetting(otLabel);
      setMaxTimeSetting(10);
      setExtraTimeLabel(otUnitLabel);
    }
    const secs = initTimeDiscreteScale[allvals.initialtimeslider].seconds;
    const incrementSecs =
      allvals.incOrOT === 'increment'
        ? Math.round(allvals.extratime as number)
        : 0;
    const maxOvertime =
      allvals.incOrOT === 'increment'
        ? 0
        : Math.round(allvals.extratime as number);
    const [tc, , tt] = timeCtrlToDisplayName(secs, incrementSecs, maxOvertime);
    if (allvals.lexicon === 'ECWL') {
      setShowChallengeRule(false);
    } else {
      setShowChallengeRule(true);
    }
    setTimectrl(tc);
    setTtag(tt);
    setLexiconCopyright(AllLexica[allvals.lexicon]?.longDescription);
    setMyRating(
      myDisplayRating(
        lobbyContext.profile.ratings,
        secs,
        incrementSecs,
        maxOvertime,
        allvals.variant || 'classic',
        allvals.lexicon
      )
    );
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
      ratingKey: '',

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
      receiverIsPermanent: receiver.getDisplayName() !== '',
      // these are independent values in the backend but for now will be
      // modified together on the front end.
      minRatingRange: -val.ratingRange || 0,
      maxRatingRange: val.ratingRange || 0,
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
        <Form.Item label="Select bot level" name="botType">
          <Select listHeight={500}>
            {[
              BotTypesEnum.MASTER,
              BotTypesEnum.EXPERT,
              BotTypesEnum.INTERMEDIATE,
              BotTypesEnum.EASY,
              BotTypesEnum.BEGINNER,
            ].map((v) => (
              <Select.Option value={v} key={v}>
                <span className="level">
                  {BotTypesEnumProperties[v].userVisible}{' '}
                </span>
                <span className="average">
                  {BotTypesEnumProperties[v].shortDescription}
                </span>
                <span className="description">
                  {BotTypesEnumProperties[v].description(
                    selections?.lexicon || ''
                  )}{' '}
                </span>
              </Select.Option>
            ))}
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
              (typeof option.value === 'string' &&
                option.value.toUpperCase().indexOf(inputValue.toUpperCase()) !==
                  -1)
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
            <Select.Option value="classic_super">
              <VariantIcon vcode="classic_super" withName />
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
          getTooltipPopupContainer={() =>
            document.getElementById(props.id) as HTMLElement
          }
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

      {!props.showFriendInput && !props.vsBot && (
        <Form.Item label="Rating range" name="ratingRange">
          <Slider
            min={50}
            max={500}
            tipFormatter={(v) => `${myRating} ± ${v ? v : 0}`}
            step={50}
            tooltipVisible={sliderTooltipVisible}
          />
        </Form.Item>
      )}

      <small className="readable-text-color">
        {lexiconCopyright ? lexiconCopyright : ''}
      </small>
    </Form>
  );
};
