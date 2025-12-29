import { useMutation, useQuery } from "@connectrpc/connect-query";
import {
  Button,
  Form,
  Input,
  List,
  message,
  Select,
  Table,
  Tag,
  Typography,
} from "antd";
import { TablePaginationConfig } from "antd/lib/table/interface";
import {
  assignRole,
  getRoleMetadata,
  getUsersWithRoles,
  unassignRole,
} from "../gen/api/proto/user_service/user_service-AuthorizationService_connectquery";
import { useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";

const layout = {
  labelCol: {
    span: 2,
  },
  wrapperCol: {
    span: 16,
  },
};

export const PermsAndRoles = () => {
  const [username, setUsername] = useState("");
  const [role, setRole] = useState("");
  const { data: roleMetadata } = useQuery(getRoleMetadata);
  const { data: usersWithRoles } = useQuery(
    getUsersWithRoles,
    {
      roles: roleMetadata?.rolesWithPermissions.map((v) => v.roleName),
    },
    { enabled: !!roleMetadata },
  );
  const addRole = useMutation(assignRole);
  const removeRole = useMutation(unassignRole);
  const queryClient = useQueryClient();

  const cachedUsersWithRoles = useMemo(() => {
    if (!usersWithRoles) {
      return [];
    }

    const m: Record<string, string[]> = {};

    usersWithRoles.userAndRoleObjs.forEach(({ username, roleName }) => {
      if (username) {
        (m[username] ||= []).push(roleName);
      }
    });

    return Object.entries(m)
      .map(([username, roleNames]) => ({
        username,
        lowerUsername: username.toLowerCase(),
        roleNames,
        roleName: roleNames.join(", "),
        priority: roleNames.some((r) => r !== "League Player") ? 1 : 2,
        key: username,
      }))
      .sort((a, b) => {
        if (a.priority < b.priority) return -1;
        if (a.priority > b.priority) return 1;
        return a.username.localeCompare(b.username);
      });
  }, [usersWithRoles]);

  const filteredUsersWithRoles = useMemo(() => {
    if (!username && !role) {
      return cachedUsersWithRoles;
    }
    const lowerUsername = username.toLowerCase();
    const exactMatches: typeof cachedUsersWithRoles = [];
    const matches = cachedUsersWithRoles.filter((elt) => {
      if (lowerUsername && elt.lowerUsername === lowerUsername) {
        exactMatches.push(elt); // put exact username match first, regardless of role.
        return false;
      }
      return (
        (!lowerUsername || elt.lowerUsername.includes(lowerUsername)) &&
        (!role || elt.roleNames.includes(role))
      );
    });
    return exactMatches.length > 0 ? [...exactMatches, ...matches] : matches;
  }, [cachedUsersWithRoles, username, role]);

  const [pagination, setPagination] = useState<TablePaginationConfig>({
    pageSize: 20,
  });

  return (
    <>
      <div>
        <h3>Add or remove a role from a user</h3>
        <Form {...layout} style={{ marginBottom: 60 }}>
          <Form.Item label="Username" name="username">
            <Input onChange={(e) => setUsername(e.target.value)} />
          </Form.Item>
          <Form.Item label="Role" name="role">
            <Select onChange={(v) => setRole(v)}>
              <Select.Option value="">
                {"" /* provide a blank option to reset the filter */}
              </Select.Option>
              {roleMetadata?.rolesWithPermissions.map((v) => (
                <Select.Option key={v.roleName} value={v.roleName}>
                  {v.roleName}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item hidden={!username || !role}>
            <Button
              onClick={async () => {
                try {
                  await addRole.mutateAsync({ username, roleName: role });
                  await queryClient.refetchQueries({
                    queryKey: [
                      "connect-query",
                      { methodName: "GetUsersWithRoles" },
                    ],
                  });
                } catch (error) {
                  message.error({
                    content: "Error adding role: " + String(error),
                  });
                }
              }}
            >
              ADD the role {role} to {username}
            </Button>
            <Button
              danger
              onClick={async () => {
                try {
                  await removeRole.mutateAsync({ username, roleName: role });
                  await queryClient.refetchQueries({
                    queryKey: [
                      "connect-query",
                      { methodName: "GetUsersWithRoles" },
                    ],
                  });
                } catch (error) {
                  message.error({
                    content: "Error removing role: " + String(error),
                  });
                }
              }}
            >
              REMOVE the role {role} from {username}
            </Button>
          </Form.Item>
        </Form>
      </div>
      <h3>Current users with roles</h3>
      <Table
        size="small"
        pagination={pagination}
        onChange={(v) => {
          setPagination(v);
        }}
        dataSource={filteredUsersWithRoles}
        columns={[
          { title: "Username", dataIndex: "username", key: "username" },
          { title: "Role", dataIndex: "roleName", key: "roleName" },
        ]}
      />
      <List
        header={<h3>Roles with permissions</h3>}
        dataSource={roleMetadata?.rolesWithPermissions.map((r) => ({
          ...r,
          key: r.roleName,
        }))}
        renderItem={(item) => (
          <List.Item>
            <Typography.Text strong style={{ marginRight: 24 }}>
              {item.roleName}
            </Typography.Text>{" "}
            {item.permissions.map((p) => (
              <Tag key={item.roleName + ":" + p} color="green">
                {p}
              </Tag>
            ))}
          </List.Item>
        )}
      ></List>
    </>
  );
};
