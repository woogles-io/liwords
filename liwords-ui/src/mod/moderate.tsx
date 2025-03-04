import { useCallback, useEffect, useState } from "react";

import { message, Form, Select, InputNumber, Input, Button } from "antd";
import { ExclamationCircleOutlined } from "@ant-design/icons";
import { flashError, useClient } from "../utils/hooks/connect";
import {
  ModActionsListSchema,
  ModActionsMapSchema,
  ModService,
} from "../gen/api/proto/mod_service/mod_service_pb";
import {
  EmailType,
  ModActionsList,
  ModActionsMap,
} from "../gen/api/proto/mod_service/mod_service_pb";
import { ModActionType } from "../gen/api/proto/mod_service/mod_service_pb";
import { HookAPI } from "antd/lib/modal/useModal";
import { Client } from "@connectrpc/connect";
import { create, toJsonString } from "@bufbuild/protobuf";
import { getEnumValue } from "../utils/protobuf";

type ModProps = {
  userID: string;
};

const Moderation = (props: ModProps) => {
  const [activeActions, setActiveActions] = useState<ModActionsMap>(
    create(ModActionsMapSchema, {}),
  );
  const [actionsHistory, setActionsHistory] = useState<ModActionsList>(
    create(ModActionsListSchema, {}),
  );

  const modClient = useClient(ModService);

  const onFinish = async (values: { [key: string]: string | number }) => {
    const actionType = getEnumValue(ModActionType, values.action as string);
    const emailType = getEnumValue(EmailType, values.emailType as string);
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
        content: "Applied mod action",
        duration: 2,
      });
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
          <InputNumber inputMode="numeric" min={0} max={720 * 6} />
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
        {toJsonString(ModActionsMapSchema, activeActions, { prettySpaces: 2 })}
      </pre>
      <h3>Moderation history</h3>
      <pre
        className="readable-text-color"
        style={{ maxHeight: 200, overflowY: "scroll" }}
      >
        {toJsonString(ModActionsListSchema, actionsHistory, {
          prettySpaces: 2,
        })}
      </pre>
    </div>
  );
};

export const moderateUser = (
  modal: HookAPI,
  uuid: string,
  username: string,
) => {
  modal.info({
    title: (
      <p className="readable-text-color">Moderation for user {username}</p>
    ),
    icon: <ExclamationCircleOutlined />,
    content: <Moderation userID={uuid} />,
    onOk() {
      console.log("ok");
    },
    onCancel() {
      console.log("no");
    },
    width: 800,
    maskClosable: true,
  });
};

export const deleteChatMessage = async (
  userid: string,
  msgid: string,
  channel: string,
  modClient: Client<typeof ModService>,
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
      content: "Removed chat",
      duration: 2,
    });
  } catch (e) {
    flashError(e);
  }
};
