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
} from 'antd';
import { Store } from 'antd/lib/form/interface';
import '../App.scss';
import '../lobby/lobby.scss';
import 'antd/dist/antd.css';
import ReactMarkdown from 'react-markdown';
import { useMountedState } from '../utils/mounted';
import { postBinary, toAPIUrl } from '../api/api';
import {
  GetTournamentMetadataRequest,
  TournamentMetadataResponse,
  TType,
  TTypeMap,
} from '../gen/api/proto/tournament_service/tournament_service_pb';

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

export const TourneyEditor = (props: Props) => {
  const { useState } = useMountedState();
  const [description, setDescription] = useState('');
  const [disclaimer, setDisclaimer] = useState('');
  const [name, setName] = useState('');
  const [color, setColor] = useState('');
  const [logo, setLogo] = useState('');
  const [form] = Form.useForm();

  const onSearch = async (val: string) => {
    const tmreq = new GetTournamentMetadataRequest();
    tmreq.setSlug(val);

    try {
      const resp = await postBinary(
        'tournament_service.TournamentService',
        'GetTournamentMetadata',
        tmreq
      );
      const m = TournamentMetadataResponse.deserializeBinary(resp.data);
      const metadata = m.getMetadata();
      setDescription(metadata?.getDescription()!);
      setDisclaimer(metadata?.getDisclaimer() || '');
      setName(metadata?.getName()!);
      setColor(metadata?.getColor() || '');
      setLogo(metadata?.getLogo() || '');
      form.setFieldsValue({
        name: metadata?.getName(),
        description: metadata?.getDescription(),
        slug: metadata?.getSlug(),
        id: metadata?.getId(),
        type: metadata?.getType(),
        directors: m.getDirectorsList().join(', '),
        boardStyle: metadata?.getBoardStyle(),
        tileStyle: metadata?.getTileStyle(),
        disclaimer: metadata?.getDisclaimer(),
        logo: metadata?.getLogo(),
        color: metadata?.getColor(),
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
          boardStyle: vals.boardStyle,
          tileStyle: vals.tileStyle,
          disclaimer: vals.disclaimer,
          logo: vals.logo,
          color: vals.color,
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

  // XXX: this function shouldn't require an "int"
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
          persons: {
            [director]: 10,
          },
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
              <Select>
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
      </Row>
    </div>
  );
};
