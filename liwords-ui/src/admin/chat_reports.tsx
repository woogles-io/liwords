import React, { useCallback, useEffect, useRef, useState } from "react";
import { Button, Radio, Table, Tag, Typography } from "antd";
import type { ColumnsType } from "antd/es/table";
import { Timestamp, timestampMs } from "@bufbuild/protobuf/wkt";
import {
  ChatReport,
  ModService,
} from "../gen/api/proto/mod_service/mod_service_pb";
import { flashError, useClient } from "../utils/hooks/connect";
import { ChannelChats, formatTime, Name } from "./chat_review";

const formatStamp = (ts?: Timestamp) =>
  ts ? formatTime(BigInt(timestampMs(ts))) : "";

// The chat messages people reported: the open reports, newest first, until a
// moderator marks them handled, or the handled ones, the most recently handled
// first, with who handled each, to reopen one handled by mistake. Each comes
// with who sent the message and who reported it, and the channel around it to
// open.
export const ChatReports = () => {
  const modClient = useClient(ModService);
  const [handled, setHandled] = useState(false);
  const [reports, setReports] = useState<Array<ChatReport> | null>(null);
  // the latest list asked for: an older answer arriving after it is dropped
  const asked = useRef(0);

  const load = useCallback(async () => {
    const n = ++asked.current;
    try {
      const resp = await modClient.getChatReports({ handled });
      if (n === asked.current) setReports(resp.reports);
    } catch (e) {
      flashError(e);
    }
  }, [modClient, handled]);
  useEffect(() => {
    load();
  }, [load]);

  const act = async (f: () => Promise<unknown>) => {
    try {
      await f();
      await load();
    } catch (e) {
      flashError(e);
    }
  };

  const columns: ColumnsType<ChatReport> = [
    // first, so it is in view however far the table scrolls
    {
      title: "",
      key: "act",
      render: (_, r) =>
        handled ? (
          <Button
            onClick={() =>
              act(() => modClient.reopenChatReport({ reportId: r.id }))
            }
          >
            Reopen
          </Button>
        ) : (
          <Button
            onClick={() =>
              act(() => modClient.handleChatReport({ reportId: r.id }))
            }
          >
            Mark handled
          </Button>
        ),
    },
    // what was said and by whom first, then the report's own details
    {
      title: "Message",
      key: "message",
      render: (_, r) => (
        <React.Fragment>
          {r.message?.message}{" "}
          {r.deleted && <Tag className="chat-reports-deleted">deleted</Tag>}
        </React.Fragment>
      ),
    },
    {
      title: "Sender",
      key: "sender",
      render: (_, r) => (
        <Name
          username={r.message?.username ?? ""}
          userId={r.message?.userId ?? ""}
        />
      ),
    },
    {
      title: "By",
      key: "by",
      render: (_, r) => (
        <Name username={r.reporterUsername} userId={r.reporterId} />
      ),
    },
    {
      title: "Reported",
      key: "reported",
      className: "chat-review-nowrap",
      render: (_, r) => formatStamp(r.reportedAt),
    },
    {
      title: "Channel",
      key: "channel",
      render: (_, r) => r.message?.channel,
    },
    ...(handled
      ? ([
          {
            title: "Handled by",
            key: "handledBy",
            render: (_, r) => (
              <Name username={r.handledByUsername} userId={r.handledById} />
            ),
          },
          {
            title: "Handled",
            key: "handledAt",
            className: "chat-review-nowrap",
            render: (_, r) => formatStamp(r.handledAt),
          },
        ] satisfies ColumnsType<ChatReport>)
      : []),
  ];

  return (
    <div className="chat-review chat-reports">
      <Typography.Paragraph>
        Chat messages people reported.
      </Typography.Paragraph>
      <div className="chat-reports-bar">
        <Radio.Group
          optionType="button"
          value={handled ? "handled" : "open"}
          onChange={(e) => {
            setReports(null);
            setHandled(e.target.value === "handled");
          }}
          options={[
            { label: "Open", value: "open" },
            { label: "Handled", value: "handled" },
          ]}
        />
        <Button onClick={load}>Refresh</Button>
      </div>
      <Table
        // paged as the other admin lists are (antd's own: ten a page, more
        // to choose from on a long list); each list starts at its first page
        key={handled ? "handled" : "open"}
        rowKey={(r) => String(r.id)}
        loading={reports === null}
        dataSource={reports ?? []}
        // wider than the page, the table scrolls within itself
        scroll={{ x: true }}
        expandable={{
          expandedRowRender: (r) =>
            r.message ? (
              <ChannelChats channel={r.message.channel} around={r.message.id} />
            ) : null,
        }}
        columns={columns}
      />
    </div>
  );
};
