import React from "react";
import { Card, Table, Tag, Spin, Empty } from "antd";
import { useQuery } from "@connectrpc/connect-query";
import { getSeasonPlayersWithUnstartedGames } from "../gen/api/proto/league_service/league_service-LeagueService_connectquery";

type ZeroMoveGamesDashboardProps = {
  seasonId: string;
  seasonNumber: number;
};

export const ZeroMoveGamesDashboard: React.FC<ZeroMoveGamesDashboardProps> = ({
  seasonId,
  seasonNumber,
}) => {
  const { data, isLoading, error } = useQuery(
    getSeasonPlayersWithUnstartedGames,
    {
      seasonId,
    },
  );

  const columns = [
    {
      title: "Player",
      dataIndex: "username",
      key: "username",
      render: (username: string) => <strong>{username}</strong>,
    },
    {
      title: "Unstarted Games",
      dataIndex: "unstartedGameCount",
      key: "unstartedGameCount",
      render: (count: number) => (
        <Tag color={count > 3 ? "red" : count > 1 ? "orange" : "blue"}>
          {count} game{count !== 1 ? "s" : ""}
        </Tag>
      ),
      sorter: (
        a: { unstartedGameCount: number },
        b: { unstartedGameCount: number },
      ) => b.unstartedGameCount - a.unstartedGameCount,
      defaultSortOrder: "ascend" as const,
    },
  ];

  const dataSource =
    data?.players.map((player) => ({
      key: player.userId,
      userId: player.userId,
      username: player.username,
      unstartedGameCount: player.unstartedGameCount,
    })) || [];

  return (
    <Card
      title={`Players Needing to Start Games - Season ${seasonNumber}`}
      style={{ marginBottom: 24 }}
    >
      {isLoading && (
        <div style={{ textAlign: "center", padding: "40px" }}>
          <Spin size="large" />
        </div>
      )}
      {error && (
        <div style={{ color: "red", padding: "20px", textAlign: "center" }}>
          Failed to load player data: {error.message}
        </div>
      )}
      {!isLoading && !error && (
        <>
          {dataSource.length === 0 ? (
            <Empty
              description="No unstarted games found. All players have made their first move!"
              image={Empty.PRESENTED_IMAGE_SIMPLE}
            />
          ) : (
            <>
              <div style={{ marginBottom: 16, color: "#666" }}>
                <p>
                  These players are on turn but haven't made their first move
                  yet. They may need a reminder to start their games.
                </p>
              </div>
              <Table
                columns={columns}
                dataSource={dataSource}
                pagination={false}
                size="small"
              />
            </>
          )}
        </>
      )}
    </Card>
  );
};
