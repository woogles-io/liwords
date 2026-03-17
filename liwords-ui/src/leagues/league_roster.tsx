import React, { useState, useMemo } from "react";
import { Input, Table, Tag, Tooltip, theme } from "antd";
import type { SortOrder } from "antd/es/table/interface";
import {
  StarOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
  MinusOutlined,
} from "@ant-design/icons";
import { useQuery } from "@connectrpc/connect-query";
import { getLeagueRoster } from "../gen/api/proto/league_service/league_service-LeagueService_connectquery";
import {
  LeagueRosterPlayer,
  LeagueRosterSeason,
} from "../gen/api/proto/league_service/league_service_pb";
import { StandingResult } from "../gen/api/proto/ipc/league_pb";
import { UsernameWithContext } from "../shared/usernameWithContext";

type Props = {
  leagueId: string;
  onJumpToSeason: (seasonNumber: number, divisionNumber: number) => void;
};

const resultIcon = (result: StandingResult) => {
  switch (result) {
    case StandingResult.RESULT_CHAMPION:
      return (
        <Tooltip title="Champion">
          <StarOutlined style={{ color: "#d4af37" }} />
        </Tooltip>
      );
    case StandingResult.RESULT_PROMOTED:
      return (
        <Tooltip title="Promoted">
          <ArrowUpOutlined style={{ color: "#52c41a" }} />
        </Tooltip>
      );
    case StandingResult.RESULT_RELEGATED:
      return (
        <Tooltip title="Relegated">
          <ArrowDownOutlined style={{ color: "#ff4d4f" }} />
        </Tooltip>
      );
    case StandingResult.RESULT_STAYED:
      return (
        <Tooltip title="Stayed">
          <MinusOutlined style={{ color: "#8c8c8c" }} />
        </Tooltip>
      );
    default:
      return null;
  }
};

const formatSeason = (season: LeagueRosterSeason | undefined) => {
  if (!season) return <span className="roster-empty">—</span>;
  if (season.divisionNumber === 0) {
    return <Tag color="lime">Registered</Tag>;
  }
  const record = `${season.wins}-${season.losses}${season.draws ? `-${season.draws}` : ""}`;
  const spread = season.spread > 0 ? `+${season.spread}` : `${season.spread}`;
  return (
    <Tooltip title={`${record} (${spread})`}>
      <span className="roster-season" style={{ cursor: "pointer" }}>
        <Tag
          color={
            season.divisionNumber === 1
              ? "gold"
              : season.divisionNumber === 2
                ? "blue"
                : "green"
          }
        >
          D{season.divisionNumber}
        </Tag>
        {season.rank > 0 && <span className="roster-rank">#{season.rank}</span>}
        {resultIcon(season.result)}
      </span>
    </Tooltip>
  );
};

// Sort key for a player in a given season: division ASC, rank ASC.
// Players not in that season sort last.
const seasonSortKey = (
  player: LeagueRosterPlayer,
  seasonNumber: number,
): number => {
  const s = player.seasons.find((x) => x.seasonNumber === seasonNumber);
  if (!s) return 999999;
  if (s.divisionNumber === 0) return 999998; // registered but unplaced
  return s.divisionNumber * 1000 + (s.rank || 999);
};

export const LeagueRoster: React.FC<Props> = ({ leagueId, onJumpToSeason }) => {
  const [search, setSearch] = useState("");
  const { token } = theme.useToken();

  const { data, isLoading } = useQuery(getLeagueRoster, {
    leagueId,
  });

  const filteredPlayers = useMemo(() => {
    if (!data?.players) return [];
    if (!search) return data.players;
    const q = search.toLowerCase();
    return data.players.filter((p: LeagueRosterPlayer) =>
      p.username.toLowerCase().includes(q),
    );
  }, [data?.players, search]);

  const seasonNumbers = useMemo(
    () => [...(data?.seasonNumbers ?? [])].reverse(),
    [data?.seasonNumbers],
  );

  const columns = [
    {
      title: "Player",
      key: "username",
      fixed: "left" as const,
      width: 140,
      onCell: () => ({
        style: { background: token.colorBgContainer },
      }),
      sorter: (a: LeagueRosterPlayer, b: LeagueRosterPlayer) =>
        a.username.localeCompare(b.username),
      sortDirections: ["ascend", "descend"] as SortOrder[],
      render: (_: unknown, record: LeagueRosterPlayer) => (
        <UsernameWithContext
          username={record.username}
          userID={record.userId}
          omitSendMessage
          omitFriend
          omitBlock
        />
      ),
    },
    {
      title: <Tooltip title="Number of seasons played">#S</Tooltip>,
      key: "count",
      width: 40,
      render: (_: unknown, record: LeagueRosterPlayer) => record.seasons.length,
      sorter: (a: LeagueRosterPlayer, b: LeagueRosterPlayer) =>
        a.seasons.length - b.seasons.length,
      sortDirections: ["descend", "ascend"] as SortOrder[],
    },
    ...seasonNumbers.map((sn: number) => ({
      title: `S${sn}`,
      key: `s${sn}`,
      width: 120,
      render: (_: unknown, record: LeagueRosterPlayer) => {
        const season = record.seasons.find(
          (s: LeagueRosterSeason) => s.seasonNumber === sn,
        );
        if (!season) return <span className="roster-empty">—</span>;
        return (
          <span onClick={() => onJumpToSeason(sn, season.divisionNumber)}>
            {formatSeason(season)}
          </span>
        );
      },
      sorter: (a: LeagueRosterPlayer, b: LeagueRosterPlayer) =>
        seasonSortKey(a, sn) - seasonSortKey(b, sn),
      sortDirections: ["ascend", "descend"] as SortOrder[],
    })),
  ];

  return (
    <div className="league-roster">
      <Input
        placeholder="Search player..."
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        style={{ marginBottom: 12, maxWidth: 300 }}
        allowClear
      />
      <Table
        columns={columns}
        dataSource={filteredPlayers}
        rowKey="userId"
        loading={isLoading}
        pagination={false}
        size="small"
        scroll={{ x: "max-content" }}
        showSorterTooltip={false}
      />
    </div>
  );
};
