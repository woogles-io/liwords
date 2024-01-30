import React, { useCallback, useEffect, useState } from 'react';

import { message, Modal, Form, Select, InputNumber, Input, Button } from 'antd';
import { ExclamationCircleOutlined } from '@ant-design/icons';
import { flashError, useClient } from '../utils/hooks/connect';
import { ModService } from '../gen/api/proto/mod_service/mod_service_connectweb';
import { proto3 } from '@bufbuild/protobuf';
import {
  EmailType,
  ModActionsList,
  ModActionsMap,
} from '../gen/api/proto/mod_service/mod_service_pb';
import { PromiseClient } from '@domino14/connect-web';
import { ModActionType } from '../gen/api/proto/mod_service/mod_service_pb';

type ModProps = {
  userID: string;
  destroy: () => void;
};

const Moderation = (props: ModProps) => {
  const [activeActions, setActiveActions] = useState<ModActionsMap>(
    new ModActionsMap()
  );
  const [actionsHistory, setActionsHistory] = useState<ModActionsList>(
    new ModActionsList()
  );

  const modClient = useClient(ModService);

  const onFinish = async (values: { [key: string]: string | number }) => {
    const actionType = proto3
      .getEnumType(ModActionType)
      .findName(values.action as string)?.no;
    const emailType = proto3
      .getEnumType(EmailType)
      .findName(values.emailType as string)?.no;
    const obj = {
      actions: [
        {
          userId: props.userID,
          type: actionType,
          note: values.note as string,
          duration: Math.round((values.duration as number) * 3600),
          emailType: emailType,
        },
      ],
    };

    try {
      await modClient.applyActions(obj);
      message.info({
        content: 'Applied mod action',
        duration: 2,
      });
      props.destroy();
    } catch (e) {
      flashError(e);
    }
  };

  useEffect(() => {
    const obj = {
      userId: props.userID,
    };
    (async () => {
      const actions = await modClient.getActions(obj);
      setActiveActions(actions);
      const actionHistory = await modClient.getActionHistory(obj);
      actionHistory.actions.reverse();
      setActionsHistory(actionHistory);
    })();
  }, [modClient, props.userID]);

  const [form] = Form.useForm();

  const handleSetShortDuration = useCallback(() => {
    form.setFieldsValue({
      duration: 0.0003, // 1.08 seconds
    });
  }, [form]);

  return (
    <div>
      <h3>Apply mod action</h3>
      <Form
        name="modder"
        onFinish={onFinish}
        initialValues={{ duration: 1 }}
        form={form}
      >
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

        <div className="readable-text-color">
          Some actions will send an email to the user. Select the type of email
          from the list:
        </div>
        <Form.Item name="emailType" label="Email type">
          <Select>
            <Select.Option value="DEFAULT">Default</Select.Option>
            <Select.Option value="CHEATING">Cheating</Select.Option>
          </Select>
        </Form.Item>

        <Form.Item>
          <Button type="primary" htmlType="submit">
            Apply action
          </Button>
          <Button onClick={handleSetShortDuration}>Set short duration</Button>
        </Form.Item>
      </Form>

      <h3>Active mod actions</h3>
      <pre className="readable-text-color">
        {activeActions.toJsonString({ prettySpaces: 2 })}
      </pre>
      <h3>Moderation history</h3>
      <pre
        className="readable-text-color"
        style={{ maxHeight: 200, overflowY: 'scroll' }}
      >
        {actionsHistory.toJsonString({ prettySpaces: 2 })}
      </pre>
    </div>
  );
};

export const moderateUser = (uuid: string, username: string) => {
  const modal = Modal.info({
    title: (
      <p className="readable-text-color">Moderation for user {username}</p>
    ),
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

export const deleteChatMessage = async (
  userid: string,
  msgid: string,
  channel: string,
  modClient: PromiseClient<typeof ModService>
) => {
  const obj = {
    actions: [
      {
        userId: userid,
        type: ModActionType.REMOVE_CHAT,
        channel: channel,
        messageId: msgid,
      },
    ],
  };
  try {
    await modClient.applyActions(obj);
    message.info({
      content: 'Removed chat',
      duration: 2,
    });
  } catch (e) {
    flashError(e);
  }
};
