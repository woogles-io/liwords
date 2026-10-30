// Player management forms

import { MinusCircleOutlined, PlusOutlined } from "@ant-design/icons";
import {
  AutoComplete,
  Button,
  Form,
  Input,
  InputNumber,
  message,
  Space,
  Switch,
} from "antd";
import { Store } from "rc-field-form/lib/interface";
import React, { useState } from "react";
import { TournamentService } from "../../../gen/api/proto/tournament_service/tournament_service_pb";
import { useTournamentStoreContext } from "../../../store/store";
import { flashError, useClient } from "../../../utils/hooks/connect";
import { HelptipLabel } from "../helptip_label";
import { teamNames } from "../pairing_methods";
import {
  DivisionFormItem,
  fullPlayerID,
  PlayersFormItem,
  showError,
} from "./shared";

export const AddPlayers = (props: { tournamentID: string }) => {
  const { tournamentContext } = useTournamentStoreContext();

  const tClient = useClient(TournamentService);
  const [showRating, setShowRating] = useState(
    tournamentContext.metadata.irlMode,
  );
  const [showTeam, setShowTeam] = useState(false);
  const [form] = Form.useForm();
  const division = Form.useWatch("division", form);
  const rows: Array<{ team?: string }> | undefined = Form.useWatch(
    "players",
    form,
  );
  // Suggest the division's teams and the ones typed so far, so a misspelled
  // team is less likely.
  const teamOptions = [
    ...new Set([
      ...teamNames(tournamentContext.divisions[division]?.players ?? []),
      ...(rows ?? []).map((r) => r?.team?.trim()).filter((t) => !!t),
    ]),
  ].map((t) => ({ value: t as string }));
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
        team: (showTeam && vals.players[i].team?.trim()) || "",
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
    <Form form={form} onFinish={onFinish}>
      <DivisionFormItem />
      <Form.Item label="Enter ratings">
        <Switch checked={showRating} onChange={(c) => setShowRating(c)} />
      </Form.Item>
      <Form.Item
        label={
          <HelptipLabel
            labelText="Enter teams"
            help="Only for a team round robin between teams you choose, such as Portland vs Seattle."
          />
        }
      >
        <Switch checked={showTeam} onChange={(c) => setShowTeam(c)} />
      </Form.Item>

      <Form.List name="players">
        {(fields, { add, remove }) => (
          <>
            {/* field carries a key, which would give every input in a row
                the same key, so it's left out of the spread. */}
            {fields.map(({ key, ...field }) => (
              <Space
                key={key}
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
                {showTeam && (
                  <Form.Item {...field} name={[field.name, "team"]}>
                    <AutoComplete
                      placeholder="Team"
                      options={teamOptions}
                      filterOption={(input, option) =>
                        !!option?.value
                          .toLowerCase()
                          .startsWith(input.toLowerCase())
                      }
                      style={{ width: 140 }}
                    />
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

export const EditPlayer = (props: { tournamentID: string }) => {
  const [division, setDivision] = useState("");
  const { tournamentContext } = useTournamentStoreContext();
  const tClient = useClient(TournamentService);
  const [form] = Form.useForm();

  const onValuesChange = (changed: Store) => {
    if (changed.username === undefined) {
      return;
    }
    // Fill in the current rating and team once the director picks a player.
    const divobj = tournamentContext.divisions[division];
    const fullID = fullPlayerID(changed.username, divobj);
    const player = divobj?.players.find((p) => p.id === fullID);
    if (player) {
      form.setFieldsValue({ rating: player.rating, team: player.team });
    }
  };

  const irlMode = tournamentContext.metadata.irlMode;

  const onFinish = async (vals: Store) => {
    const newName = vals.newName?.trim() || undefined;
    const obj = {
      id: props.tournamentID,
      division: vals.division,
      playerId: vals.username,
      rating: vals.rating,
      newName,
      team: vals.team?.trim() ?? "",
    };
    try {
      await tClient.editPlayer(obj);
      message.info({
        content: newName
          ? "Player renamed. Their scorecard QR code still works; reprint scorecards if you want the new name on it."
          : "Player updated",
        duration: newName ? 8 : 3,
      });
      form.resetFields(["username", "newName"]);
    } catch (e) {
      flashError(e);
    }
  };

  return (
    <Form form={form} onFinish={onFinish} onValuesChange={onValuesChange}>
      <DivisionFormItem onChange={(div: string) => setDivision(div)} />

      <PlayersFormItem
        name="username"
        label="Player to edit"
        division={division}
        required
      />

      <Form.Item
        name="rating"
        label={
          <HelptipLabel
            labelText="Rating"
            help="Changing a rating re-seeds the division if the tournament hasn't started yet."
          />
        }
        rules={[{ required: true, message: "Missing rating" }]}
      >
        <InputNumber min={0} />
      </Form.Item>

      <Form.Item
        name="team"
        label={
          <HelptipLabel
            labelText="Team"
            help="Only for a team round robin between teams you choose. Leave blank for no team. Teams can't change once the division starts."
          />
        }
      >
        <AutoComplete
          options={teamNames(
            tournamentContext.divisions[division]?.players ?? [],
          ).map((t) => ({ value: t }))}
          filterOption={(input, option) =>
            !!option?.value.toLowerCase().startsWith(input.toLowerCase())
          }
          allowClear
        />
      </Form.Item>

      {irlMode && (
        <Form.Item
          name="newName"
          label={
            <HelptipLabel
              labelText="New name"
              help="Leave blank to keep the current name. Use this if someone takes over a player's spot."
            />
          }
        >
          <Input />
        </Form.Item>
      )}

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
  const [, setTargetDivision] = useState("");
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
