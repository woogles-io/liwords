// Ghetto tools are Cesar tools before making things pretty.

import {
  DeleteOutlined,
  MinusCircleOutlined,
  PlusOutlined,
} from "@ant-design/icons";
import {
  AutoComplete,
  Button,
  Collapse,
  DatePicker,
  Divider,
  Form,
  Input,
  InputNumber,
  message,
  Popconfirm,
  Select,
  Space,
  Switch,
} from "antd";
import { Modal } from "../../utils/focus_modal";
import { Store } from "rc-field-form/lib/interface";
import React, { useCallback, useEffect, useMemo, useState } from "react";
import {
  SingleRoundControlsRequestSchema,
  TType,
} from "../../gen/api/proto/tournament_service/tournament_service_pb";
import { Division } from "../../store/reducers/tournament_reducer";
import { useTournamentStoreContext } from "../../store/store";
import { DisplayedGameSetting, SettingsForm } from "./game_settings_form";
import "../../lobby/seek_form.scss";

import {
  fieldsForMethod,
  PairingMethodField,
  RoundSetting,
  settingsEqual,
} from "./pairing_methods";
import {
  TournamentGameResult,
  PairingMethod,
  RoundControl,
  FirstMethod,
  TournamentPerson,
  DivisionControlsSchema,
  RoundControlSchema,
  DivisionRoundControlsSchema,
} from "../../gen/api/proto/ipc/tournament_pb";
import {
  GameEndReason,
  GameRequest,
} from "../../gen/api/proto/ipc/omgwords_pb";
import { HelptipLabel } from "./helptip_label";
import { flashError, useClient } from "../../utils/hooks/connect";
import { TournamentService } from "../../gen/api/proto/tournament_service/tournament_service_pb";
import { create, clone } from "@bufbuild/protobuf";
import { getEnumValue } from "../../utils/protobuf";
import {
  dayjsToProtobufTimestampIgnoringNanos,
  doesCurrentUserUse24HourTime,
  protobufTimestampToDayjsIgnoringNanos,
} from "../../utils/datetime";
import { isClubType } from "../../store/constants";
import { singularCount } from "../../utils/plural";

type ModalProps = {
  title: string;
  visible: boolean;
  type: string;
  handleOk: () => void;
  handleCancel: () => void;
  tournamentID: string;
};

const FormModal = (props: ModalProps) => {
  // const [form] = Form.useForm();

  const forms = {
    "add-division": <AddDivision tournamentID={props.tournamentID} />,
    "rename-division": <RenameDivision tournamentID={props.tournamentID} />,
    "remove-division": <RemoveDivision tournamentID={props.tournamentID} />,
    "add-players": <AddPlayers tournamentID={props.tournamentID} />,
    "remove-player": <RemovePlayer tournamentID={props.tournamentID} />,
    // 'clear-checked-in': <ClearCheckedIn tournamentID={props.tournamentID} />,
    "set-single-pairing": <SetPairing tournamentID={props.tournamentID} />,
    "set-game-result": <SetResult tournamentID={props.tournamentID} />,
    "pair-entire-round": <PairRound tournamentID={props.tournamentID} />,
    "unpair-entire-round": <UnpairRound tournamentID={props.tournamentID} />,
    "set-tournament-controls": (
      <SetTournamentControls tournamentID={props.tournamentID} />
    ),
    "set-round-controls": (
      <SetDivisionRoundControls tournamentID={props.tournamentID} />
    ),
    "set-single-round-controls": (
      <SetSingleRoundControls tournamentID={props.tournamentID} />
    ),
    "create-printable-scorecards": (
      <CreatePrintableScorecards tournamentID={props.tournamentID} />
    ),
    "export-tournament": <ExportTournament tournamentID={props.tournamentID} />,
    "edit-description-and-other-settings": (
      <EditDescription tournamentID={props.tournamentID} />
    ),
    "unstart-tournament": (
      <UnstartTournament tournamentID={props.tournamentID} />
    ),
    "unfinish-tournament": (
      <UnfinishTournament tournamentID={props.tournamentID} />
    ),
    "manage-check-ins-and-registrations": (
      <ManageCheckIns tournamentID={props.tournamentID} />
    ),
    "count-firsts": <CountFirsts tournamentID={props.tournamentID} />,
    // "run-cop": <RunCOP tournamentID={props.tournamentID} />,
  };

  type FormKeys = keyof typeof forms;

  return (
    <Modal
      title={props.title}
      open={props.visible}
      footer={null}
      destroyOnClose={true}
      onCancel={props.handleCancel}
      className="seek-modal" // temporary display hack
    >
      {forms[props.type as FormKeys]}
    </Modal>
  );
};

const lowerAndJoin = (v: string): string => {
  const l = v.toLowerCase();
  return l.split(" ").join("-");
};

const showError = (msg: string) => {
  message.error({
    content: "Error: " + msg,
    duration: 5,
  });
};

type Props = {
  tournamentID: string;
};

export const GhettoTools = (props: Props) => {
  const [modalTitle, setModalTitle] = useState("");
  const [modalVisible, setModalVisible] = useState(false);
  const [modalType, setModalType] = useState("");
  const { tournamentContext } = useTournamentStoreContext();

  const showModal = (key: string, title: string) => {
    setModalType(key);
    setModalVisible(true);
    setModalTitle(title);
  };

  const metadataTypes = ["Edit description and other settings"];

  const preTournamentTypes = [
    "Add division",
    "Rename division",
    "Remove division",
    "Set tournament controls",
    "Set round controls",
    "Manage check-ins and registrations",
    "Create printable scorecards",
  ];

  const inTournamentTypes = [
    "Add players",
    "Remove player",
    "Set single round controls", // Set controls for a single round
    "Set single pairing", // Set a single pairing
    "Pair entire round", // Pair a whole round
    "Set game result", // Set a single result
    "Unpair entire round", // Unpair a whole round
    // "Run COP", // experimental run COP (new pairing method)
    // 'Clear checked in',
    "Count firsts",
  ];

  const postTournamentTypes = ["Export tournament", "Unfinish tournament"];

  const dangerousTypes = ["Unstart tournament"];

  const mapFn = (v: string) => {
    const key = lowerAndJoin(v);
    return (
      <li key={key} style={{ marginBottom: 5 }}>
        <Button onClick={() => showModal(key, v)} size="small">
          {v}
        </Button>
      </li>
    );
  };

  const metadataItems = metadataTypes.map(mapFn);
  const preListItems = preTournamentTypes.map(mapFn);
  const inListItems = inTournamentTypes.map(mapFn);
  const postListItems = postTournamentTypes.map(mapFn);
  const dangerListItems = dangerousTypes.map(mapFn);

  return (
    <>
      <h3>Tournament Tools</h3>
      <h4>Edit tournament metadata</h4>
      <ul>{metadataItems}</ul>
      {(tournamentContext.metadata.type === TType.STANDARD ||
        tournamentContext.metadata.type === TType.CHILD) && (
        <>
          <h4>Pre-tournament settings</h4>
          <ul>{preListItems}</ul>
          <Divider />
          <h4>In-tournament management</h4>
          <ul>{inListItems}</ul>
          <Divider />
          <h4>Post-tournament utilities</h4>
          <ul>{postListItems}</ul>
          <Divider />
          <h4>Danger!</h4>
          <ul>{dangerListItems}</ul>
          <Divider />
        </>
      )}
      <FormModal
        title={modalTitle}
        visible={modalVisible}
        type={modalType}
        handleOk={() => setModalVisible(false)}
        handleCancel={() => setModalVisible(false)}
        tournamentID={props.tournamentID}
      />
    </>
  );
};

const DivisionSelector = (props: {
  onChange?: (value: string) => void;
  value?: string;
  exclude?: Array<string>;
}) => {
  const { tournamentContext } = useTournamentStoreContext();
  return (
    <Select onChange={props.onChange} value={props.value}>
      {Object.keys(tournamentContext.divisions)
        .filter((d) => !props.exclude?.includes(d))
        .map((d) => (
          <Select.Option value={d} key={`div-${d}`}>
            {d}
          </Select.Option>
        ))}
    </Select>
  );
};

const DivisionFormItem = (props: {
  onChange?: (value: string) => void;
  value?: string;
}) => {
  return (
    <Form.Item
      name="division"
      label="Division Name"
      rules={[
        {
          required: true,
          message: "Please input division name",
        },
      ]}
    >
      <DivisionSelector onChange={props.onChange} value={props.value} />
    </Form.Item>
  );
};

const PlayersFormItem = (props: {
  name: string;
  label: string;
  division: string;
  required?: boolean;
}) => {
  const { tournamentContext } = useTournamentStoreContext();
  const thisDivPlayers = tournamentContext.divisions[props.division]?.players;
  const alphabetizedOptions = useMemo(() => {
    if (!thisDivPlayers) {
      return null;
    }
    const playersCopy = [...thisDivPlayers];
    const players = playersCopy.sort(
      (a: TournamentPerson, b: TournamentPerson) => {
        const usera = username(a.id);
        const userb = username(b.id);
        if (usera < userb) {
          return -1;
        } else if (usera > userb) {
          return 1;
        }
        return 0;
      },
    );
    return players.map((u) => ({ value: username(u.id) }));
  }, [thisDivPlayers]);

  return (
    <Form.Item
      name={props.name}
      label={props.label}
      required={props.required ?? false}
    >
      {alphabetizedOptions ? (
        <AutoComplete
          options={alphabetizedOptions}
          filterOption={(inputValue, option) =>
            option?.value.toUpperCase().indexOf(inputValue.toUpperCase()) !== -1
          }
        />
      ) : null}
    </Form.Item>
  );
};

const AddDivision = (props: { tournamentID: string }) => {
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

const RenameDivision = (props: { tournamentID: string }) => {
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

const RemoveDivision = (props: { tournamentID: string }) => {
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

const AddPlayers = (props: { tournamentID: string }) => {
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

const RemovePlayer = (props: { tournamentID: string }) => {
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

// userUUID looks up the UUID of a username
const userUUID = (username: string, divobj: Division) => {
  if (!divobj) {
    return "";
  }
  const p = divobj.players.find((p) => {
    const parts = p.id.split(":");
    const pusername = parts[1].toLowerCase();

    if (username.toLowerCase() === pusername) {
      return true;
    }
    return false;
  });
  if (!p) {
    return "";
  }
  return p.id.split(":")[0];
};

const username = (fullID: string) => {
  const parts = fullID.split(":");
  return parts[1];
};

const fullPlayerID = (username: string, divobj: Division) => {
  if (!divobj) {
    return "";
  }
  const p = divobj.players.find((p) => {
    const parts = p.id.split(":");
    const pusername = parts[1].toLowerCase();

    if (username.toLowerCase() === pusername) {
      return true;
    }
    return false;
  });
  if (!p) {
    return "";
  }
  return p.id;
};

const SetPairing = (props: { tournamentID: string }) => {
  const { tournamentContext } = useTournamentStoreContext();
  const [division, setDivision] = useState("");
  const [selfplay, setSelfplay] = useState(false);
  const tClient = useClient(TournamentService);
  const onFinish = async (vals: Store) => {
    if (!vals.p1) {
      message.error("Player 1 is required.");
      return;
    }
    if (!vals.selfplay && !vals.p2) {
      message.error("Player 2 is required.");
      return;
    }

    const p1id = fullPlayerID(
      vals.p1,
      tournamentContext.divisions[vals.division],
    );
    let p2id;

    if (vals.selfplay) {
      p2id = p1id;
      if (!vals.selfplayresult) {
        message.error("Desired result for Player 1 is required.");
        return;
      }
    } else {
      p2id = fullPlayerID(vals.p2, tournamentContext.divisions[vals.division]);
    }

    const obj = {
      id: props.tournamentID,
      division: vals.division,
      pairings: [
        {
          playerOneId: p1id,
          playerTwoId: p2id,
          round: vals.round - 1, // 1-indexed input
          // use self-play result only if it was set.
          selfPlayResult: vals.selfplay ? vals.selfplayresult : undefined,
        },
      ],
    };
    try {
      await tClient.setPairing(obj);
      message.info({
        content: "Pairing set",
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
        name="p1"
        label="Player 1 username"
        division={division}
        required
      />

      <Form.Item name="selfplay" label="Player has no opponent">
        <Switch checked={selfplay} onChange={(c) => setSelfplay(c)} />
      </Form.Item>

      {!selfplay ? (
        <PlayersFormItem
          name="p2"
          label="Player 2 username"
          division={division}
          required
        />
      ) : (
        <Form.Item
          name="selfplayresult"
          label="Desired result for this player"
          required
        >
          <Select>
            <Select.Option value={TournamentGameResult.BYE}>
              Bye (+50)
            </Select.Option>
            <Select.Option value={TournamentGameResult.FORFEIT_LOSS}>
              Forfeit loss (-50)
            </Select.Option>
            <Select.Option value={TournamentGameResult.VOID}>
              Void (no record change)
            </Select.Option>
          </Select>
        </Form.Item>
      )}

      <Form.Item name="round" label="Round (1-indexed)">
        <InputNumber inputMode="numeric" min={1} required />
      </Form.Item>

      <Form.Item>
        <Button type="primary" htmlType="submit">
          Submit
        </Button>
      </Form.Item>
    </Form>
  );
};

const SetResult = (props: { tournamentID: string }) => {
  const { tournamentContext } = useTournamentStoreContext();
  const [division, setDivision] = useState("");
  const [score1, setScore1] = useState(0);
  const [score2, setScore2] = useState(0);
  const [form] = Form.useForm();
  const tClient = useClient(TournamentService);
  const onFinish = async (vals: Store) => {
    const obj = {
      id: props.tournamentID,
      division: vals.division,
      playerOneId: userUUID(
        vals.p1,
        tournamentContext.divisions[vals.division],
      ),
      playerTwoId: userUUID(
        vals.p2,
        tournamentContext.divisions[vals.division],
      ),
      round: vals.round - 1, // 1-indexed input
      playerOneScore: vals.p1score,
      playerTwoScore: vals.p2score,
      playerOneResult: getEnumValue(TournamentGameResult, vals.p1result),
      playerTwoResult: getEnumValue(TournamentGameResult, vals.p2result),
      gameEndReason: getEnumValue(GameEndReason, vals.gameEndReason),
      amendment: vals.amendment,
    };
    try {
      await tClient.setResult(obj);
      message.info({
        content: "Result set",
        duration: 3,
      });
    } catch (e) {
      flashError(e);
    }
  };

  useEffect(() => {
    if (score1 > score2) {
      form.setFieldsValue({
        p1result: "WIN",
        p2result: "LOSS",
      });
    } else if (score1 < score2) {
      form.setFieldsValue({
        p1result: "LOSS",
        p2result: "WIN",
      });
    } else {
      form.setFieldsValue({
        p1result: "DRAW",
        p2result: "DRAW",
      });
    }
  }, [form, score1, score2]);

  const score1Change = (v: number | string | null | undefined) => {
    if (typeof v !== "number") {
      return;
    }
    setScore1(v);
  };
  const score2Change = (v: number | string | null | undefined) => {
    if (typeof v !== "number") {
      return;
    }
    setScore2(v);
  };

  return (
    <Form
      form={form}
      onFinish={onFinish}
      initialValues={{ gameEndReason: "STANDARD" }}
    >
      <DivisionFormItem onChange={(div: string) => setDivision(div)} />

      <PlayersFormItem
        name="p1"
        label="Player 1 username"
        division={division}
        required
      />
      <PlayersFormItem
        name="p2"
        label="Player 2 username"
        division={division}
        required
      />

      <Form.Item name="round" label="Round (1-indexed)">
        <InputNumber inputMode="numeric" min={1} required />
      </Form.Item>

      <Form.Item name="p1score" label="Player 1 score">
        <InputNumber
          inputMode="numeric"
          onChange={score1Change}
          value={score1}
        />
      </Form.Item>

      <Form.Item name="p2score" label="Player 2 score">
        <InputNumber
          inputMode="numeric"
          onChange={score2Change}
          value={score2}
        />
      </Form.Item>

      <Form.Item name="p1result" label="Player 1 result">
        <Select>
          <Select.Option value="VOID">VOID (no win or loss)</Select.Option>
          <Select.Option value="WIN">WIN</Select.Option>
          <Select.Option value="LOSS">LOSS</Select.Option>
          <Select.Option value="DRAW">DRAW</Select.Option>
          <Select.Option value="BYE">BYE</Select.Option>
          <Select.Option value="FORFEIT_WIN">FORFEIT_WIN</Select.Option>
          <Select.Option value="FORFEIT_LOSS">FORFEIT_LOSS</Select.Option>
        </Select>
      </Form.Item>

      <Form.Item name="p2result" label="Player 2 result">
        <Select>
          <Select.Option value="VOID">VOID</Select.Option>
          <Select.Option value="WIN">WIN</Select.Option>
          <Select.Option value="LOSS">LOSS</Select.Option>
          <Select.Option value="DRAW">DRAW</Select.Option>
          <Select.Option value="BYE">BYE</Select.Option>
          <Select.Option value="FORFEIT_WIN">FORFEIT_WIN</Select.Option>
          <Select.Option value="FORFEIT_LOSS">FORFEIT_LOSS</Select.Option>
          <Select.Option value="NO_RESULT">NO_RESULT</Select.Option>
        </Select>
      </Form.Item>

      <Form.Item name="gameEndReason" label="Game End Reason">
        <Select>
          <Select.Option value="NONE">NONE</Select.Option>
          <Select.Option value="TIME">TIME</Select.Option>
          <Select.Option value="STANDARD">STANDARD</Select.Option>
          <Select.Option value="CONSECUTIVE_ZEROES">
            CONSECUTIVE_ZEROES
          </Select.Option>
          <Select.Option value="RESIGNED">RESIGNED</Select.Option>
          <Select.Option value="ABORTED">ABORTED</Select.Option>
          <Select.Option value="TRIPLE_CHALLENGE">
            TRIPLE_CHALLENGE
          </Select.Option>
          <Select.Option value="CANCELLED">CANCELLED</Select.Option>
          <Select.Option value="FORCE_FORFEIT">FORCE_FORFEIT</Select.Option>
        </Select>
      </Form.Item>

      <Form.Item name="amendment" label="Amendment" valuePropName="checked">
        <Switch />
      </Form.Item>

      <Form.Item>
        <Button type="primary" htmlType="submit">
          Submit
        </Button>
      </Form.Item>
    </Form>
  );
};

const PairRound = (props: { tournamentID: string }) => {
  const tClient = useClient(TournamentService);
  const onFinish = async (vals: Store) => {
    const obj = {
      id: props.tournamentID,
      division: vals.division,
      round: vals.round - 1, // 1-indexed input
      preserveByes: vals.preserveByes,
    };
    try {
      await tClient.pairRound(obj);
      message.info({
        content: "Pair round completed",
        duration: 3,
      });
    } catch (e) {
      flashError(e);
    }
  };

  return (
    <Form onFinish={onFinish}>
      <DivisionFormItem />

      <Form.Item name="round" label="Round (1-indexed)">
        <InputNumber inputMode="numeric" min={1} required />
      </Form.Item>

      <Form.Item name="preserveByes" label="Preserve byes">
        <Switch />
      </Form.Item>

      <Form.Item>
        <Button type="primary" htmlType="submit">
          Submit
        </Button>
      </Form.Item>
    </Form>
  );
};

const UnpairRound = (props: { tournamentID: string }) => {
  const tClient = useClient(TournamentService);
  const onFinish = async (vals: Store) => {
    const obj = {
      id: props.tournamentID,
      division: vals.division,
      round: vals.round - 1, // 1-indexed input
      deletePairings: true,
    };
    try {
      await tClient.pairRound(obj);
      message.info({
        content: "Pairings for selected round have been deleted",
        duration: 3,
      });
    } catch (e) {
      flashError(e);
    }
  };
  return (
    <Form onFinish={onFinish}>
      <DivisionFormItem />
      <Form.Item name="round" label="Round (1-indexed)">
        <InputNumber inputMode="numeric" min={1} required />
      </Form.Item>
      <Form.Item>
        <Button type="primary" htmlType="submit">
          Submit
        </Button>
      </Form.Item>
    </Form>
  );
};

const SetTournamentControls = (props: { tournamentID: string }) => {
  const [modalVisible, setModalVisible] = useState(false);
  const [selectedGameRequest, setSelectedGameRequest] = useState<
    GameRequest | undefined
  >(undefined);

  const [division, setDivision] = useState("");
  const [copyFromDivision, setCopyFromDivision] = useState("");
  const [gibsonize, setGibsonize] = useState(false);
  const [gibsonSpread, setGibsonSpread] = useState(500);

  // min placement is 0-indexed, but we want to display 1-indexed
  // this variable will be the display variable:
  const [gibsonMinPlacement, setGibsonMinPlacement] = useState(1);
  // bye max placement is 0-indexed, this is also the display variable
  const [byeMaxPlacement, setByeMaxPlacement] = useState(1);
  const [spreadCap, setSpreadCap] = useState(0);
  const [suspendedResult, setSuspendedResult] = useState<TournamentGameResult>(
    TournamentGameResult.FORFEIT_LOSS,
  );
  const { tournamentContext } = useTournamentStoreContext();

  useEffect(() => {
    if (!division) {
      setSelectedGameRequest(undefined);
      return;
    }
    const div = tournamentContext.divisions[division];
    const gameRequest = div.divisionControls?.gameRequest;
    if (gameRequest) {
      setSelectedGameRequest(gameRequest);
    } else {
      setSelectedGameRequest(undefined);
    }
    if (div.divisionControls) {
      setGibsonize(div.divisionControls.gibsonize);
      setGibsonSpread(div.divisionControls.gibsonSpread);
      setGibsonMinPlacement(div.divisionControls.minimumPlacement + 1);
      setByeMaxPlacement(div.divisionControls.maximumByePlacement + 1);
      setSuspendedResult(div.divisionControls.suspendedResult);
      setSpreadCap(div.divisionControls.spreadCap);
    }
  }, [division, tournamentContext.divisions]);

  const SettingsModalForm = (mprops: {
    visible: boolean;
    onCancel: () => void;
  }) => {
    return (
      <Modal
        title="Set Game Request"
        open={mprops.visible}
        onCancel={mprops.onCancel}
        className="seek-modal"
        okButtonProps={{ style: { display: "none" } }}
      >
        <SettingsForm
          setGameRequest={(gr) => {
            setSelectedGameRequest(gr);
            setModalVisible(false);
          }}
          gameRequest={selectedGameRequest}
        />
      </Modal>
    );
  };

  const tClient = useClient(TournamentService);

  const submit = async () => {
    if (!selectedGameRequest) {
      showError("No game request");
      return;
    }
    const ctrls = create(DivisionControlsSchema, {
      id: props.tournamentID,
      division,
      gameRequest: selectedGameRequest,
      // can set this later to whatever values, along with a spread
      suspendedResult,
      autoStart: false,
      gibsonize,
      gibsonSpread,
      minimumPlacement: gibsonMinPlacement - 1,
      maximumByePlacement: byeMaxPlacement - 1,
      spreadCap: spreadCap,
    });

    if (suspendedResult === TournamentGameResult.BYE) {
      ctrls.suspendedSpread = 50;
    } else if (suspendedResult === TournamentGameResult.FORFEIT_LOSS) {
      ctrls.suspendedSpread = -50;
    }

    try {
      await tClient.setDivisionControls(ctrls);
      message.info({
        content: "Controls set",
        duration: 3,
      });
    } catch (e) {
      flashError(e);
    }
  };

  const formItemLayout = {
    labelCol: {
      span: 10,
    },
    wrapperCol: {
      span: 12,
    },
  };

  const SuspendedGameResultHelptip = (
    <>
      What result would you like to assign to players who join your tournament
      late, for unplayed rounds?<p>&nbsp;</p>
      <p>- Recommended value for tournaments is Forfeit loss. </p>
      <p>- Clubs can probably use a Void result.</p>
    </>
  );

  const copySettings = React.useCallback(() => {
    // copy settings from copyFromDivision to division
    const cd = tournamentContext.divisions[copyFromDivision];
    if (!cd.divisionControls) {
      return;
    }
    const cdCopy = clone(DivisionControlsSchema, cd.divisionControls);
    setSelectedGameRequest(cdCopy.gameRequest);
    setSuspendedResult(cdCopy.suspendedResult);
    setGibsonize(cdCopy.gibsonize);
    setGibsonSpread(cdCopy.gibsonSpread);
    setSpreadCap(cdCopy.spreadCap);
    // These are display variables so add 1 since they're 0-indexed:
    setGibsonMinPlacement(cdCopy.minimumPlacement + 1);
    setByeMaxPlacement(cdCopy.maximumByePlacement + 1);
  }, [copyFromDivision, tournamentContext.divisions]);

  return (
    <>
      <Form>
        <Form.Item {...formItemLayout} label="Division">
          <DivisionSelector
            value={division}
            onChange={(value: string) => {
              setDivision(value);
              setCopyFromDivision("");
            }}
          />
        </Form.Item>

        {Object.keys(tournamentContext.divisions).length > 1 &&
          division !== "" &&
          selectedGameRequest == null && (
            <Collapse style={{ marginBottom: 10 }}>
              <Collapse.Panel header="Copy from division" key="copyFrom">
                <p className="readable-text-color" style={{ marginBottom: 10 }}>
                  Copy from existing division:
                  <HelptipLabel
                    labelText=""
                    help="If you want to copy settings from another division, select
                that division and click the Copy button. Note that you must still
                click Save tournament controls to save your settings after copying them."
                  />
                </p>
                <DivisionSelector
                  value={copyFromDivision}
                  onChange={(value: string) => setCopyFromDivision(value)}
                  exclude={[division]}
                />
                <Button onClick={() => copySettings()}>
                  Copy from {copyFromDivision}
                </Button>
                <p className="readable-text-color" style={{ marginTop: 10 }}>
                  Or, set from scratch:
                </p>
              </Collapse.Panel>
            </Collapse>
          )}

        <Form.Item
          {...formItemLayout}
          label={
            <HelptipLabel
              labelText="Gibsonize"
              help="If Gibsonize is on, players who have won the tournament before it is over will be paired against players not in contention."
            />
          }
        >
          <Switch
            checked={gibsonize}
            onChange={(c: boolean) => setGibsonize(c)}
          />
        </Form.Item>

        <Form.Item
          {...formItemLayout}
          label={
            <HelptipLabel
              labelText="Gibson spread"
              help="Gibson spread is used to determine whether a player should be Gibsonized. With one round to go, if the first-place player is one win and this much spread ahead of second place, they will be Gibsonized."
            />
          }
        >
          <InputNumber
            inputMode="numeric"
            min={0}
            value={gibsonSpread}
            onChange={(v: number | string | undefined | null) =>
              setGibsonSpread(v as number)
            }
          />
        </Form.Item>

        <Form.Item
          {...formItemLayout}
          label={
            <HelptipLabel
              labelText="Gibson min placement"
              help="If Gibsonize is on, you typically want this number to be at least 2. This number should be the number of places that have prizes."
            />
          }
        >
          <InputNumber
            inputMode="numeric"
            min={1}
            value={gibsonMinPlacement}
            onChange={(p: number | string | undefined | null) =>
              setGibsonMinPlacement(p as number)
            }
          />
        </Form.Item>

        <Form.Item
          {...formItemLayout}
          label={
            <HelptipLabel
              help="Byes may be assigned to players ranked this, and worse,
          if odd. Make this 1 if you wish everyone in the tournament to be eligible for byes."
              labelText="Bye cut-off"
            />
          }
        >
          <InputNumber
            inputMode="numeric"
            min={1}
            value={byeMaxPlacement}
            onChange={(p: number | string | undefined | null) =>
              setByeMaxPlacement(p as number)
            }
          />
        </Form.Item>

        <Form.Item
          {...formItemLayout}
          label={
            <HelptipLabel
              help="Limit spread from losses to this number. If set to 0, there is no spread cap."
              labelText="Spread cap"
            />
          }
        >
          <InputNumber
            inputMode="numeric"
            min={0}
            value={spreadCap}
            onChange={(p: number | string | undefined | null) =>
              setSpreadCap(p as number)
            }
          />
        </Form.Item>

        <Form.Item
          {...formItemLayout}
          label={
            <HelptipLabel
              help={SuspendedGameResultHelptip}
              labelText="Suspended game result"
            />
          }
        >
          <Select
            value={suspendedResult}
            onChange={(v) => setSuspendedResult(v)}
          >
            <Select.Option value={TournamentGameResult.NO_RESULT}>
              Please select an option
            </Select.Option>
            <Select.Option value={TournamentGameResult.FORFEIT_LOSS}>
              Forfeit loss (-50)
            </Select.Option>
            <Select.Option value={TournamentGameResult.BYE}>
              Bye +50
            </Select.Option>
            <Select.Option value={TournamentGameResult.VOID}>
              Void (No win or loss)
            </Select.Option>
          </Select>
        </Form.Item>
      </Form>

      <div>{DisplayedGameSetting(selectedGameRequest)}</div>

      <Button
        htmlType="button"
        style={{
          margin: "0 8px",
        }}
        onClick={() => setModalVisible(true)}
      >
        Edit game settings
      </Button>
      <Button type="primary" onClick={submit}>
        Save tournament controls
      </Button>

      <SettingsModalForm
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
      />
    </>
  );
};

type RdCtrlFieldsProps = {
  setting: RoundSetting;
  onChange: (fieldName: string, value: string | number | boolean) => void;
  onRemove: () => void;
};

type SingleRdCtrlFieldsProps = {
  setting: RoundControl;
  onChange: (
    fieldName: keyof RoundControl,
    value: string | number | boolean | PairingMethod,
  ) => void;
};

const SingleRoundControlFields = (props: SingleRdCtrlFieldsProps) => {
  const { setting } = props;
  const addlFields = fieldsForMethod(setting.pairingMethod);

  const formItemLayout = {
    labelCol: {
      span: 6,
    },
    wrapperCol: {
      span: 8,
    },
  };

  const pairingTypesHelptip = (
    <>
      <ul>
        <li>
          - <strong>Random:</strong> Pairings are random. This is only
          recommended for the very first round.
        </li>
        <li>
          - <strong>Swiss:</strong> Swiss pairings by default try to match
          players who are performing similarly.
        </li>
        <li>
          - <strong>Round Robin:</strong> These pairings match everyone in the
          division against each other. If there are fewer rounds than players,
          it will do a partial round robin.
        </li>
        <li>
          - <strong>Initial Fontes:</strong> These pairings split up the field
          into groups of size N+1, and pair everyone in the group against each
          other. The number you provide (N) must be an odd number. This should
          be used at the beginning of a tournament.
        </li>
        <li>
          - <strong>Shirts and Skins:</strong> These pairings split the division
          into two halves, by alternating seeds. Each player in one half plays a
          round robin against every player in the other half.
        </li>
        <li>
          - <strong>King of the hill:</strong> These pairings pair 1v2, 3v4,
          5v6, and so forth. It is a good format for the very last round of a
          tournament.
        </li>
        <li>
          - <strong>Factor:</strong> Factor 1 pairs 1 vs 2 and the rest Swiss.
          Factor 2 pairs 1v3, 2v4, and the rest Swiss. Factor 3 pairs 1v4, 2v5,
          3v6 and the rest Swiss, and so on.
        </li>
        <li>
          - <strong>Manual:</strong> Manual pairings must be provided manually
          by the director every round. This is not recommended except for the
          smallest tournaments.
        </li>
        <li>
          - <strong>Team Round Robin:</strong> Set up a round robin where each
          "team" member plays each other team member some set number of times in
          a row. You must divide teams into top and bottom halves, by "rating",
          and you must have an even number of players.
        </li>
      </ul>
    </>
  );

  return (
    <>
      <Form.Item
        {...formItemLayout}
        label={
          <HelptipLabel labelText="Pairing type" help={pairingTypesHelptip} />
        }
      >
        <Select
          value={setting.pairingMethod}
          onChange={(e) => {
            props.onChange("pairingMethod", e);
            // Show more fields potentially.
          }}
        >
          <Select.Option value={PairingMethod.RANDOM}>Random</Select.Option>
          <Select.Option value={PairingMethod.SWISS}>Swiss</Select.Option>

          <Select.Option value={PairingMethod.ROUND_ROBIN}>
            Round Robin
          </Select.Option>
          <Select.Option value={PairingMethod.INITIAL_FONTES}>
            Initial Fontes
          </Select.Option>
          <Select.Option value={PairingMethod.KING_OF_THE_HILL}>
            King of the Hill
          </Select.Option>
          <Select.Option value={PairingMethod.INTERLEAVED_ROUND_ROBIN}>
            Shirts and Skins Round Robin
          </Select.Option>
          <Select.Option value={PairingMethod.SNAKED_ROUND_ROBIN}>
            Snaked (Boustrophedonic) Round Robin
          </Select.Option>
          <Select.Option value={PairingMethod.FACTOR}>Factor</Select.Option>
          <Select.Option value={PairingMethod.MANUAL}>Manual</Select.Option>
          <Select.Option value={PairingMethod.TEAM_ROUND_ROBIN}>
            Team (Top vs Bottom) Round Robin
          </Select.Option>
        </Select>
      </Form.Item>
      <p></p>
      {/* potential additional fields */}
      {addlFields.map((v: PairingMethodField, idx) => {
        const key = `ni-${idx}`;
        const [fieldType, fieldName, displayName, help] = v;
        switch (fieldType) {
          case "number":
            return (
              <Form.Item
                {...formItemLayout}
                labelCol={{ span: 16, offset: 1 }}
                label={<HelptipLabel labelText={displayName} help={help} />}
                key={`${idx}-${fieldName}`}
              >
                <InputNumber
                  inputMode="numeric"
                  key={key}
                  min={0}
                  value={setting[fieldName] as number}
                  onChange={(e) => {
                    props.onChange(fieldName, e as number);
                  }}
                />
              </Form.Item>
            );

          case "boolean":
            return (
              <Form.Item
                {...formItemLayout}
                labelCol={{ span: 12, offset: 1 }}
                label={<HelptipLabel labelText={displayName} help={help} />}
                key={`${idx}-${fieldName}`}
              >
                <Switch
                  key={key}
                  checked={setting[fieldName] as boolean}
                  onChange={(e) => props.onChange(fieldName, e)}
                />
              </Form.Item>
            );
        }
        return null;
      })}
    </>
  );
};

const RoundControlFields = (props: RdCtrlFieldsProps) => {
  const { setting } = props;
  return (
    <>
      <Form size="small">
        <Form.Item label="First round">
          <InputNumber
            inputMode="numeric"
            min={1}
            value={setting.beginRound}
            onChange={(e) => props.onChange("beginRound", e as number)}
          />
        </Form.Item>
        <Form.Item label="Last round">
          <InputNumber
            inputMode="numeric"
            min={1}
            value={setting.endRound}
            onChange={(e) => props.onChange("endRound", e as number)}
          />
        </Form.Item>
      </Form>
      <Form size="small" style={{ marginTop: 8 }}>
        <SingleRoundControlFields
          setting={setting.setting}
          onChange={props.onChange}
        />
      </Form>
      <Button onClick={props.onRemove}>- Remove</Button>
      <Divider />
    </>
  );
};

const rdCtrlFromSetting = (rdSetting: RoundControl): RoundControl => {
  const rdCtrl = create(RoundControlSchema, {
    firstMethod: FirstMethod.AUTOMATIC_FIRST,
    gamesPerRound: 1,
    pairingMethod: rdSetting.pairingMethod,
  });

  switch (rdSetting.pairingMethod) {
    case PairingMethod.SWISS:
    case PairingMethod.FACTOR:
      rdCtrl.maxRepeats = rdSetting.maxRepeats || 0;
      rdCtrl.allowOverMaxRepeats = true;
      rdCtrl.repeatRelativeWeight = rdSetting.repeatRelativeWeight || 0;
      rdCtrl.winDifferenceRelativeWeight =
        rdSetting.winDifferenceRelativeWeight || 0;
      // This should be auto-calculated, and only for factor
      if (
        rdSetting.pairingMethod === PairingMethod.FACTOR &&
        rdSetting.factor <= 0
      ) {
        throw new Error(
          "Factor 0 is equivalent to just Swiss for every player. Use Swiss pairings instead, or use a Factor greater than 0.",
        );
      }
      rdCtrl.factor = rdSetting.factor || 0;

      if (rdCtrl.maxRepeats <= 0) {
        throw new Error(
          "Max Pairings Between Any Two Players should be at least 1. Setting it to 0 will allow no pairings to occur.",
        );
      }

      if (rdCtrl.repeatRelativeWeight <= 0) {
        throw new Error(
          "Repeat relative weight should be at least 1. Please hover on the question mark to see more info about what this means.",
        );
      }

      if (rdCtrl.winDifferenceRelativeWeight <= 0) {
        throw new Error(
          "Win difference relative weight should be at least 1. Please hover on the question mark to see more info about what this means.",
        );
      }

      if (rdCtrl.repeatRelativeWeight > 100) {
        throw new Error(
          "Repeat relative weight should be at most 100. Please hover on the question mark to see more info about what this means.",
        );
      }

      if (rdCtrl.winDifferenceRelativeWeight > 100) {
        throw new Error(
          "Win difference relative weight should be at most 100. Please hover on the question mark to see more info about what this means.",
        );
      }

      break;

    case PairingMethod.TEAM_ROUND_ROBIN:
      rdCtrl.gamesPerRound = rdSetting.gamesPerRound || 1;
      break;
  }
  // Other cases don't matter, we've already set the pairing method.
  return rdCtrl;
};

const SetSingleRoundControls = (props: { tournamentID: string }) => {
  const [division, setDivision] = useState("");
  const [roundSetting, setRoundSetting] = useState<RoundControl>(
    create(RoundControlSchema, {
      pairingMethod: PairingMethod.RANDOM,
    }),
  );
  const [userVisibleRound, setUserVisibleRound] = useState(1);
  const tClient = useClient(TournamentService);
  const setRoundControls = async () => {
    if (!division) {
      showError("Division is missing");
      return;
    }
    if (userVisibleRound <= 0) {
      showError("Round must be a positive round number");
      return;
    }
    if (!roundSetting) {
      showError("Missing round setting");
      return;
    }

    const ctrls = create(SingleRoundControlsRequestSchema, {
      id: props.tournamentID,
      division: division,
      round: userVisibleRound - 1, // round is 0-indexed on backend.
    });
    let rdCtrl;
    try {
      rdCtrl = rdCtrlFromSetting(roundSetting);
    } catch (e) {
      message.error({
        content: (e as Error).message,
        duration: 5,
      });
      return;
    }
    ctrls.roundControls = rdCtrl;
    try {
      await tClient.setSingleRoundControls(ctrls);
      message.info({
        content: `Controls set for round ${userVisibleRound}`,
        duration: 3,
      });
    } catch (e) {
      flashError(e);
    }
  };

  const formItemLayout = {
    labelCol: {
      span: 7,
    },
    wrapperCol: {
      span: 12,
    },
  };

  return (
    <>
      <Form>
        <Form.Item {...formItemLayout} label="Division">
          <DivisionSelector
            value={division}
            onChange={(value: string) => setDivision(value)}
          />
        </Form.Item>
        <Form.Item {...formItemLayout} label="Round">
          <InputNumber
            inputMode="numeric"
            value={userVisibleRound}
            onChange={(e) => e && setUserVisibleRound(e as number)}
          />
        </Form.Item>
      </Form>
      <Divider />
      <Form>
        <SingleRoundControlFields
          setting={roundSetting}
          onChange={(
            fieldName: keyof RoundControl,
            value: string | number | boolean | PairingMethod,
          ) => {
            const val = { ...roundSetting, [fieldName]: value };
            setRoundSetting(create(RoundControlSchema, val));
          }}
        />
        <Form.Item>
          <Button type="primary" onClick={() => setRoundControls()}>
            Submit
          </Button>
        </Form.Item>
      </Form>
    </>
  );
};

const SetDivisionRoundControls = (props: { tournamentID: string }) => {
  const { tournamentContext } = useTournamentStoreContext();
  // This form is too complicated to use the Ant Design built-in forms;
  // So we're just going to use form components instead.

  const [roundArray, setRoundArray] = useState<Array<RoundSetting>>([]);
  const [division, setDivision] = useState("");
  const [copyFromDivision, setCopyFromDivision] = useState("");

  const roundControlsToDisplayArray = React.useCallback(
    (roundControls: RoundControl[]) => {
      const settings = new Array<RoundSetting>();

      let lastSetting: RoundControl | null = null;
      let min = 1;
      let max = 1;
      roundControls.forEach((v: RoundControl, rd: number) => {
        const thisSetting = create(RoundControlSchema, {
          pairingMethod: v.pairingMethod,
          gamesPerRound: v.gamesPerRound,
          factor: v.factor,
          maxRepeats: v.maxRepeats,
          allowOverMaxRepeats: v.allowOverMaxRepeats,
          repeatRelativeWeight: v.repeatRelativeWeight,
          winDifferenceRelativeWeight: v.winDifferenceRelativeWeight,
        });
        if (lastSetting !== null) {
          if (settingsEqual(lastSetting, thisSetting)) {
            max = rd + 1;
          } else {
            settings.push({
              beginRound: min,
              endRound: max,
              setting: lastSetting,
            });
            min = max + 1;
            max = max + 1;
          }
        }
        lastSetting = thisSetting;
      });

      if (lastSetting !== null) {
        settings.push({
          beginRound: min,
          endRound: max,
          setting: lastSetting,
        });
      }
      return settings;
    },
    [],
  );

  useEffect(() => {
    if (!division) {
      setRoundArray([]);
      return;
    }
    const div = tournamentContext.divisions[division];
    setRoundArray(roundControlsToDisplayArray(div.roundControls));
  }, [division, roundControlsToDisplayArray, tournamentContext.divisions]);
  const tClient = useClient(TournamentService);

  const setRoundControls = async () => {
    if (!division) {
      showError("Division is missing");
      return;
    }
    if (!roundArray.length) {
      showError("Round controls are missing");
      return;
    }
    // validate round array
    let lastRd = 0;
    for (let i = 0; i < roundArray.length; i++) {
      const rdCtrl = roundArray[i];
      if (rdCtrl.beginRound <= lastRd) {
        showError("Round numbers must be consecutive and increasing");
        return;
      }
      if (rdCtrl.endRound < rdCtrl.beginRound) {
        showError("End round must not be smaller than begin round");
        return;
      }
      if (rdCtrl.beginRound > lastRd + 1) {
        showError("Round numbers must be consecutive; you cannot skip rounds");
        return;
      }
      lastRd = rdCtrl.endRound;
    }

    const ctrls = create(DivisionRoundControlsSchema, {
      id: props.tournamentID,
      division: division,
    });

    const roundControls = new Array<RoundControl>();

    for (let r = 0; r < roundArray.length; r++) {
      const v = roundArray[r];
      for (let i = v.beginRound; i <= v.endRound; i++) {
        let rdCtrl;
        try {
          rdCtrl = rdCtrlFromSetting(v.setting);
        } catch (e) {
          message.error({
            content: (e as Error).message,
            duration: 5,
          });
          return;
        }
        roundControls.push(rdCtrl);
      }
    }

    ctrls.roundControls = roundControls;
    try {
      await tClient.setRoundControls(ctrls);
      message.info({
        content: "Controls set",
        duration: 3,
      });
    } catch (e) {
      flashError(e);
    }
  };

  const formItemLayout = {
    labelCol: {
      span: 7,
    },
    wrapperCol: {
      span: 12,
    },
  };

  const copySettings = React.useCallback(() => {
    const div = tournamentContext.divisions[copyFromDivision];
    setRoundArray(roundControlsToDisplayArray(div.roundControls));
  }, [
    copyFromDivision,
    roundControlsToDisplayArray,
    tournamentContext.divisions,
  ]);

  return (
    <>
      <Form.Item {...formItemLayout} label="Division">
        <DivisionSelector
          value={division}
          onChange={(value: string) => setDivision(value)}
        />
      </Form.Item>

      {Object.keys(tournamentContext.divisions).length > 1 &&
        division !== "" &&
        roundArray.length === 0 && (
          <Collapse style={{ marginBottom: 10 }}>
            <Collapse.Panel header="Copy from division" key="copyFrom">
              <p className="readable-text-color" style={{ marginBottom: 10 }}>
                Copy from existing division:
                <HelptipLabel
                  labelText=""
                  help="If you want to copy round controls from another division, select
                that division and click the Copy button. Note that you must still
                click Save round controls to save your controls after copying them."
                />
              </p>
              <DivisionSelector
                value={copyFromDivision}
                onChange={(value: string) => setCopyFromDivision(value)}
                exclude={[division]}
              />
              <Button onClick={() => copySettings()}>
                Copy from {copyFromDivision}
              </Button>
              <p className="readable-text-color" style={{ marginTop: 10 }}>
                Or, set from scratch:
              </p>
            </Collapse.Panel>
          </Collapse>
        )}

      <Divider />
      {roundArray.map((v, idx) => (
        <RoundControlFields
          key={`rdctrl-${idx}`}
          setting={v}
          onChange={(fieldName: string, value: string | number | boolean) => {
            const newRdArray = [...roundArray];

            if (fieldName === "beginRound" || fieldName === "endRound") {
              newRdArray[idx] = {
                ...newRdArray[idx],
                [fieldName]: value,
              };
            } else {
              newRdArray[idx] = {
                ...newRdArray[idx],
                setting: create(RoundControlSchema, {
                  ...newRdArray[idx].setting,
                  [fieldName]: value,
                }),
              };
            }
            setRoundArray(newRdArray);
          }}
          onRemove={() => {
            const newRdArray = [...roundArray];
            newRdArray.splice(idx, 1);
            setRoundArray(newRdArray);
          }}
        />
      ))}
      <Button
        onClick={() => {
          const newRdArray = [...roundArray];
          const last = roundArray[roundArray.length - 1];
          newRdArray.push({
            beginRound: last?.endRound ? last.endRound + 1 : 1,
            endRound: last?.endRound ? last.endRound + 1 : 1,
            setting: create(RoundControlSchema, {
              pairingMethod: PairingMethod.MANUAL,
            }),
          });
          setRoundArray(newRdArray);
        }}
      >
        + Add more pairings
      </Button>

      <Button onClick={() => setRoundControls()}>Save round controls</Button>
    </>
  );
};

const ExportTournament = (props: { tournamentID: string }) => {
  const { tournamentContext } = useTournamentStoreContext();
  const formItemLayout = {
    labelCol: {
      span: 7,
    },
    wrapperCol: {
      span: 12,
    },
  };
  const tClient = useClient(TournamentService);
  const onSubmit = async (vals: Store) => {
    const obj = {
      id: props.tournamentID,
      format: vals.format,
    };
    try {
      const resp = await tClient.exportTournament(obj);
      const url = window.URL.createObjectURL(new Blob([resp.exported]));
      const link = document.createElement("a");
      link.href = url;
      const tname = tournamentContext.metadata.name;
      let extension;
      switch (vals.format) {
        case "tsh":
          extension = "tsh";
          break;
        case "standingsonly":
          extension = "csv";
          break;
      }
      const downloadFilename = `${tname}.${extension}`;
      link.setAttribute("download", downloadFilename);
      document.body.appendChild(link);
      link.onclick = () => {
        link.remove();
        setTimeout(() => {
          window.URL.revokeObjectURL(url);
        }, 1000);
      };
      link.click();
    } catch (e) {
      flashError(e);
    }
  };

  return (
    <>
      <Form onFinish={onSubmit}>
        <Form.Item {...formItemLayout} label="Select format" name="format">
          <Select>
            <Select.Option value="tsh">
              NASPA tournament submit format
            </Select.Option>
            {/* <Select.Option value="aupair">AUPair format</Select.Option> */}
            <Select.Option value="standingsonly">
              Standings only (CSV)
            </Select.Option>
          </Select>
        </Form.Item>
        <Form.Item>
          <Button type="primary" htmlType="submit">
            Submit
          </Button>
        </Form.Item>
      </Form>
    </>
  );
};

const CreatePrintableScorecards = (props: { tournamentID: string }) => {
  const { tournamentContext } = useTournamentStoreContext();
  const formItemLayout = {
    labelCol: {
      span: 7,
    },
    wrapperCol: {
      span: 12,
    },
  };
  const tClient = useClient(TournamentService);
  const [isLoading, setIsLoading] = useState(false);
  const onSubmit = async (vals: Store) => {
    const obj = {
      id: props.tournamentID,
      showOpponents: vals.showOpponents,
      showSeeds: vals.showSeeds,
      showQrCode: vals.showQrCode,
    };
    setIsLoading(true);

    try {
      const resp = await tClient.getTournamentScorecards(obj);
      // @ts-expect-error - TypeScript issue with Uint8Array type
      const url = window.URL.createObjectURL(new Blob([resp.pdfZip]));
      const link = document.createElement("a");
      link.href = url;
      const tname = tournamentContext.metadata.name;
      const extension = "zip";
      const downloadFilename = `${tname}.${extension}`;
      link.setAttribute("download", downloadFilename);
      document.body.appendChild(link);
      link.onclick = () => {
        link.remove();
        setTimeout(() => {
          window.URL.revokeObjectURL(url);
        }, 1000);
      };
      link.click();
    } catch (e) {
      flashError(e);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <>
      <Form onFinish={onSubmit}>
        <Form.Item
          {...formItemLayout}
          label="Show opponents"
          name="showOpponents"
        >
          <Switch />
        </Form.Item>
        <Form.Item {...formItemLayout} label="Show seeds" name="showSeeds">
          <Switch />
        </Form.Item>

        <Form.Item {...formItemLayout} label="Show QR code" name="showQrCode">
          <Switch />
        </Form.Item>

        <Form.Item>
          <Button type="primary" htmlType="submit" loading={isLoading}>
            Submit
          </Button>
        </Form.Item>
      </Form>
    </>
  );
};

const CountFirsts = (props: { tournamentID: string }) => {
  const [division, setDivision] = useState("");
  const { tournamentContext } = useTournamentStoreContext();
  const [firstsDisplay, setFirstsDisplay] = useState<string[]>([]);
  const countFirsts = useCallback(() => {
    // go through the tournament and count the number of firsts for every player,
    // then display these in this widget. we're not using the API, but just doing
    // a front-end count.
    const firsts: Record<string, number> = {};
    if (!division) {
      message.error("Please select a division first.");
      return;
    }
    const div = tournamentContext.divisions[division];
    for (let i = 0; i < div.pairings.length; i++) {
      const m = new Set<string>();
      for (let j = 0; j < div.pairings[i].roundPairings.length; j++) {
        const rp = div.pairings[i].roundPairings[j];
        const playerName = rp.players[0].id.split(":")[1];
        if (!m.has(playerName)) {
          m.add(playerName);
          firsts[playerName] = (firsts[playerName] || 0) + 1;
        }
      }
    }
    const firstsList = Object.entries(firsts)
      .sort((a, b) => {
        if (b[1] !== a[1]) {
          return b[1] - a[1]; // Sort by count descending
        }
        return a[0].localeCompare(b[0]); // If counts are equal, sort by name ascending
      })
      .map(([name, count]) => `${name}: ${count} firsts`);
    setFirstsDisplay(firstsList);
  }, [division, tournamentContext.divisions]);
  return (
    <>
      <DivisionFormItem onChange={(div: string) => setDivision(div)} />

      <Button onClick={countFirsts}>Count firsts</Button>
      <div className="readable-text-color" style={{ marginTop: 10 }}>
        {firstsDisplay.length > 0 ? (
          <ul>
            {firstsDisplay.map((f, idx) => (
              <li key={`first-${idx}`}>{f}</li>
            ))}
          </ul>
        ) : (
          "No firsts counted yet."
        )}
      </div>
    </>
  );
};

const UnstartTournament = (props: { tournamentID: string }) => {
  const tClient = useClient(TournamentService);
  const [form] = Form.useForm();

  const onSubmit = async (vals: Store) => {
    try {
      await tClient.unstartTournament({ id: props.tournamentID });
      window.location.reload();
    } catch (e) {
      flashError(e);
    }
  };

  return (
    <Form form={form} onFinish={onSubmit}>
      <div className="readable-text-color">
        This button will <strong style={{ color: "red" }}>DELETE</strong> all
        tournament game results. It is as if it had never started. IT IS NOT
        UNDOABLE. Once you click it, the tournament will RESET!
      </div>

      <Form.Item>
        <Button htmlType="submit" type="primary" danger>
          Unstart this tournament
        </Button>
      </Form.Item>
    </Form>
  );
};

const UnfinishTournament = (props: { tournamentID: string }) => {
  const tClient = useClient(TournamentService);
  const [form] = Form.useForm();

  const onSubmit = async (vals: Store) => {
    try {
      await tClient.unfinishTournament({ id: props.tournamentID });
      window.location.reload();
    } catch (e) {
      flashError(e);
    }
  };

  return (
    <Form form={form} onFinish={onSubmit}>
      <div className="readable-text-color">
        This button will unfinish a tournament that has already finished. This
        allows you to edit scores or even more.
      </div>
      <Form.Item>
        <Button htmlType="submit" type="primary">
          Unfinish this tournament
        </Button>
      </Form.Item>
    </Form>
  );
};

const ManageCheckIns = (props: { tournamentID: string }) => {
  const { tournamentContext } = useTournamentStoreContext();

  const tClient = useClient(TournamentService);
  const [form] = Form.useForm();
  const [checkinsOpen, setCheckinsOpen] = useState(false); // Actual state from metadata
  const [desiredCheckinsState, setDesiredCheckinsState] = useState(false); // Desired state for the button
  const [registrationsOpen, setRegistrationsOpen] = useState(false); // Actual state from metadata
  const [desiredRegistrationsState, setDesiredRegistrationsState] =
    useState(false); // Desired state for the button

  const setCheckinsState = async (vals: Store) => {
    if (!vals.checkinsOpen) {
      try {
        await tClient.closeCheckins({
          id: props.tournamentID,
        });
        message.info({
          content:
            "Closed check-ins successfully. If you would like to delete players who have not checked in, please click the Delete Unchecked-In Players button.",
          duration: 10,
        });
      } catch (e) {
        flashError(e);
      }
    } else {
      try {
        await tClient.openCheckins({
          id: props.tournamentID,
        });
        message.info({
          content: "Opened check-ins successfully.",
          duration: 3,
        });
      } catch (e) {
        flashError(e);
      }
    }
  };

  const setRegistrationState = async (vals: Store) => {
    if (!vals.registrationOpen) {
      try {
        await tClient.closeRegistration({
          id: props.tournamentID,
        });
        message.info({
          content: "Closed registration successfully.",
          duration: 3,
        });
      } catch (e) {
        flashError(e);
      }
    } else {
      try {
        await tClient.openRegistration({
          id: props.tournamentID,
        });
        message.info({
          content: "Opened registration successfully.",
          duration: 3,
        });
      } catch (e) {
        flashError(e);
      }
    }
  };

  useEffect(() => {
    const metadata = tournamentContext.metadata;
    setCheckinsOpen(metadata.checkinsOpen);
    setDesiredCheckinsState(metadata.checkinsOpen); // Initialize desired state to match actual state
    setRegistrationsOpen(metadata.registrationOpen);
    setDesiredRegistrationsState(metadata.registrationOpen); // Initialize desired state to match actual state
    form.setFieldsValue({
      checkinsOpen: metadata.checkinsOpen,
      registrationOpen: metadata.registrationOpen,
    });
  }, [form, tournamentContext.metadata]);

  const uncheckedText = useMemo(() => {
    const uncheckedInPlayers = Object.values(
      tournamentContext.divisions,
    ).flatMap((division) =>
      division.players.filter((player) => !player.checkedIn),
    );

    const uncheckedInCount = uncheckedInPlayers.length;

    if (uncheckedInCount > 0) {
      const uncheckedInNames = uncheckedInPlayers
        .slice(0, 3)
        .map((player) => username(player.id))
        .join(", ");

      const remainingCount = uncheckedInCount - 3;

      return (
        <>
          <div>
            <strong>
              Players who have not checked in:{" "}
              {remainingCount > 0
                ? `${uncheckedInNames}, and ${remainingCount} more...`
                : uncheckedInNames}
            </strong>
          </div>
          <div style={{ marginTop: 8 }}>
            To delete these players from the tournament, click this button,
            after closing check-ins:
          </div>
          <div>
            <Popconfirm
              title="Are you sure you want to delete these players from the tournament? This cannot be undone."
              onConfirm={async () => {
                try {
                  await tClient.removeAllPlayersNotCheckedIn({
                    id: props.tournamentID,
                  });
                  message.info({
                    content: "Deleted unchecked-in players successfully.",
                    duration: 3,
                  });
                } catch (e) {
                  flashError(e);
                }
              }}
            >
              <Button
                type="primary"
                danger
                disabled={tournamentContext.metadata.checkinsOpen}
              >
                <DeleteOutlined />
                Delete&nbsp;
                {singularCount(uncheckedInCount, "player", "players")}
              </Button>
            </Popconfirm>
          </div>
        </>
      );
    } else {
      return <div>All players have checked in.</div>;
    }
  }, [
    tournamentContext.divisions,
    tournamentContext.metadata.checkinsOpen,
    tClient,
    props.tournamentID,
  ]);

  return (
    <Form form={form}>
      <h3>Check-ins</h3>
      <div>Check-ins are currently: {checkinsOpen ? "Open" : "Closed"}</div>
      <Form.Item name="checkinsOpen" label="Allow players to check in">
        <Switch
          checked={desiredCheckinsState}
          onChange={(checked) => setDesiredCheckinsState(checked)}
        />
      </Form.Item>

      <Form.Item>
        <Button
          type="primary"
          onClick={() => setCheckinsState(form.getFieldsValue())}
          disabled={checkinsOpen === desiredCheckinsState}
        >
          {desiredCheckinsState ? "Open" : "Close"} check-ins
        </Button>
      </Form.Item>
      <div style={{ fontSize: "12px", marginBottom: "8px" }}>
        If check-ins are on, players can check in to the tournament. If they
        don't check in, they can be removed from the tournament.
      </div>
      {uncheckedText}
      <Divider />
      <h3>Registrations</h3>

      <div>
        Registration is currently: {registrationsOpen ? "Open" : "Closed"}
      </div>

      <div style={{ fontSize: "12px", marginBottom: "8px" }}>
        If self-register is on, players can register themselves for the
        tournament and choose their division. Otherwise, you have to add players
        before they can check in.
      </div>
      <Form.Item name="registrationOpen" label="Allow players to self-register">
        <Switch
          checked={desiredRegistrationsState}
          onChange={(checked) => setDesiredRegistrationsState(checked)}
        />
      </Form.Item>
      <Form.Item>
        <Button
          type="primary"
          onClick={() => setRegistrationState(form.getFieldsValue())}
          disabled={registrationsOpen === desiredRegistrationsState}
        >
          {desiredRegistrationsState ? "Open" : "Close"} registration
        </Button>
      </Form.Item>
    </Form>
  );
};

// const RunCOP = (props: { tournamentID: string }) => {
//   const { tournamentContext } = useTournamentStoreContext();
//   const [division, setDivision] = useState("");
//   const tClient = useClient(TournamentService);
//   const [form] = Form.useForm();

//   useEffect(() => {
//     form.setFieldsValue({
//       controlLossActivationRound: Math.max(
//         (tournamentContext.divisions[division]?.numRounds ?? 0) - 3,
//         1,
//       ),
//     });
//   }, [division, tournamentContext.divisions[division]?.numRounds]);

//   return (
//     <Form
//       form={form}
//       initialValues={{
//         gibsonSpreads: "250, 200",
//         placePrizes: 5,
//         divisionSims: 100000,
//         controlLossSims: 10000,
//         roundNumber: 0,
//       }}
//     >
//       <h3>(Experimental) Run COP for one round</h3>
//       <Form.Item label="Division">
//         <DivisionSelector
//           value={division}
//           onChange={(value: string) => setDivision(value)}
//         />
//       </Form.Item>
//       <div style={{ fontSize: "12px", marginBottom: "8px" }}>
//         Comma-separated Gibson game spreads per round. Ordered from last round
//         to first round.
//       </div>
//       <Form.Item name="gibsonSpreads" label="Gibsonization spreads">
//         <Input />
//       </Form.Item>

//       <Form.Item name="placePrizes" label="Number of place prizes">
//         <InputNumber inputMode="numeric" min={1} required />
//       </Form.Item>

//       <div style={{ fontSize: "12px", marginBottom: "8px" }}>
//         The "Control Loss Activation Round" is the round at which the control
//         loss simulation is activated. You want this to be towards the end of the
//         tournament, where there are a few people in contention. A good number is
//         number of rounds minus 3 (so, the last 4 rounds would use this sim).
//       </div>

//       <Form.Item
//         name="controlLossActivationRound"
//         label="Control Loss Activation Round"
//       >
//         <InputNumber inputMode="numeric" min={1} required />
//       </Form.Item>

//       <Form.Item name="divisionSims" label="Division sims">
//         <InputNumber inputMode="numeric" min={1} required />
//       </Form.Item>

//       <Form.Item name="controlLossSims" label="Control loss sims">
//         <InputNumber inputMode="numeric" min={1} required />
//       </Form.Item>

//       <div style={{ fontSize: "12px", marginBottom: "8px" }}>
//         The round number is not required for COP, as it always tries to create
//         pairings for the next round. However, you can specify it here to see
//         what the pairings would have been for some round in the past. Keep it at
//         "0" to pair the next round.
//       </div>

//       <Form.Item name="roundNumber" label="Round number">
//         <InputNumber inputMode="numeric" min={0} required />
//       </Form.Item>
//     </Form>
//   );
// };

const EditDescription = (props: { tournamentID: string }) => {
  const { tournamentContext } = useTournamentStoreContext();
  const tClient = useClient(TournamentService);
  const [form] = Form.useForm();

  useEffect(() => {
    const metadata = tournamentContext.metadata;
    const scheduledStartTime = metadata.scheduledStartTime
      ? protobufTimestampToDayjsIgnoringNanos(metadata.scheduledStartTime)
      : null;
    const scheduledEndTime = metadata.scheduledEndTime
      ? protobufTimestampToDayjsIgnoringNanos(metadata.scheduledEndTime)
      : null;

    form.setFieldsValue({
      name: metadata.name,
      description: metadata.description,
      logo: metadata.logo,
      color: metadata.color,
      scheduledTime: {
        range: [scheduledStartTime, scheduledEndTime],
      },
      irlMode: metadata.irlMode,
    });
  }, [form, tournamentContext.metadata]);

  const onSubmit = async (vals: Store) => {
    const [scheduledStartTime, scheduledEndTime] = vals.scheduledTime
      ?.range ?? [null, null];
    const obj = {
      metadata: {
        name: vals.name,
        description: vals.description,
        logo: vals.logo,
        color: vals.color,
        id: props.tournamentID,
        scheduledStartTime: scheduledStartTime
          ? dayjsToProtobufTimestampIgnoringNanos(scheduledStartTime)
          : undefined,
        scheduledEndTime: scheduledEndTime
          ? dayjsToProtobufTimestampIgnoringNanos(scheduledEndTime)
          : undefined,
        irlMode: vals.irlMode,
      },
      setOnlySpecified: true,
    };
    try {
      await tClient.setTournamentMetadata(obj);
      message.info({
        content: "Set tournament metadata successfully.",
        duration: 3,
      });
    } catch (e) {
      flashError(e);
    }
  };

  const timeFormat = doesCurrentUserUse24HourTime() ? "HH:mm" : "hh:mm A";

  return (
    <>
      <Form form={form} onFinish={onSubmit} layout="vertical">
        <Form.Item name="name" label="Club or tournament name">
          <Input />
        </Form.Item>
        <Form.Item label="Tournament Start and End Times">
          <div style={{ fontSize: "12px", color: "#666", marginBottom: "8px" }}>
            Use your local time zone. Times are used for tournament listing. The
            tournament will still only start/end when the director does so
            manually.
          </div>
          <Form.Item name={["scheduledTime", "range"]} noStyle>
            <DatePicker.RangePicker
              style={{ width: "100%" }}
              showTime={{ format: timeFormat }}
              format={`YYYY-MM-DD ${timeFormat}`}
              showNow={false}
            />
          </Form.Item>
        </Form.Item>
        <Form.Item name="description" label="Description">
          <Input.TextArea rows={12} />
        </Form.Item>
        <Form.Item label="Logo URL">
          <div style={{ fontSize: "12px", color: "#666", marginBottom: "8px" }}>
            Optional, requires refresh
          </div>
          <Form.Item name="logo" noStyle>
            <Input />
          </Form.Item>
        </Form.Item>
        <Form.Item label="Hex Color">
          <div style={{ fontSize: "12px", color: "#666", marginBottom: "8px" }}>
            Optional, requires refresh
          </div>

          <Form.Item name="color" noStyle>
            <Input placeholder="#00bdff" />
          </Form.Item>
        </Form.Item>
        <Form.Item>
          <div style={{ fontSize: "12px", color: "#666", marginBottom: "8px" }}>
            IRL (In-Real-Life) Mode is used for real-life tournaments - games
            being played with a physical board and tiles. Once you turn this
            mode on, and click Submit, <em>you cannot turn it off</em>.
          </div>
          <Form.Item name="irlMode" label="IRL Mode">
            <Switch
              disabled={
                tournamentContext.metadata.irlMode ||
                isClubType(tournamentContext.metadata.type)
              }
            />
          </Form.Item>
        </Form.Item>
        <Form.Item style={{ paddingBottom: 20 }}>
          <Button type="primary" htmlType="submit">
            Submit
          </Button>
        </Form.Item>
      </Form>
    </>
  );
};
