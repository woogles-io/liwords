import React from "react";
import { Form, Select } from "antd";
import { create } from "@bufbuild/protobuf";
import { GameRulesSchema } from "../gen/api/proto/ipc/omgwords_pb";
import { defaultLetterDistribution } from "../lobby/sought_game_interactions";
import { VariantIcon } from "../shared/variant_icons";

// A new annotated game's rules for its lexicon and variant. The server sets
// the board and tiles from the variant itself; these are the ones it picks:
// a super variant, such as ZOMGWords (classic_super), on the 21x21 board
// with the lexicon's 200-tile set, which the English and Catalan tiles have.
export const annotatedGameRules = (lexicon: string, variant: string) => {
  const big = variant.endsWith("_super");
  const letterDistribution = defaultLetterDistribution(lexicon);
  return create(GameRulesSchema, {
    boardLayoutName: big ? "SuperCrosswordGame" : "CrosswordGame",
    letterDistributionName: big
      ? `${letterDistribution}_super`
      : letterDistribution,
    variantName: variant,
  });
};

// The game type of a new annotated game, as the seek form offers it.
export const VariantFormItem = () => (
  <Form.Item label="Game type" name="variant" initialValue="classic">
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
);
