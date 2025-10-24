// Player management forms

import { MinusCircleOutlined, PlusOutlined } from "@ant-design/icons";
import { Button, Form, Input, message, Space, Switch } from "antd";
import { Store } from "rc-field-form/lib/interface";
import React, { useState } from "react";
import { TournamentService } from "../../../gen/api/proto/tournament_service/tournament_service_pb";
import { useTournamentStoreContext } from "../../../store/store";
import { flashError, useClient } from "../../../utils/hooks/connect";
import { DivisionFormItem, PlayersFormItem, showError } from "./shared";

export const AddPlayers = (props: { tournamentID: string }) => {
  const { tournamentContext } = useTournamentStoreContext();

  const tClient = useClient(TournamentService);
  const [showRating, setShowRating] = useState(
    tournamentContext.metadata.irlMode,
  );
  const onFinish = async (vals: Store) => {
    const players = [];
    // const playerMap: { [username: string]: number } = {};
    if (!vals.players) {
      showError("Add some players first");
      return;
    }
    for (let i = 0; i < vals.players.length; i++) {
      const enteredUsername = vals.players[i].username;
      if (!enteredUsername) {
        continue;
      }
      const username = enteredUsername.trim();
      if (username === "") {
        continue;
      }
      players.push({
        id: username,
        rating: Number(vals.players[i].rating) || 0,
      });
    }

    if (players.length === 0) {
      showError("Add some players first");
      return;
    }

    const obj = {
      id: props.tournamentID,
      division: vals.division,
      persons: players,
    };

    try {
      await tClient.addPlayers(obj);
      message.info({
        content: "Players added",
        duration: 3,
      });
    } catch (e) {
      flashError(e);
    }
  };

  return (
    <Form onFinish={onFinish}>
      <DivisionFormItem />
      <Form.Item label="Enter ratings">
        <Switch checked={showRating} onChange={(c) => setShowRating(c)} />
      </Form.Item>

      <Form.List name="players">
        {(fields, { add, remove }) => (
          <>
            {fields.map((field) => (
              <Space
                key={field.key}
                style={{ display: "flex", marginBottom: 8 }}
                align="baseline"
              >
                <Form.Item
                  {...field}
                  name={[field.name, "username"]}
                  rules={[{ required: true, message: "Missing username" }]}
                >
                  <Input placeholder="Username" />
                </Form.Item>
                {showRating && (
                  <Form.Item
                    {...field}
                    name={[field.name, "rating"]}
                    rules={[{ required: true, message: "Missing rating" }]}
                  >
                    <Input placeholder="Rating" />
                  </Form.Item>
                )}
                <MinusCircleOutlined onClick={() => remove(field.name)} />
              </Space>
            ))}
            <Form.Item>
              <Button
                type="dashed"
                onClick={() => add()}
                block
                icon={<PlusOutlined />}
              >
                Add player
              </Button>
            </Form.Item>
          </>
        )}
      </Form.List>

      <Form.Item>
        <Button type="primary" htmlType="submit">
          Submit
        </Button>
      </Form.Item>
    </Form>
  );
};

export const RemovePlayer = (props: { tournamentID: string }) => {
  const [division, setDivision] = useState("");
  const tClient = useClient(TournamentService);
  const onFinish = async (vals: Store) => {
    const obj = {
      id: props.tournamentID,
      division: vals.division,
      persons: [
        {
          id: vals.username,
        },
      ],
    };
    try {
      await tClient.removePlayers(obj);
      message.info({
        content: "Player removed",
        duration: 3,
      });
    } catch (e) {
      flashError(e);
    }
  };

  return (
    <Form onFinish={onFinish}>
      <DivisionFormItem onChange={(div: string) => setDivision(div)} />

      <PlayersFormItem
        name="username"
        label="Username to remove"
        division={division}
        required
      />
      <Form.Item>
        <Button type="primary" htmlType="submit">
          Submit
        </Button>
      </Form.Item>
    </Form>
  );
};

export const MovePlayer = (props: { tournamentID: string }) => {
  const [sourceDivision, setSourceDivision] = useState("");
  const [targetDivision, setTargetDivision] = useState("");
  const { tournamentContext } = useTournamentStoreContext();
  const tClient = useClient(TournamentService);

  const onFinish = async (vals: Store) => {
    const obj = {
      id: props.tournamentID,
      sourceDivision: vals.sourceDivision,
      targetDivision: vals.targetDivision,
      playerId: vals.username,
    };
    try {
      await tClient.movePlayer(obj);
      message.info({
        content: "Player moved successfully",
        duration: 3,
      });
    } catch (e) {
      flashError(e);
    }
  };

  // Get list of divisions excluding the source division for target dropdown
  const targetDivisions = Object.keys(tournamentContext.divisions).filter(
    (div) => div !== sourceDivision,
  );

  return (
    <Form onFinish={onFinish}>
      <DivisionFormItem
        name="sourceDivision"
        label="Source Division"
        onChange={(div: string) => {
          setSourceDivision(div);
          setTargetDivision(""); // Reset target when source changes
        }}
      />

      <PlayersFormItem
        name="username"
        label="Player to move"
        division={sourceDivision}
        required
      />

      <DivisionFormItem
        name="targetDivision"
        label="Target Division"
        onChange={(div: string) => setTargetDivision(div)}
        divisions={targetDivisions}
      />

      <Form.Item>
        <Button type="primary" htmlType="submit">
          Submit
        </Button>
      </Form.Item>
    </Form>
  );
};

/*
const ClearCheckedIn = (props: { tournamentID: string }) => {
  const tClient = useClient(TournamentService);
  const onFinish = async (vals: Store) => {
    const obj = {
      id: props.tournamentID,
    };

    postJsonObj(
      'tournament_service.TournamentService',
      'ClearCheckedIn',
      obj,
      () => {
        message.info({
          content: 'Checked-in cleared',
          duration: 3,
        });
      }
    );
  };

  return (
    <Form onFinish={onFinish}>
      <Form.Item>
        <Button type="primary" htmlType="submit">
          Submit
        </Button>
      </Form.Item>
    </Form>
  );
};
*/
