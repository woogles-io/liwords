import React, { useCallback, useEffect, useMemo, useState } from "react";
import {
  Form,
  Radio,
  InputNumber,
  Switch,
  Select,
  Tag,
  Slider,
  AutoComplete,
} from "antd";

import { Store } from "antd/lib/form/interface";
import { ChallengeRule } from "../gen/api/vendor/macondo/macondo_pb";
import {
  initialTimeMinutesToSlider,
  initialTimeSecondsToSlider,
  initTimeDiscreteScale,
  ratingKey,
  StartingRating,
  timeCtrlToDisplayName,
} from "../store/constants";
import { SoughtGame } from "../store/reducers/lobby_reducer";
import { useDebounce } from "../utils/debounce";
import { ChallengeRulesFormItem } from "./challenge_rules_form_item";
import {
  useFriendsStoreContext,
  useLobbyStoreContext,
  usePresenceStoreContext,
  useTournamentStoreContext,
} from "../store/store";
import { VariantIcon } from "../shared/variant_icons";
import { excludedLexica, LexiconFormItem } from "../shared/lexicon_display";
import { AllLexica } from "../shared/lexica";
import { BotTypesEnum, BotTypesEnumProperties } from "./bots";
import { GameRequest, RatingMode } from "../gen/api/proto/ipc/omgwords_pb";
import { MatchUserSchema } from "../gen/api/proto/ipc/omgseeks_pb";
import { ProfileUpdate_Rating } from "../gen/api/proto/ipc/users_pb";
import { useClient } from "../utils/hooks/connect";
import { AutocompleteService } from "../gen/api/proto/user_service/user_service_pb";
import { create } from "@bufbuild/protobuf";
import BotSelector from "./bot_selector";
import { useQuery } from "@connectrpc/connect-query";
import { getIntegrations } from "../gen/api/proto/user_service/user_service-IntegrationService_connectquery";
import {
  getSelfRoles,
  getSubscriptionCriteria,
} from "../gen/api/proto/user_service/user_service-AuthorizationService_connectquery";

const initTimeFormatter = (val?: number) => {
  return val != null ? initTimeDiscreteScale[val].label : null;
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
  challengerule: ChallengeRule;
  initialtimeslider: number;
  rated: boolean;
  extratime: number;
  friend: string;
  incOrOT: "overtime" | "increment";
  vsBot: boolean;
  variant: string;
  botType: BotTypesEnum;
  ratingRange: number;
};

type mandatoryFormValues = Partial<seekPropVals> &
  Pick<
    seekPropVals,
    | "lexicon"
    | "challengerule"
    | "initialtimeslider"
    | "rated"
    | "extratime"
    | "incOrOT"
    | "variant"
  >;

export const GameRequestToFormValues: (
  gameRequest: GameRequest | undefined,
) => mandatoryFormValues = (gameRequest: GameRequest | undefined) => {
  if (!gameRequest) {
    return {
      lexicon: "CSW24",
      variant: "classic",
      challengerule: ChallengeRule.FIVE_POINT,
      initialtimeslider: initialTimeMinutesToSlider(15),
      rated: true,
      extratime: 1,
      incOrOT: "overtime",
    };
  }

  const vals: mandatoryFormValues = {
    lexicon: gameRequest.lexicon,
    variant: gameRequest.rules?.variantName ?? "",
    challengerule: gameRequest.challengeRule,
    rated: gameRequest.ratingMode === RatingMode.RATED,
    initialtimeslider: 0,
    extratime: 0,
    incOrOT: "overtime",
  };

  const secs = gameRequest.initialTimeSeconds;
  try {
    vals.initialtimeslider = initialTimeSecondsToSlider(secs);
  } catch (e) {
    const msg = `cannot find ${secs} seconds in slider`;
    console.error(msg, e);
    alert(msg);
    vals.initialtimeslider = 0;
  }
  if (gameRequest.maxOvertimeMinutes) {
    vals.extratime = gameRequest.maxOvertimeMinutes;
    vals.incOrOT = "overtime";
  } else if (gameRequest.incrementSeconds) {
    vals.extratime = gameRequest.incrementSeconds;
    vals.incOrOT = "increment";
  }
  return vals;
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

const myDisplayRating = (
  ratings: { [k: string]: ProfileUpdate_Rating },
  secs: number,
  incrementSecs: number,
  maxOvertime: number,
  variant: string,
  lexicon: string,
) => {
  const r =
    ratings[ratingKey(secs, incrementSecs, maxOvertime, variant, lexicon)];
  if (r) {
    return Math.round(r.rating);
  }
  return `${StartingRating}?`;
};

export const SeekForm = (props: Props) => {
  const { friends } = useFriendsStoreContext();
  const { presences } = usePresenceStoreContext();
  const { tournamentContext } = useTournamentStoreContext();
  const { tournamentID, username } = props;
  const { lobbyContext } = useLobbyStoreContext();

  const { data: subscriptionCriteria } = useQuery(
    getSubscriptionCriteria,
    {},
    { enabled: !!props.vsBot },
  );

  const { data: userIntegrations } = useQuery(
    getIntegrations,
    {},
    { enabled: !!props.vsBot },
  );

  const { data: ourRoles } = useQuery(
    getSelfRoles,
    {},
    { enabled: !!props.vsBot },
  );

  const enableAllLexicons = React.useMemo(
    () => localStorage.getItem("enableAllLexicons") === "true",
    [],
  );

  const enableCSW24X = React.useMemo(
    () => localStorage.getItem("enableCSW24X") === "true",
    [],
  );

  const enableVariants = React.useMemo(
    () => localStorage.getItem("enableVariants") === "true",
    [],
  );

  let storageKey = "lastSeekForm";
  if (props.vsBot) {
    storageKey = "lastBotForm";
  }
  if (props.showFriendInput) {
    storageKey = "lastMatchForm";
  }
  if (props.storageKey) {
    storageKey = props.storageKey;
  }

  const storedValues = window.localStorage
    ? JSON.parse(window.localStorage.getItem(storageKey) || "{}")
    : {};

  switch (storedValues.lexicon) {
    case "NWL20":
    case "NWL18":
      storedValues.lexicon = "NWL23";
      break;
    case "CSW19":
    case "CSW21":
      storedValues.lexicon = "CSW24";
      break;
    case "FRA20":
      storedValues.lexicon = "FRA24";
      break;
    case "OSPS49":
      storedValues.lexicon = "OSPS50";
      break;
  }

  const givenFriend = useMemo(
    () => props.friendRef?.current ?? "",
    [props.friendRef],
  );
  useEffect(() => {
    if (props.friendRef) {
      return () => {
        props.friendRef!.current = "";
      };
    }
  }, [props.friendRef]);
  const defaultValues: seekPropVals = {
    lexicon: "CSW24",
    challengerule: ChallengeRule.VOID,
    initialtimeslider: initialTimeMinutesToSlider(20),
    rated: true,
    extratime: 1,
    friend: "",
    incOrOT: "overtime",
    vsBot: false,
    variant: "classic",
    botType: BotTypesEnum.BEGINNER,
    ratingRange: 500,
  };
  let disableTimeControls = false;
  let disableVariantControls = false;
  let disableLexiconControls = false;
  let disableChallengeControls = false;
  let disableRatedControls = false;
  let initialValues;

  if (props.tournamentID && tournamentContext.metadata.defaultClubSettings) {
    const fixedClubSettings = tournamentContext.metadata.defaultClubSettings;
    const initFormValues = GameRequestToFormValues(fixedClubSettings);
    const freeformItems =
      tournamentContext.metadata.freeformClubSettingFields || [];
    disableVariantControls = !freeformItems.includes("variant_name");
    disableLexiconControls = !freeformItems.includes("lexicon");
    disableChallengeControls = !freeformItems.includes("challenge_rule");
    disableTimeControls = !freeformItems.includes("time");
    disableRatedControls = !freeformItems.includes("rating_mode");
    // Pass through default values only if they are NOT disabled
    // (If they are disabled, we should use the hardcoded values)
    const valuesToPassThrough: Partial<seekPropVals> = {};
    if (!disableVariantControls) {
      valuesToPassThrough.variant =
        storedValues.variant || defaultValues.variant;
    }
    if (!disableLexiconControls) {
      valuesToPassThrough.lexicon =
        storedValues.lexicon || defaultValues.lexicon;
    }
    if (!disableChallengeControls) {
      valuesToPassThrough.challengerule =
        storedValues.challengerule || defaultValues.challengerule;
    }
    if (!disableTimeControls) {
      valuesToPassThrough.initialtimeslider =
        storedValues.initialtimeslider || defaultValues.initialtimeslider;
      valuesToPassThrough.extratime =
        storedValues.extratime || defaultValues.extratime;
      valuesToPassThrough.incOrOT =
        storedValues.incOrOT || defaultValues.incOrOT;
    }
    if (!disableRatedControls) {
      valuesToPassThrough.rated = storedValues.rated || defaultValues.rated;
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
    initialValues.incOrOT === "increment"
      ? Math.round(initialValues.extratime as number)
      : 0,
    initialValues.incOrOT === "increment"
      ? 0
      : Math.round(initialValues.extratime as number),
  ];

  const [itc, , itt] = timeCtrlToDisplayName(
    selectedSecs,
    selectedIncrementSecs,
    selectedMaxOvertime,
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
      initialValues.lexicon,
    ),
  );
  const [selections, setSelections] = useState<Store | null>(initialValues);
  const [timeSetting, setTimeSetting] = useState(
    initialValues.incOrOT === "overtime" ? otLabel : incLabel,
  );
  const [extraTimeLabel, setExtraTimeLabel] = useState(
    initialValues.incOrOT === "overtime" ? otUnitLabel : incUnitLabel,
  );
  const [maxTimeSetting, setMaxTimeSetting] = useState(
    initialValues.incOrOT === "overtime" ? 10 : 60,
  );
  const [hideChallengeRule, setHideChallengeRule] = useState(
    initialValues.lexicon === "ECWL" ||
      (props.vsBot &&
        BotTypesEnumProperties[initialValues.botType as BotTypesEnum].voidonly),
  );
  const [sliderTooltipVisible, setSliderTooltipVisible] = useState(true);
  const handleDropdownVisibleChange = useCallback((open: boolean) => {
    setSliderTooltipVisible(!open);
  }, []);
  const [usernameOptions, setUsernameOptions] = useState<Array<string>>([]);
  const [lexiconCopyright, setLexiconCopyright] = useState(
    AllLexica[initialValues.lexicon]?.longDescription,
  );

  const onFormChange = (val: Store, allvals: Store) => {
    let shouldHideChallengeRule = false;
    if (props.vsBot) {
      const voidonly =
        BotTypesEnumProperties[allvals.botType as BotTypesEnum].voidonly;
      if (voidonly) {
        shouldHideChallengeRule = true;
        allvals.challengerule = ChallengeRule.VOID;
      }
    }
    if (window.localStorage) {
      localStorage.setItem(
        storageKey,
        JSON.stringify({ ...allvals, friend: "" }),
      );
    }
    setSelections(allvals);
    if (allvals.incOrOT === "increment") {
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
      allvals.incOrOT === "increment"
        ? Math.round(allvals.extratime as number)
        : 0;
    const maxOvertime =
      allvals.incOrOT === "increment"
        ? 0
        : Math.round(allvals.extratime as number);
    const [tc, , tt] = timeCtrlToDisplayName(secs, incrementSecs, maxOvertime);
    if (allvals.lexicon === "ECWL") {
      shouldHideChallengeRule = true;
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
        allvals.variant || "classic",
        allvals.lexicon,
      ),
    );
    setHideChallengeRule(shouldHideChallengeRule);
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
  const acClient = useClient(AutocompleteService);

  const onUsernameSearch = useCallback(
    async (searchText: string) => {
      const resp = await acClient.getCompletion({ prefix: searchText });
      setUsernameOptions(
        !searchText ? defaultOptions : resp.users.map((u) => u.username),
      );
    },
    [defaultOptions, acClient],
  );

  const searchUsernameDebounced = useDebounce(onUsernameSearch, 300);

  const onFormSubmit = (val: Store) => {
    const receiver = create(MatchUserSchema, {
      displayName: val.friend as string,
    });
    const obj = {
      // These items are assigned by the server:
      seeker: "",
      userRating: "",
      seekID: "",
      ratingKey: "",

      lexicon: val.lexicon as string,
      challengeRule:
        (val.lexicon as string) === "ECWL"
          ? ChallengeRule.VOID
          : (val.challengerule as number),
      initialTimeSecs: initTimeDiscreteScale[val.initialtimeslider].seconds,
      incrementSecs:
        val.incOrOT === "increment" ? Math.round(val.extratime as number) : 0,
      rated: val.rated as boolean,
      maxOvertimeMinutes:
        val.incOrOT === "increment" ? 0 : Math.round(val.extratime as number),
      receiver,
      rematchFor: "",
      playerVsBot: props.vsBot || false,
      botType: val.botType,
      tournamentID: props.tournamentID || "",
      variant: (val.variant as string) || "",
      receiverIsPermanent: receiver.displayName !== "",
      // these are independent values in the backend but for now will be
      // modified together on the front end.
      minRatingRange: -val.ratingRange || 0,
      maxRatingRange: val.ratingRange || 0,
    };
    props.onFormSubmit(obj, val);
  };

  const validateMessages = {
    required: "This field is required.",
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
        <BotSelector
          lexicon={selections?.lexicon || ""}
          subscriptionCriteria={subscriptionCriteria}
          specialAccessPlayer={
            ourRoles?.roles.includes("Special Access Player") || false
          }
          botType={initialValues.botType}
          hasPatreonIntegration={userIntegrations?.integrations.some(
            (i) => i.integrationName === "patreon",
          )}
        />
      )}
      {props.showFriendInput && (
        <Form.Item
          label={props.tournamentID ? "Opponent" : "Friend"}
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
              (typeof option.value === "string" &&
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
      {(enableVariants ||
        (disableVariantControls && initialValues.variant !== "classic")) && (
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
        excludedLexica={excludedLexica(enableAllLexicons, enableCSW24X)}
      />
      {!hideChallengeRule && (
        <ChallengeRulesFormItem disabled={disableChallengeControls} />
      )}
      <Form.Item
        className="initial custom-tags"
        label="Initial minutes"
        name="initialtimeslider"
        extra={<Tag color={ttag}>{timectrl}</Tag>}
      >
        <Slider
          disabled={disableTimeControls}
          tooltip={{
            formatter: initTimeFormatter,
            open: sliderTooltipVisible || usernameOptions.length === 0,
            getPopupContainer: (triggerNode) =>
              triggerNode.parentElement ?? document.body,
          }}
          min={0}
          max={initTimeDiscreteScale.length - 1}
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
          inputMode="numeric"
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
            tooltip={{
              formatter: (v) => `${myRating} ± ${v ? v : 0}`,
              open: sliderTooltipVisible,
              getPopupContainer: (triggerNode) =>
                triggerNode.parentElement ?? document.body,
            }}
            step={50}
          />
        </Form.Item>
      )}

      <small className="readable-text-color">
        {lexiconCopyright ? lexiconCopyright : ""}
      </small>
    </Form>
  );
};
