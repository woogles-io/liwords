import { ConfigProvider, Form, Select } from "antd";
import { BotTypesEnum, BotTypesEnumProperties } from "./bots";
import React, { useEffect, useState } from "react";
import ExternalLink from "../assets/external-link.svg?react";
import { Link } from "react-router";
import { Timestamp, timestampDate } from "@bufbuild/protobuf/wkt";
import { LoginWithPatreonLink } from "../settings/integrations";
import PatreonLogo from "../assets/patreon.svg?react";
import { GetSubscriptionCriteriaResponse } from "../gen/api/proto/user_service/user_service_pb";

const botTypes = [
  BotTypesEnum.BEGINNER,
  BotTypesEnum.EASY,
  BotTypesEnum.INTERMEDIATE,
  BotTypesEnum.EXPERT,
  BotTypesEnum.MASTER,
  BotTypesEnum.GRANDMASTER,
];

interface BotSelectorProps {
  lexicon: string;
  specialAccessPlayer: boolean; // specialAccessPlayer has rights to paid bots.
  subscriptionCriteria?: GetSubscriptionCriteriaResponse;
  botType: BotTypesEnum;
  hasPatreonIntegration?: boolean;
}

const tierToNumGames = (tierName: string) => {
  switch (tierName) {
    case "Chihuahua":
      return 4;
    case "Dalmatian":
      return 50;
    case "Golden Retriever":
      // Infinity = 500
      return 500;
  }
};

const nextChargeDate = (lastChargeDate: Date) => {
  const nextDate = new Date(lastChargeDate);
  nextDate.setMonth(nextDate.getMonth() + 1);
  return nextDate.toLocaleDateString("en-US", {
    month: "long",
    day: "numeric",
    year: "numeric",
  });
};

const BotSelector: React.FC<BotSelectorProps> = ({
  lexicon,
  subscriptionCriteria,
  botType,
  hasPatreonIntegration,
  specialAccessPlayer,
}) => {
  let entitledToBestBot: boolean | undefined;
  let tierName: string | undefined;
  let lastChargeDate: Timestamp | undefined;
  if (subscriptionCriteria) {
    ({
      entitledToBotGames: entitledToBestBot,
      tierName,
      lastChargeDate,
    } = subscriptionCriteria);
  }
  // for testing
  // tierName = "Chihuahua";
  // entitledToBestBot = true;

  const [selectedOption, setSelectedOption] = useState(botType);

  useEffect(() => {
    setSelectedOption(botType);
  }, [botType]);

  const handleSelectChange = (value: BotTypesEnum) => {
    setSelectedOption(value);
  };

  const options = botTypes.map((k) => {
    const bestbot = k === BotTypesEnum.GRANDMASTER;
    let displayPtCallout =
      bestbot && (!entitledToBestBot || !tierName || !hasPatreonIntegration);
    // Override if special player:
    if (specialAccessPlayer) {
      displayPtCallout = false;
    }
    return {
      value: k,
      label: (
        <div className="bot-selector-item-holder">
          <div
            className={`bot-selector-item ${displayPtCallout ? "bot-selector-item-disabled" : ""}`}
          >
            <div className="bot-selector-item-layout">
              <img
                className="bot-selector-image"
                src={BotTypesEnumProperties[k].image}
                alt={BotTypesEnumProperties[k].userVisible}
              />
              <div className="bot-selector-info">
                <div className="bot-selector-name-and-short-desc">
                  <div className="bot-selector-name">
                    {BotTypesEnumProperties[k].userVisible}
                  </div>
                  <div className="bot-selector-short-desc">
                    {BotTypesEnumProperties[k].shortDescription}
                  </div>
                </div>
                <div className="bot-selector-long-desc">
                  {bestbot && !entitledToBestBot && tierName && lastChargeDate
                    ? `You've used your ${tierToNumGames(tierName)} games this month. ` +
                      `You can play again on ${nextChargeDate(timestampDate(lastChargeDate))}`
                    : BotTypesEnumProperties[k].description(lexicon)}
                </div>
              </div>
            </div>
          </div>
          {/* display Patreon callout if not subscribed */}
          {displayPtCallout && (
            <PatreonCallout
              tierName={tierName}
              entitledToBestBot={entitledToBestBot}
              hasPatreonIntegration={hasPatreonIntegration}
            />
          )}
        </div>
      ),
    };
  });

  return (
    <ConfigProvider
      theme={{
        components: {
          Select: {
            optionPadding: 0,
            // optionActiveBg: "#eaf7ff",
            // optionSelectedBg: "#eaf7ff",
          },
        },
        token: {
          paddingSM: 0,
          paddingMD: 0,
          paddingLG: 0,
          padding: 0,
          paddingXS: 0,
          paddingXXS: 0,
        },
      }}
    >
      <Form.Item label="Select bot level" name="botType">
        <Select
          listHeight={700}
          options={options}
          className={`bot-selector ${selectedOption === BotTypesEnum.GRANDMASTER && !specialAccessPlayer && !entitledToBestBot ? "bot-selector-contains-patreon-callout" : ""}`}
          onChange={handleSelectChange}
          value={selectedOption}
        />
      </Form.Item>
    </ConfigProvider>
  );
};

interface PatreonCalloutProps {
  hasPatreonIntegration?: boolean;
  tierName?: string;
  entitledToBestBot?: boolean;
}

const PatreonCallout: React.FC<PatreonCalloutProps> = ({
  entitledToBestBot,
  tierName,
  hasPatreonIntegration,
}) => {
  // Account has no Patreon integration
  if (!hasPatreonIntegration) {
    return (
      <LoginWithPatreonLink className="bot-selector-patreon-callout-not-subscribed">
        <PatreonLogo className="pt-callout-link" />
        <div className="pt-callout-text">
          Log in with Patreon to play BestBot
        </div>
      </LoginWithPatreonLink>
    );
  }
  // Account has a Patreon integration, but doesn't have a Patreon subscription
  if (!tierName) {
    return (
      <Link
        to="https://www.patreon.com/woogles_io/"
        className="bot-selector-patreon-callout-not-subscribed"
      >
        <ExternalLink className="pt-callout-link" />
        <div className="pt-callout-text">Join our Patreon to play BestBot</div>
      </Link>
    );
  }
  // Subscribed, but ran out of games:
  if (!entitledToBestBot) {
    return (
      <Link
        to="https://www.patreon.com/woogles_io/"
        className="bot-selector-patreon-callout-ran-out-of-games"
      >
        <ExternalLink className="pt-callout-link" />
        <div className="pt-callout-text">
          Upgrade your Patreon tier to play BestBot
        </div>
      </Link>
    );
  }
};

export default BotSelector;
