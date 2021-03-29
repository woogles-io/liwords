// Ghetto tools are Cesar tools before making things pretty.

import { MinusCircleOutlined, PlusOutlined } from '@ant-design/icons';
import {
  Button,
  Form,
  Input,
  InputNumber,
  message,
  Modal,
  Select,
  Space,
  Switch,
} from 'antd';
import axios from 'axios';
import { Store } from 'rc-field-form/lib/interface';
import React, { useState } from 'react';
import { toAPIUrl } from '../api/api';
import { Division } from '../store/reducers/tournament_reducer';
import { useTournamentStoreContext } from '../store/store';

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
    'set-pairing': <SetPairing tournamentID={props.tournamentID} />,
    'set-result': <SetResult tournamentID={props.tournamentID} />,
    'pair-round': <PairRound tournamentID={props.tournamentID} />,
  };

  type FormKeys = keyof typeof forms;

  return (
    <Modal
      title={props.title}
      visible={props.visible}
      footer={null}
      destroyOnClose={true}
      onCancel={props.handleCancel}
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
  const [modalTitle, setModalTitle] = useState('');
  const [modalVisible, setModalVisible] = useState(false);
  const [modalType, setModalType] = useState('');

  const showModal = (key: string, title: string) => {
    setModalType(key);
    setModalVisible(true);
    setModalTitle(title);
  };

  const types = [
    'Add division',
    'Remove division',
    'Add players',
    'Remove player',
    'Set tournament controls',
    'Set single round controls',
    'Set pairing', // Set a single pairing
    'Pair round', // Pair a whole round
    'Set result', // Set a single result
    'Clear checked in',
  ];

  const listItems = types.map((v) => {
    const key = lowerAndJoin(v);
    return (
      <li key={key} onClick={() => showModal(key, v)}>
        {v}
      </li>
    );
  });

  return (
    <>
      <p>
        <ul>{listItems}</ul>
      </p>
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
  // XXX: RemoveDivision does not update list in real-time for some reason.
  // (I think it's because the back-end always sends divisions one at a time,
  // and not the fact that one was deleted)
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

const AddPlayers = (props: { tournamentID: string }) => {
  const onFinish = (vals: Store) => {
    const playerMap: { [username: string]: number } = {};
    if (!vals.players) {
      message.error({
        content: 'Add some players first',
        duration: 5,
      });
      return;
    }
    console.log(vals.players);
    for (let i = 0; i < vals.players.length; i++) {
      const enteredUsername = vals.players[i].username;
      if (!enteredUsername) {
        continue;
      }
      const username = enteredUsername.trim();
      if (username === '') {
        continue;
      }
      playerMap[username] = parseInt(vals.players[i].rating);
    }

    if (Object.keys(playerMap).length === 0) {
      message.error({
        content: 'Add some players first',
        duration: 5,
      });
      return;
    }

    const obj = {
      id: props.tournamentID,
      division: vals.division,
      persons: playerMap,
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
      <Form.Item name="division" label="Division Name">
        <Input />
      </Form.Item>

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
  const onFinish = (vals: Store) => {
    const obj = {
      id: props.tournamentID,
      division: vals.division,
      persons: { [vals.username]: 0 },
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
        {/* lazy right now but all of these need required */}
        <Input />
      </Form.Item>
      <Form.Item name="username" label="Username to remove">
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

const SetPairing = (props: { tournamentID: string }) => {
  const { tournamentContext } = useTournamentStoreContext();

  const onFinish = (vals: Store) => {
    const obj = {
      id: props.tournamentID,
      division: vals.division,
      pairings: [
        {
          player_one_id:
            userUUID(vals.p1, tournamentContext.divisions[vals.division]) +
            ':' +
            vals.p1,
          player_two_id:
            userUUID(vals.p2, tournamentContext.divisions[vals.division]) +
            ':' +
            vals.p2,
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
        {/* lazy right now but all of these need required */}
        <Input />
      </Form.Item>

      <Form.Item name="p1" label="Player 1 username">
        <Input />
      </Form.Item>

      <Form.Item name="p2" label="Player 2 username">
        <Input />
      </Form.Item>

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

  return (
    <Form onFinish={onFinish}>
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
        {/* lazy right now but all of these need required */}
        <Input />
      </Form.Item>

      <Form.Item name="p1" label="Player 1 username">
        <Input />
      </Form.Item>

      <Form.Item name="p2" label="Player 2 username">
        <Input />
      </Form.Item>

      <Form.Item name="round" label="Round (1-indexed)">
        <InputNumber min={1} />
      </Form.Item>

      <Form.Item name="p1score" label="Player 1 score">
        <InputNumber />
      </Form.Item>

      <Form.Item name="p2score" label="Player 2 score">
        <InputNumber />
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

      <Form.Item name="amendment" label="Amendment">
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
        {/* lazy right now but all of these need required */}
        <Input />
      </Form.Item>

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
