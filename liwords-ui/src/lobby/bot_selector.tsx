import { ConfigProvider, Form, Select } from "antd";
import { BotTypesEnum, BotTypesEnumProperties } from "./bots";
import React, { useEffect, useState } from "react";
import ExternalLink from "../assets/external-link.svg?react";
import { Link } from "react-router";
import { Timestamp, timestampDate } from "@bufbuild/protobuf/wkt";

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
  entitledToBestBot?: boolean;
  lastChargeDate?: Timestamp;
  tierName?: string;
  botType: BotTypesEnum;
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
  tierName,
  lastChargeDate,
  entitledToBestBot,
  botType,
}) => {
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
    return {
      value: k,
      label: (
        <div className="bot-selector-item-holder">
          <div
            className={`bot-selector-item ${bestbot && !entitledToBestBot ? "bot-selector-item-disabled" : ""}`}
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
          {bestbot && !tierName && (
            <Link
              to="https://www.patreon.com/woogles_io/"
              className="bot-selector-patreon-callout-not-subscribed"
            >
              <ExternalLink className="pt-callout-link" />
              <div className="pt-callout-text">
                Join our Patreon to play BestBot
              </div>
            </Link>
          )}
          {/* display Patreon callout if subscribed but ran out of games */}
          {bestbot && !entitledToBestBot && tierName && (
            <Link
              to="https://www.patreon.com/woogles_io/"
              className="bot-selector-patreon-callout-ran-out-of-games"
            >
              <ExternalLink className="pt-callout-link" />
              <div className="pt-callout-text">
                Upgrade your Patreon tier to play BestBot
              </div>
            </Link>
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
          className={`bot-selector ${selectedOption === BotTypesEnum.GRANDMASTER && !entitledToBestBot ? "bot-selector-contains-patreon-callout" : ""}`}
          onChange={handleSelectChange}
          value={selectedOption}
        />
      </Form.Item>
    </ConfigProvider>
  );
};

export default BotSelector;
