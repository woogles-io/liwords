import { useQuery } from "@connectrpc/connect-query";
import { Button, Form, Input, message, Switch, Table } from "antd";
import {
  getUserDetails,
  searchEmail,
} from "../gen/api/proto/config_service/config_service-ConfigService_connectquery";
import { useState } from "react";
import { Link } from "react-router";
import moment from "moment";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { Timestamp } from "@bufbuild/protobuf/wkt";
import { UserDetailsResponse } from "../gen/api/proto/config_service/config_service_pb";

const layout = {
  labelCol: {
    span: 4,
  },
  wrapperCol: {
    span: 16,
  },
};

const fmtTime = (ts?: Timestamp) =>
  ts ? moment(timestampDate(ts)).format("YYYY-MM-DD HH:mm") : "";

// Geolocation happens in the viewer's browser, so user IPs are never sent to a
// third party by the server.
const IPLink = ({ ip }: { ip: string }) =>
  ip ? (
    <a
      href={`https://ipinfo.io/${encodeURIComponent(ip)}`}
      target="_blank"
      rel="noreferrer"
    >
      {ip}
    </a>
  ) : (
    <>—</>
  );

const matchLabels: { [key: string]: string } = {
  client_id: "Same browser",
  registration_client_id: "Same browser (at registration)",
  ip: "Same IP",
  registration_ip: "Same IP (at registration)",
};

const UserConnectionDetails = ({ user }: { user: UserDetailsResponse }) => {
  const clients = user.clients.map((c, i) => ({
    key: `${c.ip}-${c.clientId}-${i}`,
    ip: c.ip,
    clientId: c.clientId,
    firstSeen: fmtTime(c.firstSeen),
    lastSeen: fmtTime(c.lastSeen),
  }));
  const linked = user.linkedAccounts.map((l, i) => ({
    key: `${l.uuid}-${l.matchedOn}-${l.value}-${i}`,
    username: l.username,
    matchedOn: matchLabels[l.matchedOn] ?? l.matchedOn,
    value: l.value,
    lastSeen: fmtTime(l.lastSeen),
    suspended: l.suspended,
  }));
  const actions = user.activeActions.map((a, i) => ({
    key: `${a.type}-${i}`,
    type: a.type,
    start: fmtTime(a.start),
    end: a.end ? fmtTime(a.end) : "permanent",
    note: a.note,
  }));

  return (
    <div className="user-connection-details">
      <h3>Account</h3>
      <ul>
        <li>Verified: {user.verified ? "yes" : "no"}</li>
        <li>Notoriety: {user.notoriety}</li>
        {user.isBot && <li>Bot account</li>}
        <li>
          Registered from: <IPLink ip={user.registrationIp} />
          {user.registrationClientId && (
            <> / browser {user.registrationClientId}</>
          )}
        </li>
      </ul>

      <h3>Active moderator actions</h3>
      <Table
        dataSource={actions}
        size="small"
        pagination={false}
        locale={{ emptyText: "None" }}
        columns={[
          { title: "Action", dataIndex: "type", key: "type" },
          { title: "Start", dataIndex: "start", key: "start" },
          { title: "End", dataIndex: "end", key: "end" },
          { title: "Note", dataIndex: "note", key: "note" },
        ]}
      />

      <h3>Seen from</h3>
      <Table
        dataSource={clients}
        size="small"
        pagination={false}
        locale={{ emptyText: "Nothing recorded" }}
        columns={[
          {
            title: "IP",
            dataIndex: "ip",
            key: "ip",
            render: (ip: string) => <IPLink ip={ip} />,
          },
          {
            title: "Browser ID",
            dataIndex: "clientId",
            key: "clientId",
            render: (cid: string) => cid || "(not yet assigned)",
          },
          { title: "First seen", dataIndex: "firstSeen", key: "firstSeen" },
          { title: "Last seen", dataIndex: "lastSeen", key: "lastSeen" },
        ]}
      />

      <h3>Other accounts sharing an IP or browser</h3>
      <p>
        A shared browser is a strong signal; a shared IP is weaker (mobile
        carriers, VPNs, schools and offices put many people behind one address).
      </p>
      <Table
        dataSource={linked}
        size="small"
        pagination={false}
        locale={{ emptyText: "None found" }}
        columns={[
          {
            title: "Username",
            dataIndex: "username",
            key: "username",
            render: (username: string) => (
              <Link
                to={`/profile/${encodeURIComponent(username)}`}
                target="_blank"
              >
                {username}
              </Link>
            ),
          },
          { title: "Match", dataIndex: "matchedOn", key: "matchedOn" },
          {
            title: "Shared value",
            dataIndex: "value",
            key: "value",
            render: (value: string, row: { matchedOn: string }) =>
              row.matchedOn.startsWith("Same IP") ? (
                <IPLink ip={value} />
              ) : (
                value
              ),
          },
          { title: "Last seen", dataIndex: "lastSeen", key: "lastSeen" },
          {
            title: "Suspended",
            dataIndex: "suspended",
            key: "suspended",
            render: (s: boolean) => (s ? "yes" : ""),
          },
        ]}
      />
    </div>
  );
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
      render: (username: string) => (
        <Link to={`/profile/${encodeURIComponent(username)}`} target="_blank">
          {username}
        </Link>
      ),
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
      {!searchByEmail && searchUserData && (
        <UserConnectionDetails user={searchUserData} />
      )}
    </>
  );
};
