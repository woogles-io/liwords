import { Form, Select } from "antd";
import React from "react";
import { ChallengeRule } from "../gen/api/vendor/macondo/macondo_pb";

type Props = {
  disabled: boolean;
  onDropdownVisibleChange?: (open: boolean) => void;
};

export const ChallengeRulesFormItem = (props: Props) => (
  <Form.Item label="Challenge rule" name="challengerule">
    <Select
      disabled={props.disabled}
      onDropdownVisibleChange={props.onDropdownVisibleChange}
    >
      <Select.Option value={ChallengeRule.VOID}>
        Void{" "}
        <span className="hover-help">(All words are checked before play)</span>
      </Select.Option>
      <Select.Option value={ChallengeRule.FIVE_POINT}>
        5 points{" "}
        <span className="hover-help">(Reward for winning a challenge)</span>
      </Select.Option>
      <Select.Option value={ChallengeRule.TEN_POINT}>
        10 points{" "}
        <span className="hover-help">(Reward for winning a challenge)</span>
      </Select.Option>
      <Select.Option value={ChallengeRule.DOUBLE}>
        Double{" "}
        <span className="hover-help">
          (Turn loss for challenging a valid word)
        </span>
      </Select.Option>
      <Select.Option value={ChallengeRule.SINGLE}>
        Single{" "}
        <span className="hover-help">
          (No penalty for challenging a valid word)
        </span>
      </Select.Option>
      <Select.Option value={ChallengeRule.TRIPLE}>
        Triple{" "}
        <span className="hover-help">(Losing a challenge loses the game)</span>
      </Select.Option>
    </Select>
  </Form.Item>
);
