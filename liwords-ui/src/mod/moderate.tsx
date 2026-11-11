import React, { useCallback, useEffect, useState } from "react";

import { message, Form, Select, InputNumber, Input, Button, Table } from "antd";
import { ExclamationCircleOutlined } from "@ant-design/icons";
import { flashError, useClient } from "../utils/hooks/connect";
import {
  ModActionsListSchema,
  ModActionsMapSchema,
  ModService,
} from "../gen/api/proto/mod_service/mod_service_pb";
import {
  EmailChange,
  EmailType,
  ModActionsList,
  ModActionsMap,
} from "../gen/api/proto/mod_service/mod_service_pb";
import { ModActionType } from "../gen/api/proto/mod_service/mod_service_pb";
import { HookAPI } from "antd/lib/modal/useModal";
import { Client } from "@connectrpc/connect";
import { create, toJsonString } from "@bufbuild/protobuf";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import moment from "moment";
import { getEnumValue } from "../utils/protobuf";
import { useLoginStateStoreContext } from "../store/store";
import { hasPermission, Perm } from "./perms";

type ModProps = {
  userID: string;
};

// The profile fields an admin can clear, with their menu labels. The server
// keeps each field's old value in the action.
const profileFieldRemovals = [
  ["REMOVE_FIRST_NAME", "Remove first name"],
  ["REMOVE_LAST_NAME", "Remove last name"],
  ["REMOVE_ABOUT", "Remove player bio"],
  ["REMOVE_AVATAR", "Remove avatar"],
] as const;

// ChangeEmail sets the user's email at once; the server sends no mail to
// either address, and refuses one that is malformed or another account's.
export const ChangeEmail = (props: ModProps & { onChanged?: () => void }) => {
  const modClient = useClient(ModService);

  const onFinish = async (values: { email: string }) => {
    try {
      await modClient.changeEmail({
        userId: props.userID,
        email: values.email,
      });
      message.info({
        content: "Changed email",
        duration: 2,
      });
      props.onChanged?.();
    } catch (e) {
      flashError(e);
    }
  };

  return (
    <React.Fragment>
      <h3>Change email</h3>
      <Form name="change-email" onFinish={onFinish}>
        <Form.Item name="email" label="New email" rules={[{ required: true }]}>
          <Input></Input>
        </Form.Item>
        <Form.Item>
          <Button type="primary" htmlType="submit">
            Change email
          </Button>
        </Form.Item>
      </Form>
    </React.Fragment>
  );
};

// EmailChanges lists the user's email changes, newest first, by the user or
// by an admin. Only admins may see them. A new refresh value reloads it.
export const EmailChanges = (props: ModProps & { refresh: number }) => {
  const modClient = useClient(ModService);
  const [changes, setChanges] = useState<Array<EmailChange>>([]);

  useEffect(() => {
    (async () => {
      try {
        const resp = await modClient.listEmailChanges({ userId: props.userID });
        setChanges(resp.changes);
      } catch (e) {
        flashError(e);
      }
    })();
  }, [modClient, props.userID, props.refresh]);

  return (
    <React.Fragment>
      <h3>Email changes</h3>
      <Table
        className="readable-text-color"
        dataSource={changes.map((c, i) => ({ ...c, key: i }))}
        pagination={false}
        size="small"
        columns={[
          {
            title: "Date",
            key: "changedAt",
            render: (_, c: EmailChange) =>
              c.changedAt
                ? moment(timestampDate(c.changedAt)).format("YYYY-MM-DD HH:mm")
                : "",
          },
          { title: "Old", dataIndex: "oldEmail", key: "oldEmail" },
          { title: "New", dataIndex: "newEmail", key: "newEmail" },
          { title: "Who", dataIndex: "changedByUsername", key: "who" },
        ]}
      />
    </React.Fragment>
  );
};

const Moderation = (props: ModProps) => {
  const { loginState } = useLoginStateStoreContext();
  const canRemoveProfileFields = hasPermission(
    loginState.permissions,
    Perm.CanResetAndDeleteAccounts,
  );
  const [activeActions, setActiveActions] = useState<ModActionsMap>(
    create(ModActionsMapSchema, {}),
  );
  const [emailChangesRefresh, setEmailChangesRefresh] = useState(0);
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

  // A removed avatar's old URL, kept in the action, shows as a link (never
  // as an image: some early uploads are very large).
  const removedAvatars = actionsHistory.actions.filter(
    (a) => a.type === ModActionType.REMOVE_AVATAR && a.chatText !== "",
  );

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
            {canRemoveProfileFields
              ? profileFieldRemovals.map(([value, label]) => (
                  <Select.Option key={value} value={value}>
                    {label}
                  </Select.Option>
                ))
              : null}
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

      {canRemoveProfileFields ? (
        <React.Fragment>
          <ChangeEmail
            userID={props.userID}
            onChanged={() => setEmailChangesRefresh((r) => r + 1)}
          />
          <EmailChanges userID={props.userID} refresh={emailChangesRefresh} />
        </React.Fragment>
      ) : null}

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
      {removedAvatars.length > 0 ? (
        <div>
          <h3>Removed avatars</h3>
          <ul className="readable-text-color">
            {removedAvatars.map((a) => (
              <li key={`${a.startTime?.seconds}-${a.chatText}`}>
                {/^https?:\/\//.test(a.chatText) ? (
                  <a
                    href={a.chatText}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    {a.chatText}
                  </a>
                ) : (
                  a.chatText
                )}
              </li>
            ))}
          </ul>
        </div>
      ) : null}
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
