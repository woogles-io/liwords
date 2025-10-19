// Division management forms

import { Button, Form, Input, message } from "antd";
import { Store } from "rc-field-form/lib/interface";
import React from "react";
import { TournamentService } from "../../../gen/api/proto/tournament_service/tournament_service_pb";
import { flashError, useClient } from "../../../utils/hooks/connect";
import { DivisionFormItem } from "./shared";

export const AddDivision = (props: { tournamentID: string }) => {
  const tClient = useClient(TournamentService);
  const onFinish = async (vals: Store) => {
    const obj = {
      id: props.tournamentID,
      division: vals.division,
    };
    try {
      await tClient.addDivision(obj);
      message.info({
        content: "Division added",
        duration: 3,
      });
    } catch (e) {
      flashError(e);
    }
  };

  return (
    <Form onFinish={onFinish}>
      <Form.Item name="division" label="Division Name">
        <Input />
      </Form.Item>
      <Form.Item>
        <Button type="primary" htmlType="submit">
          Submit
        </Button>
      </Form.Item>
    </Form>
  );
};

export const RenameDivision = (props: { tournamentID: string }) => {
  const tClient = useClient(TournamentService);
  const onFinish = async (vals: Store) => {
    const obj = {
      id: props.tournamentID,
      division: vals.division,
      newName: vals.newName,
    };
    try {
      await tClient.renameDivision(obj);
      message.info({
        content: "Division name changed",
        duration: 3,
      });
    } catch (e) {
      flashError(e);
    }
  };

  return (
    <Form onFinish={onFinish}>
      <DivisionFormItem />
      <Form.Item name="newName" label="New division name">
        <Input />
      </Form.Item>
      <Form.Item>
        <Button type="primary" htmlType="submit">
          Submit
        </Button>
      </Form.Item>
    </Form>
  );
};

export const RemoveDivision = (props: { tournamentID: string }) => {
  const tClient = useClient(TournamentService);

  const onFinish = async (vals: Store) => {
    const obj = {
      id: props.tournamentID,
      division: vals.division,
    };

    try {
      await tClient.removeDivision(obj);
      message.info({
        content: "Division removed",
        duration: 3,
      });
    } catch (e) {
      flashError(e);
    }
  };

  return (
    <Form onFinish={onFinish}>
      <DivisionFormItem />
      <Form.Item>
        <Button type="primary" htmlType="submit">
          Submit
        </Button>
      </Form.Item>
    </Form>
  );
};
