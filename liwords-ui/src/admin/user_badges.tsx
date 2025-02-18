import { useMutation, useQuery } from "@connectrpc/connect-query";
import { useState } from "react";
import { getBadgesMetadata } from "../gen/api/proto/user_service/user_service-ProfileService_connectquery";
import {
  assignBadge,
  getUsersForBadge,
  unassignBadge,
} from "../gen/api/proto/config_service/config_service-ConfigService_connectquery";
import { Button, Form, Input, message, Select } from "antd";
import { useQueryClient } from "@tanstack/react-query";

const layout = {
  labelCol: {
    span: 2,
  },
  wrapperCol: {
    span: 16,
  },
};

export const UserBadges = () => {
  const [username, setUsername] = useState("");
  const [badgeCode, setBadgeCode] = useState("");

  const { data: badgeMetadata } = useQuery(getBadgesMetadata);
  const { data: usersForBadgeData, refetch: refetchUsersForBadge } = useQuery(
    getUsersForBadge,
    {
      code: badgeCode,
    },
    { enabled: false, retry: false },
  );
  const addBadge = useMutation(assignBadge);
  const removeBadge = useMutation(unassignBadge);
  const queryClient = useQueryClient();

  return (
    <>
      <h3>Get users for a badge</h3>
      <Form {...layout} style={{ marginBottom: 60 }}>
        <Form.Item label="Badge Code" name="badge">
          <Select onChange={(e) => setBadgeCode(e)}>
            {badgeMetadata ? (
              Object.keys(badgeMetadata.badges).map((b) => (
                <Select.Option key={b} value={b}>
                  {b}
                </Select.Option>
              ))
            ) : (
              <Select.Option>loading</Select.Option>
            )}
          </Select>
        </Form.Item>

        <Form.Item>
          <Button
            onClick={async () => {
              try {
                await refetchUsersForBadge({ throwOnError: true });
              } catch (e) {
                message.error({ content: "Error: " + String(e) });
              }
            }}
          >
            Get users that have been assigned badge {badgeCode}
          </Button>
        </Form.Item>
      </Form>
      Users with badge {badgeCode}: {usersForBadgeData?.usernames.join(", ")}
      <h3 style={{ marginTop: 32 }}>Add or remove a badge from a user</h3>
      <Form {...layout} style={{ marginBottom: 60 }}>
        <Form.Item label="Username" name="username">
          <Input onChange={(e) => setUsername(e.target.value)} />
        </Form.Item>

        <Form.Item hidden={!username || !badgeCode}>
          <Button
            onClick={async () => {
              try {
                await addBadge.mutateAsync({ username, code: badgeCode });
                await refetchUsersForBadge({ throwOnError: true });
              } catch (error) {
                message.error({
                  content: "Error adding badge: " + String(error),
                });
              }
            }}
          >
            ADD the badge {badgeCode} to {username}
          </Button>
          <Button
            danger
            onClick={async () => {
              try {
                await removeBadge.mutateAsync({ username, code: badgeCode });
                await refetchUsersForBadge({ throwOnError: true });
              } catch (error) {
                message.error({
                  content: "Error removing role: " + String(error),
                });
              }
            }}
          >
            REMOVE the badge {badgeCode} from {username}
          </Button>
        </Form.Item>
      </Form>
    </>
  );
};
