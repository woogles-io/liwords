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
import { toAPIUrl } from '../api/api';
import { TournamentMetadataResponse } from '../gen/api/proto/tournament_service/tournament_service_pb';

type DProps = {
  markdown: string;
};

const DescriptionPreview = (props: DProps) => {
  return (
    <div className="tournament-info">
      <Card title="Tournament Information">
        <ReactMarkdown linkTarget="_blank">{props.markdown}</ReactMarkdown>
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

  // const [formVals, setFormVals] = useState({
  //   name: '',
  //   description: '',
  //   slug: '',
  //   id: '',
  //   type: 'STANDARD',
  //   directors: '',
  // });
  const [form] = Form.useForm();

  const onSearch = (val: string) => {
    axios
      .post<TournamentMetadataResponse.AsObject>(
        toAPIUrl(
          'tournament_service.TournamentService',
          'GetTournamentMetadata'
        ),
        {
          slug: val,
        }
      )
      .then((resp) => {
        setDescription(resp.data.description);

        form.setFieldsValue({
          name: resp.data.name,
          description: resp.data.description,
          slug: resp.data.slug,
          id: resp.data.id,
          type: resp.data.type,
          directors: resp.data.directorsList.join(', '),
        });
      })
      .catch((err) => {
        message.error({
          content: 'Error: ' + err.response?.data?.msg,
          duration: 5,
        });
      });
  };
  const onFinish = (vals: Store) => {
    console.log('vals', vals);
    let apicall = '';
    let obj = {};
    if (props.mode === 'new') {
      apicall = 'NewTournament';
      const directors = (vals.directors as string)
        .split(',')
        .map((u) => u.trim());
      obj = {
        name: vals.name,
        description: vals.description,
        slug: vals.slug,
        type: vals.type,
        director_usernames: directors,
      };
    } else if (props.mode === 'edit') {
      apicall = 'SetTournamentMetadata';
      obj = {
        id: vals.id,
        name: vals.name,
        description: vals.description,
        slug: vals.slug,
        type: vals.type,
      };
    }

    axios
      .post<{}>(toAPIUrl('tournament_service.TournamentService', apicall), obj)
      .then(() => {
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
          persons: {
            [director]: 10, // or whatever number?
          },
        }
      )
      .then(() => {
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
      .then(() => {
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
    <>
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
              <Input />
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
                <Select.Option value="CLUB">Club</Select.Option>
                <Select.Option value="CHILD">
                  Club Session (or Child Tournament)
                </Select.Option>
                <Select.Option value="STANDARD">
                  Standard Tournament
                </Select.Option>
                <Select.Option value="LEGACY">
                  Legacy Tournament (Clubhouse mode)
                </Select.Option>
              </Select>
            </Form.Item>

            <Form.Item name="description" label="Description">
              <Input.TextArea onChange={onDescriptionChange} rows={20} />
            </Form.Item>

            <Form.Item wrapperCol={{ ...layout.wrapperCol, offset: 4 }}>
              <Button type="primary" htmlType="submit">
                Submit
              </Button>
            </Form.Item>
          </Form>
        </Col>
        <DescriptionPreview markdown={description} />
      </Row>
    </>
  );
};
