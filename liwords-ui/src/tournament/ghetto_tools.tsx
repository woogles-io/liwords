/* eslint-disable @typescript-eslint/camelcase */
// Ghetto tools are Cesar tools before making things pretty.

import {
  MinusCircleOutlined,
  PlusOutlined,
  QuestionCircleOutlined,
} from '@ant-design/icons';
import {
  Button,
  Divider,
  Form,
  Input,
  InputNumber,
  message,
  Select,
  Space,
  Switch,
  Tooltip,
} from 'antd';
import { Modal } from '../utils/focus_modal';
import axios from 'axios';
import { Store } from 'rc-field-form/lib/interface';
import React, { useEffect } from 'react';
import { postBinary, toAPIUrl, twirpErrToMsg } from '../api/api';
import {
  DivisionControls,
  DivisionRoundControls,
  FirstMethod,
  GameRequest,
  PairingMethod,
  RoundControl,
  TournamentGameResult,
} from '../gen/api/proto/realtime/realtime_pb';
import {
  SingleRoundControlsRequest,
  TournamentResponse,
} from '../gen/api/proto/tournament_service/tournament_service_pb';
import { Division } from '../store/reducers/tournament_reducer';
import { useTournamentStoreContext } from '../store/store';
import { useMountedState } from '../utils/mounted';
import { DisplayedGameSetting, SettingsForm } from './game_settings_form';
import '../lobby/seek_form.scss';

import {
  fieldsForMethod,
  pairingMethod,
  PairingMethodField,
  RoundSetting,
  settingsEqual,
  SingleRoundSetting,
} from './pairing_methods';

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
    'add-division': <AddDivision tournamentID={props.tournamentID} />,
    'remove-division': <RemoveDivision tournamentID={props.tournamentID} />,
    'add-players': <AddPlayers tournamentID={props.tournamentID} />,
    'remove-player': <RemovePlayer tournamentID={props.tournamentID} />,
    'clear-checked-in': <ClearCheckedIn tournamentID={props.tournamentID} />,
    'set-single-pairing': <SetPairing tournamentID={props.tournamentID} />,
    'set-game-result': <SetResult tournamentID={props.tournamentID} />,
    'pair-entire-round': <PairRound tournamentID={props.tournamentID} />,
    'unpair-entire-round': <UnpairRound tournamentID={props.tournamentID} />,
    'set-tournament-controls': (
      <SetTournamentControls tournamentID={props.tournamentID} />
    ),
    'set-round-controls': (
      <SetDivisionRoundControls tournamentID={props.tournamentID} />
    ),
    'set-single-round-controls': (
      <SetSingleRoundControls tournamentID={props.tournamentID} />
    ),
  };

  type FormKeys = keyof typeof forms;

  return (
    <Modal
      title={props.title}
      visible={props.visible}
      footer={null}
      destroyOnClose={true}
      onCancel={props.handleCancel}
      className="seek-modal" // temporary display hack
    >
      {forms[props.type as FormKeys]}

      {/* <Form {...layout} form={form} layout="horizontal"></Form> */}
    </Modal>
  );
};

const lowerAndJoin = (v: string): string => {
  const l = v.toLowerCase();
  return l.split(' ').join('-');
};

type Props = {
  tournamentID: string;
};

export const GhettoTools = (props: Props) => {
  const { useState } = useMountedState();
  const [modalTitle, setModalTitle] = useState('');
  const [modalVisible, setModalVisible] = useState(false);
  const [modalType, setModalType] = useState('');

  const showModal = (key: string, title: string) => {
    setModalType(key);
    setModalVisible(true);
    setModalTitle(title);
  };

  const preTournamentTypes = [
    'Add division',
    'Remove division',
    'Set tournament controls',
    'Set round controls',
  ];

  const inTournamentTypes = [
    'Add players',
    'Remove player',
    'Set single round controls', // Set controls for a single round
    'Set single pairing', // Set a single pairing
    'Pair entire round', // Pair a whole round
    'Set game result', // Set a single result
    'Unpair entire round', // Unpair a whole round
    // 'Clear checked in',
  ];

  const preListItems = preTournamentTypes.map((v) => {
    const key = lowerAndJoin(v);
    return (
      <li key={key} style={{ marginBottom: 5 }}>
        <Button onClick={() => showModal(key, v)} size="small">
          {v}
        </Button>
      </li>
    );
  });

  const inListItems = inTournamentTypes.map((v) => {
    const key = lowerAndJoin(v);
    return (
      <li key={key} style={{ marginBottom: 5 }}>
        <Button onClick={() => showModal(key, v)} size="small">
          {v}
        </Button>
      </li>
    );
  });

  return (
    <>
      <h3>Tournament Tools</h3>
      <h4>Pre-tournament settings</h4>
      <ul>{preListItems}</ul>
      <Divider />
      <h4>In-tournament management</h4>
      <ul>{inListItems}</ul>
      <Divider />
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
}) => {
  const { tournamentContext } = useTournamentStoreContext();
  return (
    <Select onChange={props.onChange} value={props.value}>
      {Object.keys(tournamentContext.divisions).map((d) => (
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
          message: 'Please input division name',
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
}) => {
  const { tournamentContext } = useTournamentStoreContext();
  return (
    <Form.Item name={props.name} label={props.label}>
      <Select>
        {tournamentContext.divisions[props.division]?.players.map((v) => {
          const u = username(v.getId());
          return (
            <Select.Option value={u} key={v.getId()}>
              {u}
            </Select.Option>
          );
        })}
      </Select>
    </Form.Item>
  );
};

const AddDivision = (props: { tournamentID: string }) => {
  const onFinish = (vals: Store) => {
    const obj = {
      id: props.tournamentID,
      division: vals.division,
    };
    axios
      .post<{}>(
        toAPIUrl('tournament_service.TournamentService', 'AddDivision'),
        obj
      )
      .then((resp) => {
        message.info({
          content: 'Division added',
          duration: 3,
        });
      })
      .catch((err) => {
        message.error({
          content: 'Error ' + err.response?.data?.msg,
          duration: 5,
        });
      });
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

const RemoveDivision = (props: { tournamentID: string }) => {
  const onFinish = (vals: Store) => {
    const obj = {
      id: props.tournamentID,
      division: vals.division,
    };
    axios
      .post<{}>(
        toAPIUrl('tournament_service.TournamentService', 'RemoveDivision'),
        obj
      )
      .then((resp) => {
        message.info({
          content: 'Division removed',
          duration: 3,
        });
      })
      .catch((err) => {
        message.error({
          content: 'Error ' + err.response?.data?.msg,
          duration: 5,
        });
      });
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
  const onFinish = (vals: Store) => {
    const players = [];
    // const playerMap: { [username: string]: number } = {};
    if (!vals.players) {
      message.error({
        content: 'Add some players first',
        duration: 5,
      });
      return;
    }
    for (let i = 0; i < vals.players.length; i++) {
      const enteredUsername = vals.players[i].username;
      if (!enteredUsername) {
        continue;
      }
      const username = enteredUsername.trim();
      if (username === '') {
        continue;
      }
      players.push({
        id: username,
        rating: vals.players[i].rating,
      });
    }

    if (players.length === 0) {
      message.error({
        content: 'Add some players first',
        duration: 5,
      });
      return;
    }

    const obj = {
      id: props.tournamentID,
      division: vals.division,
      persons: players,
    };
    console.log(obj);
    axios
      .post<{}>(
        toAPIUrl('tournament_service.TournamentService', 'AddPlayers'),
        obj
      )
      .then((resp) => {
        message.info({
          content: 'Players added',
          duration: 3,
        });
      })
      .catch((err) => {
        message.error({
          content: 'Error ' + err.response?.data?.msg,
          duration: 5,
        });
      });
  };

  return (
    <Form onFinish={onFinish}>
      <DivisionFormItem />

      <Form.List name="players">
        {(fields, { add, remove }) => (
          <>
            {fields.map((field) => (
              <Space
                key={field.key}
                style={{ display: 'flex', marginBottom: 8 }}
                align="baseline"
              >
                <Form.Item
                  {...field}
                  name={[field.name, 'username']}
                  fieldKey={[field.fieldKey, 'username']}
                  rules={[{ required: true, message: 'Missing username' }]}
                >
                  <Input placeholder="Username" />
                </Form.Item>
                <Form.Item
                  {...field}
                  name={[field.name, 'rating']}
                  fieldKey={[field.fieldKey, 'rating']}
                  rules={[{ required: true, message: 'Missing rating' }]}
                >
                  <Input placeholder="Rating" />
                </Form.Item>
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
  const { useState } = useMountedState();
  const [division, setDivision] = useState('');

  const onFinish = (vals: Store) => {
    const obj = {
      id: props.tournamentID,
      division: vals.division,
      persons: [
        {
          id: vals.username,
        },
      ],
    };
    console.log(obj);
    axios
      .post<{}>(
        toAPIUrl('tournament_service.TournamentService', 'RemovePlayers'),
        obj
      )
      .then((resp) => {
        message.info({
          content: 'Player removed',
          duration: 3,
        });
      })
      .catch((err) => {
        message.error({
          content: 'Error ' + err.response?.data?.msg,
          duration: 5,
        });
      });
  };

  return (
    <Form onFinish={onFinish}>
      <DivisionFormItem onChange={(div: string) => setDivision(div)} />

      <PlayersFormItem
        name="username"
        label="Username to remove"
        division={division}
      />
      <Form.Item>
        <Button type="primary" htmlType="submit">
          Submit
        </Button>
      </Form.Item>
    </Form>
  );
};

const ClearCheckedIn = (props: { tournamentID: string }) => {
  const onFinish = (vals: Store) => {
    const obj = {
      id: props.tournamentID,
    };
    axios
      .post<{}>(
        toAPIUrl('tournament_service.TournamentService', 'UncheckIn'),
        obj
      )
      .then((resp) => {
        message.info({
          content: 'Checkins cleared',
          duration: 3,
        });
      })
      .catch((err) => {
        message.error({
          content: 'Error ' + err.response?.data?.msg,
          duration: 5,
        });
      });
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

// userUUID looks up the UUID of a username
const userUUID = (username: string, divobj: Division) => {
  if (!divobj) {
    return '';
  }
  const p = divobj.players.find((p) => {
    const parts = p.getId().split(':');
    const pusername = parts[1].toLowerCase();

    if (username.toLowerCase() === pusername) {
      return true;
    }
    return false;
  });
  if (!p) {
    return '';
  }
  return p.getId().split(':')[0];
};

const username = (fullID: string) => {
  const parts = fullID.split(':');
  return parts[1];
};

const fullPlayerID = (username: string, divobj: Division) => {
  if (!divobj) {
    return '';
  }
  const p = divobj.players.find((p) => {
    const parts = p.getId().split(':');
    const pusername = parts[1].toLowerCase();

    if (username.toLowerCase() === pusername) {
      return true;
    }
    return false;
  });
  if (!p) {
    return '';
  }
  return p.getId();
};

const SetPairing = (props: { tournamentID: string }) => {
  const { tournamentContext } = useTournamentStoreContext();
  const { useState } = useMountedState();
  const [division, setDivision] = useState('');

  const onFinish = (vals: Store) => {
    const obj = {
      id: props.tournamentID,
      division: vals.division,
      pairings: [
        {
          player_one_id: fullPlayerID(
            vals.p1,
            tournamentContext.divisions[vals.division]
          ),
          player_two_id: fullPlayerID(
            vals.p2,
            tournamentContext.divisions[vals.division]
          ),
          round: vals.round - 1, // 1-indexed input
        },
      ],
    };
    axios
      .post<{}>(
        toAPIUrl('tournament_service.TournamentService', 'SetPairing'),
        obj
      )
      .then((resp) => {
        message.info({
          content: 'Pairing set',
          duration: 3,
        });
      })
      .catch((err) => {
        message.error({
          content: 'Error ' + err.response?.data?.msg,
          duration: 5,
        });
      });
  };

  return (
    <Form onFinish={onFinish}>
      <DivisionFormItem onChange={(div: string) => setDivision(div)} />

      <PlayersFormItem
        name="p1"
        label="Player 1 username"
        division={division}
      />
      <PlayersFormItem
        name="p2"
        label="Player 2 username"
        division={division}
      />

      <Form.Item name="round" label="Round (1-indexed)">
        <InputNumber min={1} />
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
  const { useState } = useMountedState();
  const [division, setDivision] = useState('');
  const [score1, setScore1] = useState(0);
  const [score2, setScore2] = useState(0);
  const [form] = Form.useForm();

  const onFinish = (vals: Store) => {
    const obj = {
      id: props.tournamentID,
      division: vals.division,
      player_one_id: userUUID(
        vals.p1,
        tournamentContext.divisions[vals.division]
      ),
      player_two_id: userUUID(
        vals.p2,
        tournamentContext.divisions[vals.division]
      ),
      round: vals.round - 1, // 1-indexed input
      player_one_score: vals.p1score,
      player_two_score: vals.p2score,
      player_one_result: vals.p1result,
      player_two_result: vals.p2result,
      game_end_reason: vals.gameEndReason,
      amendment: vals.amendment,
    };
    axios
      .post<{}>(
        toAPIUrl('tournament_service.TournamentService', 'SetResult'),
        obj
      )
      .then((resp) => {
        message.info({
          content: 'Result set',
          duration: 3,
        });
      })
      .catch((err) => {
        message.error({
          content: 'Error ' + err.response?.data?.msg,
          duration: 5,
        });
      });
  };

  useEffect(() => {
    if (score1 > score2) {
      form.setFieldsValue({
        p1result: 'WIN',
        p2result: 'LOSS',
      });
    } else if (score1 < score2) {
      form.setFieldsValue({
        p1result: 'LOSS',
        p2result: 'WIN',
      });
    } else {
      form.setFieldsValue({
        p1result: 'DRAW',
        p2result: 'DRAW',
      });
    }
  }, [form, score1, score2]);

  const score1Change = (v: number | string | null | undefined) => {
    if (typeof v !== 'number') {
      return;
    }
    setScore1(v);
  };
  const score2Change = (v: number | string | null | undefined) => {
    if (typeof v !== 'number') {
      return;
    }
    setScore2(v);
  };

  return (
    <Form
      form={form}
      onFinish={onFinish}
      initialValues={{ gameEndReason: 'STANDARD' }}
    >
      <DivisionFormItem onChange={(div: string) => setDivision(div)} />

      <PlayersFormItem
        name="p1"
        label="Player 1 username"
        division={division}
      />
      <PlayersFormItem
        name="p2"
        label="Player 2 username"
        division={division}
      />

      <Form.Item name="round" label="Round (1-indexed)">
        <InputNumber min={1} />
      </Form.Item>

      <Form.Item name="p1score" label="Player 1 score">
        <InputNumber onChange={score1Change} value={score1} />
      </Form.Item>

      <Form.Item name="p2score" label="Player 2 score">
        <InputNumber onChange={score2Change} value={score2} />
      </Form.Item>

      <Form.Item name="p1result" label="Player 1 result">
        <Select>
          <Select.Option value="NO_RESULT">NO_RESULT</Select.Option>
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
          <Select.Option value="NO_RESULT">NO_RESULT</Select.Option>
          <Select.Option value="WIN">WIN</Select.Option>
          <Select.Option value="LOSS">LOSS</Select.Option>
          <Select.Option value="DRAW">DRAW</Select.Option>
          <Select.Option value="BYE">BYE</Select.Option>
          <Select.Option value="FORFEIT_WIN">FORFEIT_WIN</Select.Option>
          <Select.Option value="FORFEIT_LOSS">FORFEIT_LOSS</Select.Option>
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
  const onFinish = (vals: Store) => {
    const obj = {
      id: props.tournamentID,
      division: vals.division,
      round: vals.round - 1, // 1-indexed input
      preserve_byes: vals.preserveByes,
    };
    axios
      .post<{}>(
        toAPIUrl('tournament_service.TournamentService', 'PairRound'),
        obj
      )
      .then((resp) => {
        message.info({
          content: 'Pairing set',
          duration: 3,
        });
      })
      .catch((err) => {
        message.error({
          content: 'Error ' + err.response?.data?.msg,
          duration: 5,
        });
      });
  };

  return (
    <Form onFinish={onFinish}>
      <DivisionFormItem />

      <Form.Item name="round" label="Round (1-indexed)">
        <InputNumber min={1} />
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
  const onFinish = (vals: Store) => {
    const obj = {
      id: props.tournamentID,
      division: vals.division,
      round: vals.round - 1, // 1-indexed input
      deletePairings: true,
    };
    axios
      .post<{}>(
        toAPIUrl('tournament_service.TournamentService', 'PairRound'),
        obj
      )
      .then((resp) => {
        message.info({
          content: 'Pairings for selected round have been deleted',
          duration: 3,
        });
      })
      .catch((err) => {
        message.error({
          content: 'Error ' + err.response?.data?.msg,
          duration: 5,
        });
      });
  };
  return (
    <Form onFinish={onFinish}>
      <DivisionFormItem />
      <Form.Item name="round" label="Round (1-indexed)">
        <InputNumber min={1} />
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
  const { useState } = useMountedState();
  const [modalVisible, setModalVisible] = useState(false);
  const [
    selectedGameRequest,
    setSelectedGameRequest,
  ] = useState<GameRequest | null>(null);

  const [division, setDivision] = useState('');
  const [gibsonize, setGibsonize] = useState(false);
  const [gibsonSpread, setGibsonSpread] = useState(500);

  // min placement is 0-indexed, but we want to display 1-indexed
  // this variable will be the display variable:
  const [gibsonMinPlacement, setGibsonMinPlacement] = useState(1);
  // bye max placement is 0-indexed, this is also the display variable
  const [byeMaxPlacement, setByeMaxPlacement] = useState(1);
  const { tournamentContext } = useTournamentStoreContext();

  useEffect(() => {
    if (!division) {
      setSelectedGameRequest(null);
      return;
    }
    const div = tournamentContext.divisions[division];
    const gameRequest = div.divisionControls?.getGameRequest();
    if (gameRequest) {
      setSelectedGameRequest(gameRequest);
    } else {
      setSelectedGameRequest(null);
    }
    if (div.divisionControls) {
      setGibsonize(div.divisionControls.getGibsonize());
      setGibsonSpread(div.divisionControls.getGibsonSpread());
      setGibsonMinPlacement(div.divisionControls.getMinimumPlacement() + 1);
      setByeMaxPlacement(div.divisionControls.getMaximumByePlacement() + 1);
    }
  }, [division, tournamentContext.divisions]);

  const SettingsModalForm = (mprops: {
    visible: boolean;
    onCancel: () => void;
  }) => {
    return (
      <Modal
        title="Set Game Request"
        visible={mprops.visible}
        onCancel={mprops.onCancel}
        className="seek-modal"
        okButtonProps={{ style: { display: 'none' } }}
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

  const submit = async () => {
    if (!selectedGameRequest) {
      message.error({
        content: 'Error: No game request',
        duration: 5,
      });
      return;
    }
    const ctrls = new DivisionControls();
    ctrls.setId(props.tournamentID);
    ctrls.setDivision(division);
    ctrls.setGameRequest(selectedGameRequest);
    // can set this later to whatever values, along with a spread
    ctrls.setSuspendedResult(TournamentGameResult.FORFEIT_LOSS);
    ctrls.setAutoStart(false);
    ctrls.setGibsonize(gibsonize);
    ctrls.setGibsonSpread(gibsonSpread);
    ctrls.setMinimumPlacement(gibsonMinPlacement - 1);
    ctrls.setMaximumByePlacement(byeMaxPlacement - 1);

    try {
      const rbin = await postBinary(
        'tournament_service.TournamentService',
        'SetDivisionControls',
        ctrls
      );

      const resp = TournamentResponse.deserializeBinary(rbin.data);
      console.log('setTournamentControls', resp);
      message.info({
        content: 'Controls set',
        duration: 3,
      });
    } catch (err) {
      message.error({
        content: 'Error ' + twirpErrToMsg(err),
        duration: 5,
      });
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

        <Form.Item {...formItemLayout} label="Gibsonize">
          <Switch
            checked={gibsonize}
            onChange={(c: boolean) => setGibsonize(c)}
          />
        </Form.Item>

        <Form.Item {...formItemLayout} label="Gibson spread">
          <InputNumber
            min={0}
            value={gibsonSpread}
            onChange={(v: number | string | undefined | null) =>
              setGibsonSpread(v as number)
            }
          />
        </Form.Item>

        <Form.Item {...formItemLayout} label="Gibson min placement">
          <InputNumber
            min={1}
            value={gibsonMinPlacement}
            onChange={(p: number | string | undefined | null) =>
              setGibsonMinPlacement(p as number)
            }
          />
        </Form.Item>

        <Form.Item {...formItemLayout} label="Bye cut-off">
          <InputNumber
            min={1}
            value={byeMaxPlacement}
            onChange={(p: number | string | undefined | null) =>
              setByeMaxPlacement(p as number)
            }
          />
          <Tooltip
            title="Byes may be assigned to players ranked this, and worse,
          if odd. Make this 1 if you wish everyone in the tournament to be eligible for byes."
          >
            <QuestionCircleOutlined
              className="readable-text-color"
              style={{ marginLeft: 5 }}
            />
          </Tooltip>
        </Form.Item>
      </Form>
      <div>{DisplayedGameSetting(selectedGameRequest)}</div>

      <Button
        htmlType="button"
        style={{
          margin: '0 8px',
        }}
        onClick={() => setModalVisible(true)}
      >
        Edit Game Settings
      </Button>
      <Button type="primary" onClick={submit}>
        Save Game Settings
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
  onChange: (
    fieldName: keyof SingleRoundSetting | keyof RoundSetting,
    value: string | number | boolean | pairingMethod
  ) => void;
  onRemove: () => void;
};

type SingleRdCtrlFieldsProps = {
  setting: SingleRoundSetting;
  onChange: (
    fieldName: keyof SingleRoundSetting,
    value: string | number | boolean | pairingMethod
  ) => void;
};

const SingleRoundControlFields = (props: SingleRdCtrlFieldsProps) => {
  const { setting } = props;
  const addlFields = fieldsForMethod(setting.pairingType);

  const formItemLayout = {
    labelCol: {
      span: 7,
    },
    wrapperCol: {
      span: 6,
    },
  };

  return (
    <>
      <Form.Item {...formItemLayout} label="Pairing Type">
        <Select
          value={setting.pairingType}
          onChange={(e) => {
            props.onChange('pairingType', e);
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
          <Select.Option value={PairingMethod.FACTOR}>Factor</Select.Option>
          <Select.Option value={PairingMethod.MANUAL}>Manual</Select.Option>
          <Select.Option value={PairingMethod.TEAM_ROUND_ROBIN}>
            Team Round Robin
          </Select.Option>
        </Select>
      </Form.Item>
      <p></p>
      {/* potential additional fields */}
      {addlFields.map((v: PairingMethodField, idx) => {
        const key = `ni-${idx}`;
        const [fieldType, fieldName, displayName] = v;
        switch (fieldType) {
          case 'number':
            return (
              <Form.Item
                {...formItemLayout}
                labelCol={{ span: 12, offset: 1 }}
                label={displayName}
                key={`${idx}-${fieldName}`}
              >
                <InputNumber
                  key={key}
                  min={0}
                  value={setting[fieldName] as number}
                  onChange={(e) => {
                    props.onChange(fieldName, e as number);
                  }}
                />
              </Form.Item>
            );

          case 'bool':
            return (
              <Form.Item
                {...formItemLayout}
                labelCol={{ span: 12, offset: 1 }}
                label={displayName}
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
      <Form layout="inline" size="small">
        <Form.Item label="First round">
          <InputNumber
            min={1}
            value={setting.beginRound}
            onChange={(e) => props.onChange('beginRound', e as number)}
          />
        </Form.Item>
        <Form.Item label="Last round">
          <InputNumber
            min={1}
            value={setting.endRound}
            onChange={(e) => props.onChange('endRound', e as number)}
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

const rdCtrlFromSetting = (rdSetting: SingleRoundSetting): RoundControl => {
  const rdCtrl = new RoundControl();
  rdCtrl.setFirstMethod(FirstMethod.AUTOMATIC_FIRST);
  rdCtrl.setGamesPerRound(1);
  rdCtrl.setPairingMethod(rdSetting.pairingType);

  switch (rdSetting.pairingType) {
    case PairingMethod.SWISS:
    case PairingMethod.FACTOR:
      rdCtrl.setMaxRepeats(rdSetting.maxRepeats || 0);
      rdCtrl.setAllowOverMaxRepeats(true);
      rdCtrl.setRepeatRelativeWeight(rdSetting.repeatRelativeWeight || 0);
      rdCtrl.setWinDifferenceRelativeWeight(
        rdSetting.winDifferenceRelativeWeight || 0
      );
      // This should be auto-calculated, and only for factor
      rdCtrl.setFactor(rdSetting.factor || 0);
      break;

    case PairingMethod.TEAM_ROUND_ROBIN:
      rdCtrl.setGamesPerRound(rdSetting.gamesPerRound || 1);
      break;
  }
  // Other cases don't matter, we've already set the pairing method.
  return rdCtrl;
};

const SetSingleRoundControls = (props: { tournamentID: string }) => {
  const { useState } = useMountedState();
  const [division, setDivision] = useState('');
  const [roundSetting, setRoundSetting] = useState<SingleRoundSetting>({
    pairingType: PairingMethod.RANDOM,
  });
  const [userVisibleRound, setUserVisibleRound] = useState(1);

  const showError = (msg: string) => {
    message.error({
      content: 'Error ' + msg,
      duration: 5,
    });
  };

  const setRoundControls = async () => {
    if (!division) {
      showError('Division is missing');
      return;
    }
    if (userVisibleRound <= 0) {
      showError('Round must be a positive round number');
      return;
    }
    if (!roundSetting) {
      showError('Missing round setting');
      return;
    }

    const ctrls = new SingleRoundControlsRequest();
    ctrls.setId(props.tournamentID);
    ctrls.setDivision(division);
    ctrls.setRound(userVisibleRound - 1); // round is 0-indexed on backend.

    const rdCtrl = rdCtrlFromSetting(roundSetting);
    ctrls.setRoundControls(rdCtrl);

    try {
      const rbin = await postBinary(
        'tournament_service.TournamentService',
        'SetSingleRoundControls',
        ctrls
      );

      const resp = TournamentResponse.deserializeBinary(rbin.data);
      console.log('setSingleRoundControls', resp);
      message.info({
        content: `Controls set for round ${userVisibleRound}`,
        duration: 3,
      });
    } catch (err) {
      console.log('err is', err);
      showError(twirpErrToMsg(err));
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
            value={userVisibleRound}
            onChange={(e) => e && setUserVisibleRound(e as number)}
          />
        </Form.Item>
      </Form>
      <Divider />
      <Form size="small">
        <SingleRoundControlFields
          setting={roundSetting}
          onChange={(
            fieldName: keyof SingleRoundSetting,
            value: string | number | boolean | pairingMethod
          ) => {
            const val = { ...roundSetting, [fieldName]: value };
            setRoundSetting(val);
          }}
        />
        <Divider />
        <Button onClick={() => setRoundControls()}>Submit</Button>
      </Form>
    </>
  );
};

const SetDivisionRoundControls = (props: { tournamentID: string }) => {
  const { useState } = useMountedState();
  const { tournamentContext } = useTournamentStoreContext();
  // This form is too complicated to use the Ant Design built-in forms;
  // So we're just going to use form components instead.

  const [roundArray, setRoundArray] = useState<Array<RoundSetting>>([]);
  const [division, setDivision] = useState('');

  useEffect(() => {
    if (!division) {
      setRoundArray([]);
      return;
    }
    const div = tournamentContext.divisions[division];
    const settings = new Array<RoundSetting>();

    let lastSetting: SingleRoundSetting | null = null;
    let min = 1;
    let max = 1;
    div.roundControls.forEach((v: RoundControl, rd: number) => {
      const thisSetting = {
        pairingType: v.getPairingMethod(),
        gamesPerRound: v.getGamesPerRound(),
        factor: v.getFactor(),
        maxRepeats: v.getMaxRepeats(),
        allowOverMaxRepeats: v.getAllowOverMaxRepeats(),
        repeatRelativeWeight: v.getRepeatRelativeWeight(),
        winDifferenceRelativeWeight: v.getWinDifferenceRelativeWeight(),
      };
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

    setRoundArray(settings);
  }, [division, tournamentContext.divisions]);

  const showError = (msg: string) => {
    message.error({
      content: 'Error ' + msg,
      duration: 5,
    });
  };

  const setRoundControls = async () => {
    if (!division) {
      showError('Division is missing');
      return;
    }
    if (!roundArray.length) {
      showError('Round controls are missing');
      return;
    }
    // validate round array
    let lastRd = 0;
    for (let i = 0; i < roundArray.length; i++) {
      const rdCtrl = roundArray[i];
      if (rdCtrl.beginRound <= lastRd) {
        showError('Round numbers must be consecutive and increasing');
        return;
      }
      if (rdCtrl.endRound < rdCtrl.beginRound) {
        showError('End round must not be smaller than begin round');
        return;
      }
      if (rdCtrl.beginRound > lastRd + 1) {
        showError('Round numbers must be consecutive; you cannot skip rounds');
        return;
      }
      lastRd = rdCtrl.endRound;
    }

    const ctrls = new DivisionRoundControls();
    ctrls.setId(props.tournamentID);
    ctrls.setDivision(division);

    const roundControls = new Array<RoundControl>();

    roundArray.forEach((v) => {
      for (let i = v.beginRound; i <= v.endRound; i++) {
        roundControls.push(rdCtrlFromSetting(v.setting));
      }
    });
    ctrls.setRoundControlsList(roundControls);

    try {
      const rbin = await postBinary(
        'tournament_service.TournamentService',
        'SetRoundControls',
        ctrls
      );

      const resp = TournamentResponse.deserializeBinary(rbin.data);
      console.log('setRoundControls', resp);
      message.info({
        content: 'Controls set',
        duration: 3,
      });
    } catch (err) {
      showError(twirpErrToMsg(err));
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
      <Form.Item {...formItemLayout} label="Division">
        <DivisionSelector
          value={division}
          onChange={(value: string) => setDivision(value)}
        />
      </Form.Item>
      <Divider />
      {roundArray.map((v, idx) => (
        <RoundControlFields
          key={`rdctrl-${idx}`}
          setting={v}
          onChange={(
            fieldName: keyof RoundSetting | keyof SingleRoundSetting,
            value: string | number | boolean | pairingMethod
          ) => {
            const newRdArray = [...roundArray];

            if (fieldName === 'beginRound' || fieldName === 'endRound') {
              newRdArray[idx] = {
                ...newRdArray[idx],
                [fieldName]: value,
              };
            } else {
              newRdArray[idx] = {
                ...newRdArray[idx],
                setting: {
                  ...newRdArray[idx].setting,
                  [fieldName]: value,
                },
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
            setting: { pairingType: PairingMethod.MANUAL },
          });
          setRoundArray(newRdArray);
        }}
      >
        + Add more pairings
      </Button>

      <Button onClick={() => setRoundControls()}>Submit</Button>
    </>
  );
};
