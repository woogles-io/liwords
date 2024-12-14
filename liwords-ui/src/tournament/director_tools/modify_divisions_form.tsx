import { Input, Form } from "antd";
import { Store } from "antd/lib/form/interface";
import React from "react";
import { SeekForm } from "../../lobby/seek_form";
import { SoughtGame } from "../../store/reducers/lobby_reducer";

type Props = {
  tournamentID: string;
  onFormSubmit: (g: SoughtGame, v?: Store) => void;
};

export const ModifyDivisionsForm = (props: Props) => {
  return (
    <>
      <SeekForm
        loggedIn
        showFriendInput={false}
        vsBot={false}
        id="division-settings-form"
        onFormSubmit={props.onFormSubmit}
        storageKey={`tournament-${props.tournamentID}`}
        prefixItems={
          <Form.Item
            label="Division Name"
            name="divisionname"
            rules={[{ required: true }]}
          >
            <Input />
          </Form.Item>
        }
      ></SeekForm>
    </>
  );
};
