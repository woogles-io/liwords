import { Form, Select } from "antd";
import { BotTypesEnum, BotTypesEnumProperties } from "./bots";
import React from "react";

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
}

const BotSelector: React.FC<BotSelectorProps> = ({ lexicon }) => {
  const options = botTypes.map((k) => ({
    value: k,
    label: (
      <>
        <div className="bot-selector-item">
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
                {BotTypesEnumProperties[k].description(lexicon)}
              </div>
            </div>
          </div>
        </div>
      </>
    ),
  }));

  return (
    <Form.Item label="Select bot level" name="botType">
      <Select listHeight={600} options={options} className="bot-selector" />
    </Form.Item>
  );
};

export default BotSelector;
