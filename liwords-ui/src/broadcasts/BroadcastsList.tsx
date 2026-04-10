import React from "react";
import { Card, Tag, Typography, Spin, Button, Space, Row, Col } from "antd";
import { PlusOutlined, CalendarOutlined } from "@ant-design/icons";
import { Link, useNavigate } from "react-router";
import { useQuery } from "@connectrpc/connect-query";
import { getAllBroadcasts } from "../gen/api/proto/broadcast_service/broadcast_service-BroadcastService_connectquery";
import type { Broadcast } from "../gen/api/proto/broadcast_service/broadcast_service_pb";
import { TopBar } from "../navigation/topbar";
import { useLoginStateStoreContext } from "../store/store";
import type { Timestamp } from "@bufbuild/protobuf/wkt";

const { Title, Text } = Typography;

function formatTime(ts: Timestamp | undefined): string | null {
  if (!ts) return null;
  const d = new Date(Number(ts.seconds) * 1000);
  return d.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

const BroadcastCard: React.FC<{ b: Broadcast }> = ({ b }) => {
  const start = formatTime(b.pollStartTime);
  const end = formatTime(b.pollEndTime);
  const startDate =
    b.pollStartTime?.seconds != null
      ? new Date(Number(b.pollStartTime.seconds) * 1000)
      : null;
  const isUpcoming = startDate != null && startDate > new Date();

  return (
    <Link to={`/broadcasts/${b.slug}`} style={{ display: "block" }}>
      <Card
        hoverable
        size="small"
        style={{ marginBottom: 16 }}
        styles={{ body: { padding: "16px 20px" } }}
      >
        <Space
          style={{ width: "100%", justifyContent: "space-between" }}
          align="start"
        >
          <div>
            <Space align="center" style={{ marginBottom: 4 }}>
              <Text strong style={{ fontSize: 16 }}>
                {b.name}
              </Text>
              {b.active && !isUpcoming && <Tag color="green">LIVE</Tag>}
              {b.active && isUpcoming && <Tag color="blue">UPCOMING</Tag>}
              {!b.active && <Tag color="default">Archived</Tag>}
            </Space>
            {b.description && (
              <div>
                <Text type="secondary">{b.description}</Text>
              </div>
            )}
            {(start || end) && (
              <div style={{ marginTop: 6 }}>
                <Space size={4}>
                  <CalendarOutlined style={{ color: "#888", fontSize: 12 }} />
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {start && end
                      ? `${start} – ${end}`
                      : start
                        ? `Starts ${start}`
                        : `Ends ${end}`}
                  </Text>
                </Space>
              </div>
            )}
          </div>
          <Tag style={{ flexShrink: 0 }}>{b.lexicon}</Tag>
        </Space>
      </Card>
    </Link>
  );
};

export const BroadcastsList: React.FC = () => {
  const { data, isLoading } = useQuery(getAllBroadcasts, {});
  const { loginState } = useLoginStateStoreContext();
  const navigate = useNavigate();
  const now = new Date();
  const all = data?.broadcasts ?? [];

  // Live: active + start time is in the past (or no start time)
  const live = all
    .filter((b) => {
      if (!b.active) return false;
      const s = b.pollStartTime?.seconds;
      return s == null || new Date(Number(s) * 1000) <= now;
    })
    .sort((a, b) => {
      // Sort live broadcasts by start time descending (most recently started first)
      const aS = Number(a.pollStartTime?.seconds ?? 0);
      const bS = Number(b.pollStartTime?.seconds ?? 0);
      return bS - aS;
    });

  // Upcoming: active + start time is in the future, sorted soonest first
  const upcoming = all
    .filter((b) => {
      if (!b.active) return false;
      const s = b.pollStartTime?.seconds;
      return s != null && new Date(Number(s) * 1000) > now;
    })
    .sort((a, b) => {
      const aS = Number(a.pollStartTime?.seconds ?? 0);
      const bS = Number(b.pollStartTime?.seconds ?? 0);
      return aS - bS;
    });

  // Past: inactive, show last 5 most recent
  const past = all.filter((b) => !b.active).slice(0, 5);

  const canManage = loginState.perms.includes("adm");

  const renderSection = (broadcasts: Broadcast[]) =>
    broadcasts.map((b) => <BroadcastCard key={b.slug} b={b} />);

  return (
    <div>
      <TopBar />
      <div style={{ maxWidth: 720, margin: "32px auto", padding: "0 16px" }}>
        <Row
          align="middle"
          justify="space-between"
          style={{ marginBottom: 20 }}
        >
          <Col>
            <Title level={2} style={{ marginBottom: 0 }}>
              Broadcasts
            </Title>
          </Col>
          {canManage && (
            <Col>
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => navigate("/broadcasts/new")}
              >
                New Broadcast
              </Button>
            </Col>
          )}
        </Row>
        {isLoading ? (
          <div style={{ textAlign: "center", marginTop: 40 }}>
            <Spin size="large" />
          </div>
        ) : live.length === 0 && upcoming.length === 0 && past.length === 0 ? (
          <Text type="secondary">No broadcasts at this time.</Text>
        ) : (
          <>
            {live.length > 0 && (
              <>
                <Title level={4} style={{ marginBottom: 12 }}>
                  Live
                </Title>
                {renderSection(live)}
              </>
            )}
            {upcoming.length > 0 && (
              <>
                <Title
                  level={4}
                  style={{
                    marginTop: live.length > 0 ? 24 : 0,
                    marginBottom: 12,
                  }}
                >
                  Upcoming
                </Title>
                {renderSection(upcoming)}
              </>
            )}
            {past.length > 0 && (
              <>
                <Title
                  level={4}
                  style={{
                    marginTop: live.length > 0 || upcoming.length > 0 ? 24 : 0,
                    marginBottom: 12,
                  }}
                >
                  Past Broadcasts
                </Title>
                {renderSection(past)}
              </>
            )}
          </>
        )}
      </div>
    </div>
  );
};
