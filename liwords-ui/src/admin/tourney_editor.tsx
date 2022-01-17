import React from 'react';

import axios from 'axios';
import Search from 'antd/lib/input/Search';
import {
  Button,
  Card,
  Col,
  Divider,
  Form,
  Input,
  message,
  Row,
  Select,
  Switch,
} from 'antd';
import { Modal } from '../utils/focus_modal';
import { Store } from 'antd/lib/form/interface';
import '../App.scss';
import '../lobby/lobby.scss';
import 'antd/dist/antd.css';
import ReactMarkdown from 'react-markdown';
import {
  DisplayedGameSetting,
  SettingsForm,
} from '../tournament/game_settings_form';

import { useMountedState } from '../utils/mounted';
import { postProto, toAPIUrl } from '../api/api';
import {
  GetTournamentMetadataRequest,
  TournamentMetadataResponse,
  TType,
  TTypeMap,
} from '../gen/api/proto/tournament_service/tournament_service_pb';
import { GameRequest } from '../gen/api/proto/realtime/realtime_pb';

type DProps = {
  description: string;
  disclaimer: string;
  title: string;
  color: string;
  logo: string;
};

const DescriptionPreview = (props: DProps) => {
  const title = <span style={{ color: props.color }}>{props.title}</span>;
  return (
    <div className="tournament-info">
      <Card title={title} className="tournament">
        {props.logo && (
          <img
            src={props.logo}
            alt={props.title}
            style={{
              width: 200,
              textAlign: 'center',
              margin: '0 auto 18px',
              display: 'block',
            }}
          />
        )}
        <ReactMarkdown linkTarget="_blank">{props.description}</ReactMarkdown>
        <br />
        <ReactMarkdown linkTarget="_blank">{props.disclaimer}</ReactMarkdown>
      </Card>
    </div>
  );
};

const layout = {
  labelCol: {
    span: 8,
  },
  wrapperCol: {
    span: 16,
  },
};

type Props = {
  mode: string;
};

const SettingsModalForm = (mprops: {
  visible: boolean;
  onCancel: () => void;
  setModalVisible: (v: boolean) => void;
  selectedGameRequest: GameRequest | null;
  setSelectedGameRequest: (gr: GameRequest) => void;
}) => {
  return (
    <Modal
      title="Set Game Request"
      visible={mprops.visible}
      onCancel={mprops.onCancel}
      className="seek-modal"
      okButtonProps={{ style: { display: 'none' } }}
      destroyOnClose
    >
      <SettingsForm
        setGameRequest={(gr) => {
          mprops.setSelectedGameRequest(gr);
          mprops.setModalVisible(false);
        }}
        gameRequest={mprops.selectedGameRequest}
      />
    </Modal>
  );
};

export const TourneyEditor = (props: Props) => {
  const { useState } = useMountedState();
  const [description, setDescription] = useState('');
  const [disclaimer, setDisclaimer] = useState('');
  const [name, setName] = useState('');
  const [color, setColor] = useState('');
  const [logo, setLogo] = useState('');
  const [settingsModalVisible, setSettingsModalVisible] = useState(false);
  const [
    selectedGameRequest,
    setSelectedGameRequest,
  ] = useState<GameRequest | null>(null);
  const [form] = Form.useForm();

  const onSearch = async (val: string) => {
    const tmreq = new GetTournamentMetadataRequest();
    tmreq.setSlug(val);

    try {
      const m = await postProto(
        TournamentMetadataResponse,
        'tournament_service.TournamentService',
        'GetTournamentMetadata',
        tmreq
      );
      const metadata = m.getMetadata();
      setDescription(metadata?.getDescription()!);
      setDisclaimer(metadata?.getDisclaimer() || '');
      setName(metadata?.getName()!);
      setColor(metadata?.getColor() || '');
      setLogo(metadata?.getLogo() || '');
      setSelectedGameRequest(metadata?.getDefaultClubSettings() || null);
      form.setFieldsValue({
        name: metadata?.getName(),
        description: metadata?.getDescription(),
        slug: metadata?.getSlug(),
        id: metadata?.getId(),
        type: metadata?.getType(),
        directors: m.getDirectorsList().join(', '),
        freeformItems: metadata?.getFreeformClubSettingFieldsList(),
        boardStyle: metadata?.getBoardStyle(),
        tileStyle: metadata?.getTileStyle(),
        disclaimer: metadata?.getDisclaimer(),
        logo: metadata?.getLogo(),
        color: metadata?.getColor(),
        privateAnalysis: metadata?.getPrivateAnalysis() || false,
      });
    } catch (err) {
      message.error({
        content: 'Error: ' + err.response?.data?.msg,
        duration: 5,
      });
    }
  };
  const onFinish = (vals: Store) => {
    console.log('vals', vals);
    let apicall = '';
    let obj = {};

    const reverseTypeMap = {
      [TType.CHILD]: 'CHILD',
      [TType.CLUB]: 'CLUB',
      [TType.STANDARD]: 'STANDARD',
      [TType.LEGACY]: 'LEGACY',
    };

    const jsontype = reverseTypeMap[vals.type as TTypeMap[keyof TTypeMap]];

    if (props.mode === 'new') {
      apicall = 'NewTournament';
      const directors = (vals.directors as string)
        .split(',')
        .map((u) => u.trim());

      obj = {
        name: vals.name,
        description: vals.description,
        slug: vals.slug,
        type: jsontype,
        director_usernames: directors,
        freeformClubSettingFields: vals.freeformItems,
        defaultClubSettings: selectedGameRequest
          ? selectedGameRequest.toObject()
          : undefined,
      };
    } else if (props.mode === 'edit') {
      apicall = 'SetTournamentMetadata';
      obj = {
        metadata: {
          id: vals.id,
          name: vals.name,
          description: vals.description,
          slug: vals.slug,
          type: jsontype,
          defaultClubSettings: selectedGameRequest
            ? selectedGameRequest.toObject()
            : undefined,
          freeformClubSettingFields: vals.freeformItems,
          boardStyle: vals.boardStyle,
          tileStyle: vals.tileStyle,
          disclaimer: vals.disclaimer,
          logo: vals.logo,
          color: vals.color,
          privateAnalysis: vals.privateAnalysis,
        },
      };
    }

    axios
      .post<{}>(toAPIUrl('tournament_service.TournamentService', apicall), obj)
      .then((resp) => {
        message.info({
          content:
            'Tournament ' + (props.mode === 'new' ? 'created' : 'updated'),
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
  const onDescriptionChange = (evt: React.ChangeEvent<HTMLTextAreaElement>) => {
    setDescription(evt.target.value);
  };
  const onDisclaimerChange = (evt: React.ChangeEvent<HTMLTextAreaElement>) => {
    setDisclaimer(evt.target.value);
  };
  const onNameChange = (evt: React.ChangeEvent<HTMLInputElement>) => {
    setName(evt.target.value);
  };
  const onColorChange = (evt: React.ChangeEvent<HTMLInputElement>) => {
    setColor(evt.target.value);
  };
  const onLogoChange = (evt: React.ChangeEvent<HTMLInputElement>) => {
    setLogo(evt.target.value);
  };

  const addDirector = () => {
    const director = prompt('Enter a new director username to add:');
    if (!director) {
      return;
    }
    axios
      .post<{}>(
        toAPIUrl('tournament_service.TournamentService', 'AddDirectors'),
        {
          id: form.getFieldValue('id'),
          // Need a non-zero "rating" for director..
          persons: [{ id: director, rating: 1 }],
        }
      )
      .then((resp) => {
        message.info({
          content: 'Director successfully added',
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

  const removeDirector = () => {
    const director = prompt('Enter a director username to remove:');
    if (!director) {
      return;
    }
    axios
      .post<{}>(
        toAPIUrl('tournament_service.TournamentService', 'RemoveDirectors'),
        {
          id: form.getFieldValue('id'),
          persons: [{ id: director }],
        }
      )
      .then((resp) => {
        message.info({
          content: 'Director successfully removed',
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
    <div className="tourney-editor">
      <Form {...layout} hidden={props.mode === 'new'}>
        <Form.Item label="Tournament URL">
          <Search
            placeholder="/tournament/catcam"
            onSearch={onSearch}
            style={{ width: 300 }}
            enterButton="Search"
            size="large"
          />
        </Form.Item>
      </Form>
      <Divider />
      <Row>
        <Col span={12}>
          <Form {...layout} form={form} layout="horizontal" onFinish={onFinish}>
            <Form.Item name="name" label="Tournament Name">
              <Input onChange={onNameChange} />
            </Form.Item>

            <Form.Item name="slug" label="Tournament Slug (URL)">
              <Input />
            </Form.Item>

            <Form.Item
              name="id"
              label="Tournament ID"
              hidden={props.mode === 'new'}
            >
              <Input disabled />
            </Form.Item>

            <Form.Item name="directors" label="Directors">
              <Input
                disabled={props.mode === 'edit'}
                placeholder="comma-separated usernames"
              />
            </Form.Item>

            <Form.Item label="(Modify Directors)">
              <Button hidden={props.mode === 'new'} onClick={addDirector}>
                Add a Director
              </Button>
              <Button hidden={props.mode === 'new'} onClick={removeDirector}>
                Remove a Director
              </Button>
            </Form.Item>

            <Form.Item name="type" label="Type">
              <Select style={{ zIndex: 10 }}>
                <Select.Option value={TType.CLUB}>Club</Select.Option>
                <Select.Option value={TType.CHILD}>
                  Club Session (or Child Tournament)
                </Select.Option>
                <Select.Option value={TType.STANDARD}>
                  Standard Tournament
                </Select.Option>
                <Select.Option value={TType.LEGACY}>
                  Legacy Tournament (Clubhouse mode)
                </Select.Option>
              </Select>
            </Form.Item>
            <Form.Item name="description" label="Description">
              <Input.TextArea onChange={onDescriptionChange} rows={20} />
            </Form.Item>
            <h3 hidden={props.mode === 'new'}>
              The following applies only to clubs and tournaments in clubhouse
              mode.
            </h3>
            <Form.Item hidden={props.mode === 'new'}>
              <Button
                htmlType="button"
                onClick={() => setSettingsModalVisible(true)}
              >
                Edit Club Game Settings
              </Button>
            </Form.Item>

            <Form.Item
              label="Selected game settings"
              hidden={props.mode === 'new'}
            >
              {DisplayedGameSetting(selectedGameRequest)}
            </Form.Item>

            <Form.Item
              name="freeformItems"
              label="Settings allowed to change"
              hidden={props.mode === 'new'}
            >
              <Select
                mode="multiple"
                allowClear
                placeholder="Please select"
                style={{ zIndex: 10 }}
              >
                <Select.Option value="lexicon">Lexicon</Select.Option>
                <Select.Option value="time">Time Settings</Select.Option>
                <Select.Option value="challenge_rule">
                  Challenge Rule
                </Select.Option>
                <Select.Option value="rating_mode">
                  Rated / Unrated
                </Select.Option>
                <Select.Option value="variant_name">
                  Classic / WordSmog
                </Select.Option>
              </Select>
            </Form.Item>

            <h3 hidden={props.mode === 'new'}>
              The following will usually not be set.
            </h3>
            <Form.Item
              name="boardStyle"
              label="Board theme (optional)"
              hidden={props.mode === 'new'}
            >
              <Input />
            </Form.Item>
            <Form.Item
              name="tileStyle"
              label="Tile theme (optional)"
              hidden={props.mode === 'new'}
            >
              <Input />
            </Form.Item>
            <Form.Item
              name="disclaimer"
              label="Disclaimer (optional)"
              hidden={props.mode === 'new'}
            >
              <Input.TextArea onChange={onDisclaimerChange} rows={20} />
            </Form.Item>
            <Form.Item
              name="logo"
              label="Logo URL (optional)"
              hidden={props.mode === 'new'}
            >
              <Input onChange={onLogoChange} />
            </Form.Item>
            <Form.Item
              name="color"
              label="Hex Color (optional)"
              hidden={props.mode === 'new'}
            >
              <Input onChange={onColorChange} />
            </Form.Item>

            <Form.Item
              name="privateAnalysis"
              label="Private Analysis (optional)"
              hidden={props.mode === 'new'}
              valuePropName="checked"
            >
              <Switch />
            </Form.Item>

            <Form.Item wrapperCol={{ ...layout.wrapperCol, offset: 4 }}>
              <Button type="primary" htmlType="submit">
                Submit
              </Button>
            </Form.Item>
          </Form>
        </Col>
        <DescriptionPreview
          description={description}
          disclaimer={disclaimer}
          title={name}
          color={color}
          logo={logo}
        />
        <SettingsModalForm
          visible={settingsModalVisible}
          onCancel={() => setSettingsModalVisible(false)}
          setModalVisible={(v: boolean) => setSettingsModalVisible(v)}
          selectedGameRequest={selectedGameRequest}
          setSelectedGameRequest={setSelectedGameRequest}
        />
      </Row>
    </div>
  );
};
