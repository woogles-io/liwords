import React from "react";
import { Card, Tag, Typography, Spin, Button, Space, Row, Col } from "antd";
import { PlusOutlined, CalendarOutlined } from "@ant-design/icons";
import { Link, useNavigate } from "react-router";
import { useQuery } from "@connectrpc/connect-query";
import { getActiveBroadcasts } from "../gen/api/proto/broadcast_service/broadcast_service-BroadcastService_connectquery";
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
              {b.active && <Tag color="green">LIVE</Tag>}
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
  const { data, isLoading } = useQuery(getActiveBroadcasts, {});
  const { loginState } = useLoginStateStoreContext();
  const navigate = useNavigate();
  const broadcasts = data?.broadcasts ?? [];
  const canManage = loginState.perms.includes("adm");

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
              Live Broadcasts
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
        ) : broadcasts.length === 0 ? (
          <Text type="secondary">No active broadcasts at this time.</Text>
        ) : (
          broadcasts.map((b) => <BroadcastCard key={b.slug} b={b} />)
        )}
      </div>
    </div>
  );
};
