import { useQuery } from "@connectrpc/connect-query";
import { Button, Form, Input, message, Switch, Table } from "antd";
import {
  getUserDetails,
  searchEmail,
} from "../gen/api/proto/config_service/config_service-ConfigService_connectquery";
import { useState } from "react";
import moment from "moment";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { UserDetailsResponse } from "../gen/api/proto/config_service/config_service_pb";

const layout = {
  labelCol: {
    span: 4,
  },
  wrapperCol: {
    span: 16,
  },
};

export const UserDetails = () => {
  const [username, setUsername] = useState("");
  const [partialEmail, setPartialEmail] = useState("");
  const [searchByEmail, setSearchByEmail] = useState(false);
  const { data: searchUserData, refetch: refetchByUsername } = useQuery(
    getUserDetails,
    { username: username },
    { enabled: false, retry: false },
  );
  const { data: searchEmailData, refetch: refetchByEmail } = useQuery(
    searchEmail,
    { partialEmail: partialEmail },
    { enabled: false, retry: false },
  );

  const columns = [
    {
      title: "Username",
      dataIndex: "username",
      key: "username",
    },
    {
      title: "Email",
      dataIndex: "email",
      key: "email",
    },
    {
      title: "Joined",
      dataIndex: "created",
      key: "created",
    },
    {
      title: "Birth Date",
      dataIndex: "birthDate",
      key: "birthDate",
    },
    {
      title: "UUID",
      dataIndex: "uuid",
      key: "uuid",
    },
  ];
  let tableDataSource;
  const dataSource: UserDetailsResponse[] | undefined = !searchByEmail
    ? searchUserData
      ? [searchUserData]
      : undefined
    : searchEmailData?.users;
  if (dataSource?.length) {
    tableDataSource = dataSource.map((item) => ({
      ...item,
      created: item.created
        ? moment(timestampDate(item.created)).toISOString()
        : undefined,
      key: item.uuid,
    }));
  }

  return (
    <>
      <h4>Enter a username OR a partial email to search for user details:</h4>
      <Form {...layout} style={{ marginTop: 16, marginBottom: 16 }}>
        <Form.Item label="Search by email">
          <Switch
            value={searchByEmail}
            onChange={() => setSearchByEmail((s) => !s)}
            className="dark-toggle"
          />
        </Form.Item>
        <Form.Item label={searchByEmail ? "Email" : "Username"}>
          <Input
            onChange={(e) => {
              if (searchByEmail) {
                setPartialEmail(e.target.value);
                setUsername("");
              } else {
                setPartialEmail("");
                setUsername(e.target.value);
              }
            }}
            value={searchByEmail ? partialEmail : username}
          />
        </Form.Item>
        <Form.Item>
          <Button
            onClick={async () => {
              try {
                if (searchByEmail) {
                  if (partialEmail) {
                    await refetchByEmail({ throwOnError: true });
                  }
                } else {
                  if (username) {
                    await refetchByUsername({ throwOnError: true });
                  }
                }
              } catch (e) {
                message.error({ content: "Error: " + String(e) });
              }
            }}
          >
            Search for user
          </Button>
        </Form.Item>
      </Form>
      <h3>Results</h3>
      <Table dataSource={tableDataSource} columns={columns} size="small" />
    </>
  );
};
