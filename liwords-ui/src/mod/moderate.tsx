import React, { useEffect } from 'react';

import {
  message,
  Modal,
  notification,
  Form,
  Select,
  InputNumber,
  Input,
  Button,
} from 'antd';
import { ExclamationCircleOutlined } from '@ant-design/icons';
import axios from 'axios';
import { toAPIUrl } from '../api/api';
import { useMountedState } from '../utils/mounted';

type ModProps = {
  userID: string;
  destroy: () => void;
};

type ModAction = {
  userId: string;
  type: string;
  duration: number;
  startTime: string;
  endTime: string;
  removedTime: string;
  channel: string;
  messageId: string;
  applierUserId: string;
  removerUserId: string;
  chatText: string;
  note: string;
};

type ActionsMap = { actions: { [name: string]: ModAction } };
type ActionsList = { actions: Array<ModAction> };

const Moderation = (props: ModProps) => {
  const { useState } = useMountedState();

  const [activeActions, setActiveActions] = useState<ActionsMap>({
    actions: {},
  });
  const [actionsHistory, setActionsHistory] = useState<ActionsList>({
    actions: [],
  });

  const onFinish = (values: { [key: string]: string | number }) => {
    const obj = {
      actions: [
        {
          userId: props.userID,
          type: values.action,
          note: values.note,
          duration: Math.round((values.duration as number) * 3600),
        },
      ],
    };

    axios
      .post<{}>(toAPIUrl('mod_service.ModService', 'ApplyActions'), obj)
      .then((e) => {
        message.info({
          content: 'Applied mod action',
          duration: 2,
        });
        props.destroy();
      })
      .catch((e) => {
        if (e.response) {
          notification.error({
            message: 'Error',
            description: e.response.data.msg,
            duration: 4,
          });
        } else {
          console.log(e);
        }
      });
  };

  useEffect(() => {
    const obj = {
      user_id: props.userID,
    };

    axios
      .post<ActionsMap>(toAPIUrl('mod_service.ModService', 'GetActions'), obj)
      .then((a) => {
        setActiveActions(a.data);
      });

    axios
      .post<ActionsList>(
        toAPIUrl('mod_service.ModService', 'GetActionHistory'),
        obj
      )
      .then((a) => {
        // newest first
        a.data.actions.reverse();
        setActionsHistory(a.data);
      });
  }, [props.userID]);

  return (
    <div>
      <h3>Apply mod action</h3>
      <Form name="modder" onFinish={onFinish} initialValues={{ duration: 1 }}>
        <Form.Item name="action" label="Action" rules={[{ required: true }]}>
          <Select>
            <Select.Option value="MUTE">Mute</Select.Option>
            <Select.Option value="SUSPEND_ACCOUNT">
              Suspend account
            </Select.Option>
            <Select.Option value="SUSPEND_RATED_GAMES">
              Suspend rated games
            </Select.Option>
            <Select.Option value="SUSPEND_GAMES">Suspend games</Select.Option>
            <Select.Option value="RESET_RATINGS">Reset ratings</Select.Option>
            <Select.Option value="RESET_STATS">Reset stats</Select.Option>
            <Select.Option value="RESET_STATS_AND_RATINGS">
              Reset stats and ratings
            </Select.Option>
            <Select.Option value="DELETE_ACCOUNT">Delete account</Select.Option>
          </Select>
        </Form.Item>

        <Form.Item
          name="duration"
          label="Duration, in hours. Use 0 for indefinite"
        >
          <InputNumber min={0} max={720 * 6} />
        </Form.Item>

        <Form.Item name="note" label="Optional note">
          <Input maxLength={200}></Input>
        </Form.Item>

        <Form.Item>
          <Button type="primary" htmlType="submit">
            Apply action
          </Button>
        </Form.Item>
      </Form>

      <h3>Active mod actions</h3>
      <pre>{JSON.stringify(activeActions, null, 2)}</pre>
      <h3>Moderation history</h3>
      <pre style={{ maxHeight: 200, overflowY: 'scroll' }}>
        {JSON.stringify(actionsHistory, null, 2)}
      </pre>
    </div>
  );
};

export const moderateUser = (uuid: string, username: string) => {
  const modal = Modal.info({
    title: `Moderation for user ${username}`,
    icon: <ExclamationCircleOutlined />,
    content: <Moderation userID={uuid} destroy={() => modal.destroy()} />,
    onOk() {
      console.log('ok');
    },
    onCancel() {
      console.log('no');
    },
    width: 800,
    maskClosable: true,
  });
};

export const deleteChatMessage = (
  userid: string,
  msgid: string,
  channel: string
) => {
  const obj = {
    actions: [
      {
        user_id: userid,
        type: 'REMOVE_CHAT',
        channel: channel,
        message_id: msgid,
      },
    ],
  };
  axios
    .post<{}>(toAPIUrl('mod_service.ModService', 'ApplyActions'), obj)
    .then(() => {
      message.info({
        content: 'Removed chat',
        duration: 2,
      });
    })
    .catch((e) => {
      if (e.response) {
        notification.error({
          message: 'Error',
          description: e.response.data.msg,
          duration: 4,
        });
      } else {
        console.log(e);
      }
    });
};
