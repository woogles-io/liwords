import { Button, Input, Select, Space, Table, Tag, Typography } from "antd";
import React, { useState } from "react";
import { Link } from "react-router";
import { ChatMessage } from "../gen/api/proto/ipc/chat_pb";
import {
  ChatChannelSummary,
  ModService,
} from "../gen/api/proto/mod_service/mod_service_pb";
import { moderateUser } from "../mod/moderate";
import { UsernameWithContext } from "../shared/usernameWithContext";
import { flashError, useClient } from "../utils/hooks/connect";

const channelTypes = [
  { value: "chat.game.", label: "Game (players)" },
  { value: "chat.gametv.", label: "Game (viewers)" },
  { value: "chat.pm.", label: "Private" },
  { value: "chat.tournament.", label: "Tournament / club" },
  { value: "chat.league.", label: "League" },
  { value: "chat.lobby", label: "Lobby" },
  { value: "", label: "All" },
];

export const formatTime = (ms: bigint): string =>
  ms ? new Date(Number(ms)).toISOString() : "";

// Adds a later page of channels, skipping any already listed.
export const mergeRows = (
  prev: Array<ChatChannelSummary>,
  more: Array<ChatChannelSummary>,
): Array<ChatChannelSummary> => {
  const seen = new Set(prev.map((r) => r.channel));
  return [...prev, ...more.filter((r) => !seen.has(r.channel))];
};

const csvField = (s: string): string =>
  /[",\n\r]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s;

export const chatsToCSV = (msgs: Array<ChatMessage>): string =>
  [
    "time_utc,channel,username,user_id,message_id,message",
    ...msgs.map((m) =>
      [
        formatTime(m.timestamp),
        m.channel,
        m.username,
        m.userId,
        m.id,
        m.message,
      ]
        .map(csvField)
        .join(","),
    ),
  ].join("\n") + "\n";

export const summariesToCSV = (rows: Array<ChatChannelSummary>): string =>
  [
    "channel,label,messages,first_utc,last_utc,senders,game_created_utc,game_updated_utc",
    ...rows.map((r) =>
      [
        r.channel,
        r.context?.label ?? "",
        String(r.messageCount),
        formatTime(r.firstTimestamp),
        formatTime(r.lastTimestamp),
        r.senders.map((s) => `${s.username}:${s.count}`).join(" "),
        formatTime(r.context?.gameCreatedAt ?? BigInt(0)),
        formatTime(r.context?.gameUpdatedAt ?? BigInt(0)),
      ]
        .map(csvField)
        .join(","),
    ),
  ].join("\n") + "\n";

const download = (filename: string, body: string, type: string) => {
  const url = window.URL.createObjectURL(new Blob([body], { type }));
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  window.URL.revokeObjectURL(url);
};

// A user's name with the usual menu: profile, moderation, and the rest.
const Name = (props: { username: string; userId: string }) =>
  props.userId ? (
    <UsernameWithContext
      username={props.username || props.userId}
      userID={props.userId}
      omitSendMessage
      showModTools
      moderate={moderateUser}
    />
  ) : (
    <React.Fragment>{props.username}</React.Fragment>
  );

// What the channel is (a game's players, a private chat's two users, a league
// or club), linked to where it lives, with the raw channel name beneath.
const ChannelCell = (props: { r: ChatChannelSummary }) => {
  const ctx = props.r.context;
  const label = ctx?.label || props.r.channel;
  return (
    <React.Fragment>
      {ctx?.link ? (
        <Link to={ctx.link} target="_blank">
          {label}
        </Link>
      ) : (
        label
      )}
      {props.r.userPlayed && <Tag className="chat-review-played">played</Tag>}
      {props.r.channel.startsWith("chat.pm.") && ctx?.parties.length ? (
        <div>
          {ctx.parties.map((p, i) => (
            <React.Fragment key={p.userId}>
              {i > 0 && " & "}
              <Name username={p.username} userId={p.userId} />
            </React.Fragment>
          ))}
        </div>
      ) : null}
      <div>
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          {props.r.channel}
        </Typography.Text>
      </div>
    </React.Fragment>
  );
};

const ChannelChats = (props: { channel: string }) => {
  const modClient = useClient(ModService);
  const [msgs, setMsgs] = useState<Array<ChatMessage> | null>(null);

  const load = async () => {
    try {
      const resp = await modClient.getChannelChats({ channel: props.channel });
      setMsgs(resp.messages);
    } catch (e) {
      flashError(e);
    }
  };

  if (msgs === null) {
    return <Button onClick={load}>Load messages</Button>;
  }
  const base = props.channel.replace(/[^\w.-]/g, "_");
  return (
    <React.Fragment>
      <Space style={{ marginBottom: 8 }}>
        <Button
          onClick={() => download(`${base}.csv`, chatsToCSV(msgs), "text/csv")}
        >
          Export CSV
        </Button>
        <Button
          onClick={() =>
            download(
              `${base}.json`,
              JSON.stringify(
                msgs.map((m) => ({
                  timeUtc: formatTime(m.timestamp),
                  channel: m.channel,
                  username: m.username,
                  userId: m.userId,
                  id: m.id,
                  message: m.message,
                })),
                null,
                2,
              ),
              "application/json",
            )
          }
        >
          Export JSON
        </Button>
      </Space>
      <Table
        dataSource={msgs}
        rowKey="id"
        size="small"
        pagination={false}
        columns={[
          {
            title: "Time (UTC)",
            key: "time",
            render: (_, m) => formatTime(m.timestamp),
          },
          {
            title: "User",
            key: "username",
            render: (_, m) => <Name username={m.username} userId={m.userId} />,
          },
          { title: "Message", dataIndex: "message", key: "message" },
        ]}
      />
    </React.Fragment>
  );
};

// Review the chats Redis still holds (up to 14 days): find the channels by
// type and sender, see who spoke, read and export a channel's messages.
export const ChatReview = () => {
  const modClient = useClient(ModService);
  const [prefix, setPrefix] = useState("chat.game.");
  const [username, setUsername] = useState("");
  const [rows, setRows] = useState<Array<ChatChannelSummary>>([]);
  const [cursor, setCursor] = useState("");
  const [loading, setLoading] = useState(false);
  const [lookup, setLookup] = useState("");

  const scan = async (fromCursor: string) => {
    setLoading(true);
    try {
      const resp = await modClient.listChatChannels({
        channelPrefix: prefix,
        username,
        cursor: fromCursor,
      });
      setRows((prev) =>
        fromCursor === "" ? resp.channels : mergeRows(prev, resp.channels),
      );
      setCursor(resp.nextCursor);
    } catch (e) {
      flashError(e);
    } finally {
      setLoading(false);
    }
  };

  const showLookup = async () => {
    const id = lookup.trim();
    if (!id) {
      return;
    }
    // A full channel name, or a game id: show both of that game's channels.
    const channels = id.startsWith("chat.")
      ? [id]
      : [`chat.game.${id}`, `chat.gametv.${id}`];
    try {
      const resp = await modClient.listChatChannels({ channels });
      setRows(resp.channels);
      setCursor("0");
    } catch (e) {
      flashError(e);
    }
  };

  const sorted = [...rows].sort((a, b) =>
    Number(b.lastTimestamp - a.lastTimestamp),
  );

  return (
    <div className="chat-review">
      <h3>Chat review</h3>
      <Space wrap style={{ marginBottom: 12 }}>
        <Select
          value={prefix}
          onChange={setPrefix}
          options={channelTypes}
          style={{ width: 200 }}
        />
        <Input
          placeholder="Username (optional)"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          onPressEnter={() => scan("")}
          style={{ width: 200 }}
        />
        <Button type="primary" loading={loading} onClick={() => scan("")}>
          Scan
        </Button>
        {cursor !== "" && cursor !== "0" && (
          <Button loading={loading} onClick={() => scan(cursor)}>
            Scan more
          </Button>
        )}
        <Button
          disabled={rows.length === 0}
          onClick={() =>
            download("chat-channels.csv", summariesToCSV(sorted), "text/csv")
          }
        >
          Export list
        </Button>
      </Space>
      <Space wrap style={{ marginBottom: 12 }}>
        <Input
          placeholder="Game id or channel"
          value={lookup}
          onChange={(e) => setLookup(e.target.value)}
          onPressEnter={showLookup}
          style={{ width: 300 }}
        />
        <Button onClick={showLookup}>Open</Button>
      </Space>
      <p>
        Only chats from the last 14 days, and not deleted ones. A game with no
        chat at all is not listed. A username finds the chat of every game that
        user played (marked "played"), whoever wrote in it, plus the other
        channels they wrote in and their private chats.
        {cursor === "0" && rows.length > 0 && " Scan complete."}
      </p>
      <Table
        dataSource={sorted}
        rowKey="channel"
        size="small"
        pagination={{ pageSize: 50 }}
        expandable={{
          expandedRowRender: (r) => <ChannelChats channel={r.channel} />,
        }}
        columns={[
          {
            title: "Channel",
            key: "channel",
            render: (_, r) => <ChannelCell r={r} />,
          },
          {
            title: "Msgs",
            dataIndex: "messageCount",
            key: "messageCount",
          },
          {
            title: "Who spoke",
            key: "speakers",
            filters: [
              { text: "1 speaker", value: 1 },
              { text: "2 speakers", value: 2 },
              { text: "3+ speakers", value: 3 },
            ],
            onFilter: (value, r) =>
              value === 3 ? r.senders.length >= 3 : r.senders.length === value,
            render: (_, r) =>
              r.senders.map((s, i) => (
                <React.Fragment key={s.userId || s.username}>
                  {i > 0 && ", "}
                  <Name username={s.username} userId={s.userId} /> ({s.count})
                </React.Fragment>
              )),
          },
          {
            title: "First (UTC)",
            key: "first",
            render: (_, r) => formatTime(r.firstTimestamp),
          },
          {
            title: "Last (UTC)",
            key: "last",
            render: (_, r) => formatTime(r.lastTimestamp),
          },
          {
            title: "Game created / last updated (UTC)",
            key: "game",
            render: (_, r) =>
              r.context?.gameCreatedAt ? (
                <React.Fragment>
                  <div>{formatTime(r.context.gameCreatedAt)}</div>
                  <div>{formatTime(r.context.gameUpdatedAt)}</div>
                </React.Fragment>
              ) : null,
          },
        ]}
      />
    </div>
  );
};
