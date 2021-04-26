// Ghetto tools are Cesar tools before making things pretty.

import {
  MinusCircleOutlined,
  PlusOutlined,
  SmileOutlined,
} from '@ant-design/icons';
import {
  Button,
  Divider,
  Form,
  Input,
  InputNumber,
  message,
  Modal,
  Select,
  Space,
  Switch,
  Typography,
} from 'antd';
import axios from 'axios';
import { Store } from 'rc-field-form/lib/interface';
import React, { useState } from 'react';
import { postBinary, toAPIUrl, twirpErrToMsg } from '../api/api';
import {
  DivisionControls,
  GameMode,
  GameRequest,
  GameRules,
  PairingMethod,
  RatingMode,
  TournamentGameResult,
} from '../gen/api/proto/realtime/realtime_pb';
import { TournamentResponse } from '../gen/api/proto/tournament_service/tournament_service_pb';
import { Division } from '../store/reducers/tournament_reducer';
import { useTournamentStoreContext } from '../store/store';
import { useMountedState } from '../utils/mounted';
import { SettingsForm } from './game_settings_form';
import '../lobby/seek_form.scss';
import {
  challRuleToStr,
  initTimeDiscreteScale,
  timeScaleToNum,
} from '../store/constants';

import {
  fieldsForMethod,
  pairingMethod,
  PairingMethodField,
  RoundSetting,
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
    'set-pairing': <SetPairing tournamentID={props.tournamentID} />,
    'set-result': <SetResult tournamentID={props.tournamentID} />,
    'pair-round': <PairRound tournamentID={props.tournamentID} />,
    'set-tournament-controls': (
      <SetTournamentControls tournamentID={props.tournamentID} />
    ),
    'set-round-controls': (
      <SetDivisionRoundControls tournamentID={props.tournamentID} />
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
    'Set round controls',
    'Set pairing', // Set a single pairing
    'Pair round', // Pair a whole round
    'Set result', // Set a single result
    'Clear checked in',
  ];

  const listItems = types.map((v) => {
    const key = lowerAndJoin(v);
    return (
      <Button key={key} onClick={() => showModal(key, v)} size="small">
        {v}
      </Button>
    );
  });

  return (
    <>
      <>{listItems}</>
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

const SetTournamentControls = (props: { tournamentID: string }) => {
  const { useState } = useMountedState();
  const [modalVisible, setModalVisible] = useState(false);
  const [
    selectedGameRequest,
    setSelectedGameRequest,
  ] = useState<GameRequest | null>(null);

  const SettingsModalForm = (mprops: {
    visible: boolean;
    onCancel: () => void;
  }) => {
    const [form] = Form.useForm();
    const onOk = () => {
      form.submit();
    };

    return (
      <Modal
        title="Set Game Request"
        visible={mprops.visible}
        onOk={onOk}
        onCancel={mprops.onCancel}
        className="seek-modal"
      >
        <SettingsForm form={form} />
      </Modal>
    );
  };

  const onFinish = async (vals: Store) => {
    console.log('onFinish', vals);
    if (!selectedGameRequest) {
      message.error({
        content: 'Error: No game request',
        duration: 5,
      });
      return;
    }
    const ctrls = new DivisionControls();
    ctrls.setId(props.tournamentID);
    ctrls.setDivision(vals.division);
    ctrls.setGameRequest(selectedGameRequest);
    // can set this later to whatever values, along with a spread
    ctrls.setSuspendedResult(TournamentGameResult.BYE);
    ctrls.setAutoStart(false);

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

  return (
    <Form.Provider
      onFormFinish={(name, { values, forms }) => {
        if (name === 'gameSettingsForm') {
          const { setCtrlsForm } = forms;
          const gr = new GameRequest();
          const rules = new GameRules();
          rules.setBoardLayoutName('CrosswordGame');
          rules.setLetterDistributionName('English');
          gr.setRules(rules);

          gr.setLexicon(values.lexicon);
          gr.setInitialTimeSeconds(
            timeScaleToNum(initTimeDiscreteScale[values.initialtime]) * 60
          );

          if (values.incOrOT === 'increment') {
            gr.setIncrementSeconds(values.extratime);
          } else {
            gr.setMaxOvertimeMinutes(values.extratime);
          }
          gr.setChallengeRule(values.challengerule);
          gr.setGameMode(GameMode.REAL_TIME);
          gr.setRatingMode(values.rated ? RatingMode.RATED : RatingMode.CASUAL);

          setCtrlsForm.setFieldsValue({
            gameRequest: gr,
          });
          // We should ONLY need the above setFieldsValue, but it doesn't work
          // because of this issue:
          // https://github.com/ant-design/ant-design/issues/25087
          // So do a work-around with setState:
          setSelectedGameRequest(gr);
          console.log('setCtrlsForm gr', gr);
          setModalVisible(false);
        }
      }}
    >
      <Form onFinish={onFinish} name="setCtrlsForm">
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

        <Form.Item
          label="Game Settings"
          shouldUpdate={(prevValues, curValues) =>
            prevValues.gameRequest !== curValues.gameRequest
          }
        >
          {({ getFieldValue }) => {
            const gr =
              (getFieldValue('gameRequest') as GameRequest) || undefined;
            return gr ? (
              <dl>
                <dt>Initial Time (Minutes)</dt>
                <dd>{gr.getInitialTimeSeconds() / 60}</dd>
                <dt>Lexicon</dt>
                <dd>{gr.getLexicon()}</dd>
                <dt>Max Overtime (Minutes)</dt>
                <dd>{gr.getMaxOvertimeMinutes()}</dd>
                <dt>Increment (Seconds)</dt>
                <dd>{gr.getIncrementSeconds()}</dd>
                <dt>Challenge Rule</dt>
                <dd>{challRuleToStr(gr.getChallengeRule())}</dd>
                <dt>Rated</dt>
                <dd>
                  {gr.getRatingMode() === RatingMode.RATED ? 'Yes' : 'No'}
                </dd>
              </dl>
            ) : (
              <Typography.Text className="ant-form-text" type="secondary">
                ( <SmileOutlined /> No game settings yet. )
              </Typography.Text>
            );
          }}
        </Form.Item>

        <Form.Item>
          <Button
            htmlType="button"
            style={{
              margin: '0 8px',
            }}
            onClick={() => setModalVisible(true)}
          >
            Edit Game Settings
          </Button>
          <Button type="primary" htmlType="submit">
            Submit
          </Button>
        </Form.Item>
      </Form>

      <SettingsModalForm
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
      />
    </Form.Provider>
  );
};

type RdCtrlFieldsProps = {
  setting: RoundSetting;
  onChange: (
    fieldName: keyof RoundSetting,
    value: string | number | boolean | pairingMethod
  ) => void;
};

const RoundControlFields = (props: RdCtrlFieldsProps) => {
  const { setting } = props;

  const addlFields = fieldsForMethod(setting.pairingType);

  return (
    <>
      First round:
      <InputNumber
        min={1}
        value={setting.beginRound}
        onChange={(e) => props.onChange('beginRound', e as number)}
      />
      Last round:
      <InputNumber
        min={1}
        value={setting.endRound}
        onChange={(e) => props.onChange('endRound', e as number)}
      />
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
      <p></p>
      {/* potential additional fields */}
      {addlFields.map((v: PairingMethodField, idx) => {
        const key = `ni-${idx}`;
        const [fieldType, fieldName, displayName] = v;
        switch (fieldType) {
          case 'number':
            return (
              <p>
                {displayName}
                <InputNumber
                  key={key}
                  min={0}
                  value={setting[fieldName] as number}
                  onChange={(e) => {
                    props.onChange(fieldName, e as number);
                  }}
                />
              </p>
            );

          case 'bool':
            return (
              <p>
                {displayName}
                <Switch
                  key={key}
                  checked={setting[fieldName] as boolean}
                  onChange={(e) => props.onChange(fieldName, e)}
                />
              </p>
            );
        }
      })}
      <Divider />
    </>
  );
};

const SetDivisionRoundControls = (props: { tournamentID: string }) => {
  // This form is too complicated to use the Ant Design built-in forms;
  // So we're just going to use form components instead.

  const [roundArray, setRoundArray] = useState<Array<RoundSetting>>([]);
  const [division, setDivision] = useState('');

  return (
    <>
      <p>
        Division:
        <Input value={division} onChange={(e) => setDivision(e.target.value)} />
      </p>
      <Divider />
      {roundArray.map((v, idx) => (
        <RoundControlFields
          key={`rdctrl-${idx}`}
          setting={v}
          onChange={(
            fieldName: keyof RoundSetting,
            value: string | number | boolean | pairingMethod
          ) => {
            const newRdArray = [...roundArray];
            newRdArray[idx] = {
              ...newRdArray[idx],
              [fieldName]: value,
            };
            setRoundArray(newRdArray);
          }}
        />
      ))}
      <Button
        onClick={() => {
          const newRdArray = [...roundArray];
          newRdArray.push({
            beginRound: 1,
            endRound: 1,
            pairingType: PairingMethod.MANUAL,
          });
          setRoundArray(newRdArray);
        }}
      >
        + Add more pairings
      </Button>
    </>
  );

  /*
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
        <Input />
      </Form.Item>

      <Form.List name="rounds">
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
                  name={[field.name, 'pairingMethod']}
                  fieldKey={[field.fieldKey, 'pairingMethod']}
                  rules={[
                    { required: true, message: 'Missing pairing method' },
                  ]}
                >
                  <Select>
                    <Select.Option value={PairingMethod.RANDOM}>
                      Random
                    </Select.Option>
                    <Select.Option value={PairingMethod.SWISS}>
                      Swiss
                    </Select.Option>

                    <Select.Option value={PairingMethod.ROUND_ROBIN}>
                      Round Robin
                    </Select.Option>
                    <Select.Option value={PairingMethod.INITIAL_FONTES}>
                      Initial Fontes
                    </Select.Option>
                    <Select.Option value={PairingMethod.KING_OF_THE_HILL}>
                      King of the Hill
                    </Select.Option>
                    <Select.Option value={PairingMethod.FACTOR}>
                      Factor
                    </Select.Option>
                    <Select.Option value={PairingMethod.MANUAL}>
                      Manual
                    </Select.Option>
                    <Select.Option value={PairingMethod.TEAM_ROUND_ROBIN}>
                      Team Round Robin
                    </Select.Option>
                  </Select>
                </Form.Item
                  {...field}
                  name={[field.name, '']}
                >

                <Form.Item>

                </Form.Item>

                <MinusCircleOutlined onClick={() => remove(field.name)} />
              </Space>
            ))}
          </>
        )}
      </Form.List>

      <Form.Item>
        <Button type="primary" htmlType="submit">
          Submit
        </Button>
      </Form.Item>
    </Form>
  );*/
};
