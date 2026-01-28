import React from "react";
import { Card, Table, Tag, Spin, Empty } from "antd";
import { useQuery } from "@connectrpc/connect-query";
import { getSeasonPlayersWithUnstartedGames } from "../gen/api/proto/league_service/league_service-LeagueService_connectquery";
import { Division } from "../gen/api/proto/ipc/league_pb";
import { UsernameWithContext } from "../shared/usernameWithContext";

type ZeroMoveGamesDashboardProps = {
  seasonId: string;
  seasonNumber: number;
  playerToDivisionMap: Map<string, Division>;
  setSelectedDivisionId: React.Dispatch<React.SetStateAction<string>>;
};

export const ZeroMoveGamesDashboard: React.FC<ZeroMoveGamesDashboardProps> = ({
  seasonId,
  seasonNumber,
  playerToDivisionMap,
  setSelectedDivisionId,
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
      render: (_: unknown, player: (typeof dataSource)[0]) => {
        const division = player.division;
        return (
          <UsernameWithContext
            key={player.userId}
            username={player.username}
            userID={player.userId}
            infoText={
              division
                ? division.divisionName || `Division ${division.divisionNumber}`
                : undefined
            }
            handleInfoText={
              division
                ? () => {
                    setSelectedDivisionId(division.uuid);
                  }
                : undefined
            }
          />
        );
      },
      sorter: (a: (typeof dataSource)[0], b: (typeof dataSource)[0]) => {
        const aun = a.username;
        const bun = b.username;
        const aunl = aun.toLowerCase();
        const bunl = bun.toLowerCase();
        if (aunl < bunl) return -1;
        if (aunl > bunl) return 1;
        if (aun < bun) return -1;
        if (aun > bun) return 1;
        const aui = a.userId;
        const bui = b.userId;
        if (aui < bui) return -1;
        if (aui > bui) return 1;
        return 0;
      },
    },
    {
      title: "Division",
      dataIndex: "division",
      key: "division",
      render: (division: Division) =>
        division
          ? division.divisionName || `Division ${division.divisionNumber}`
          : "-",
      sorter: (a: (typeof dataSource)[0], b: (typeof dataSource)[0]) => {
        const adnum = a.division?.divisionNumber ?? -Infinity;
        const bdnum = b.division?.divisionNumber ?? -Infinity;
        if (adnum < bdnum) return -1;
        if (adnum > bdnum) return 1;
        return 0;
      },
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
    data?.players.map((player) => {
      const division = playerToDivisionMap.get(player.userId);
      return {
        key: player.userId,
        userId: player.userId,
        username: player.username,
        division,
        unstartedGameCount: player.unstartedGameCount,
      };
    }) || [];

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
              <div style={{ marginBottom: 16 }} className="league-color-666">
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
