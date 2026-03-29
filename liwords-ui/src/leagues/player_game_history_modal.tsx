import React from "react";
import { Modal, Spin, Table, Tag, Tooltip } from "antd";
import { useQuery } from "@connectrpc/connect-query";
import { getPlayerSeasonGames } from "../gen/api/proto/league_service/league_service-LeagueService_connectquery";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { GameEndReason } from "../gen/api/proto/ipc/omgwords_pb";
import { UsernameWithContext } from "../shared/usernameWithContext";

export const endReasonLabel = (reason: GameEndReason): string => {
  switch (reason) {
    case GameEndReason.TIME:
      return "time";
    case GameEndReason.CONSECUTIVE_ZEROES:
      return "zeroes";
    case GameEndReason.RESIGNED:
      return "resigned";
    case GameEndReason.TRIPLE_CHALLENGE:
      return "triple";
    case GameEndReason.FORCE_FORFEIT:
      return "forfeit";
    case GameEndReason.ADJUDICATED:
      return "adjudicated";
    case GameEndReason.ABORTED:
      return "aborted";
    case GameEndReason.CANCELLED:
      return "cancelled";
    case GameEndReason.STANDARD:
      return "";
    case GameEndReason.NONE:
    default:
      return "other";
  }
};

type PlayerGameHistoryModalProps = {
  visible: boolean;
  onClose: () => void;
  userId: string;
  username: string;
  seasonId: string;
  seasonNumber: number;
};

export const PlayerGameHistoryModal: React.FC<PlayerGameHistoryModalProps> = ({
  visible,
  onClose,
  userId,
  username,
  seasonId,
  seasonNumber,
}) => {
  const { data, isLoading, error } = useQuery(getPlayerSeasonGames, {
    userId,
    seasonId,
  });

  const columns = [
    {
      title: "Opponent",
      dataIndex: "opponentUsername",
      key: "opponent",
      fixed: "left" as const,
      render: (username: string) => <strong>{username}</strong>,
    },
    {
      title: "Result",
      key: "result",
      render: (
        _: unknown,
        record: { result: string; gameEndReason: GameEndReason },
      ) => {
        const reason = endReasonLabel(record.gameEndReason);
        let tag: React.ReactNode = null;
        if (record.result === "win") {
          tag = <Tag color="green">Win</Tag>;
        } else if (record.result === "loss") {
          tag = <Tag color="red">Loss</Tag>;
        } else if (record.result === "draw") {
          tag = <Tag color="blue">Draw</Tag>;
        } else if (record.result === "turn") {
          return <Tag color="gold">On Turn</Tag>;
        } else if (record.result === "in_progress") {
          return <Tag color="orange">In Progress</Tag>;
        }
        if (!tag) return null;
        return reason ? (
          <span style={{ whiteSpace: "nowrap" }}>
            {tag} ({reason})
          </span>
        ) : (
          tag
        );
      },
    },
    {
      title: "Score",
      key: "score",
      render: (
        _: unknown,
        record: { playerScore: number; opponentScore: number; result: string },
      ) => {
        const spread = record.playerScore - record.opponentScore;
        return (
          <Tooltip
            placement="left"
            title={`${spread >= 0 ? "+" : ""}${spread}`}
          >
            {record.playerScore}-{record.opponentScore}
          </Tooltip>
        );
      },
    },
    {
      title: "Date",
      dataIndex: "gameDate",
      key: "date",
      render: (date?: Date) => {
        if (!date) return "—";
        return (
          <Tooltip title={date.toLocaleString()}>
            {date.toLocaleDateString()}
          </Tooltip>
        );
      },
    },
  ];

  const dataSource =
    data?.games.map((game) => ({
      key: game.gameId,
      gameId: game.gameId,
      opponentUsername: game.opponentUsername,
      result: game.result,
      playerScore: game.playerScore,
      opponentScore: game.opponentScore,
      gameDate: game.gameDate ? timestampDate(game.gameDate) : undefined,
      gameEndReason: game.gameEndReason,
    })) || [];

  const handleRowClick = (record: { gameId: string }) => {
    window.open(`/game/${record.gameId}`, "_blank");
  };

  return (
    <Modal
      className="league-game-modal"
      title={
        <React.Fragment>
          <UsernameWithContext username={username} userID={userId} />
          's Season {seasonNumber} Games
        </React.Fragment>
      }
      open={visible}
      onCancel={onClose}
      footer={null}
      width={700}
      zIndex={2000}
    >
      {isLoading && (
        <div style={{ textAlign: "center", padding: "40px" }}>
          <Spin size="large" />
        </div>
      )}
      {error && (
        <div style={{ color: "red", padding: "20px", textAlign: "center" }}>
          Failed to load game history: {error.message}
        </div>
      )}
      {!isLoading && !error && (
        <>
          {dataSource.length === 0 ? (
            <div
              style={{ textAlign: "center", padding: "40px" }}
              className="league-color-999"
            >
              No games found for this season.
            </div>
          ) : (
            <Table
              columns={columns}
              dataSource={dataSource}
              pagination={false}
              size="small"
              scroll={{ x: "max-content" }}
              onRow={(record) => ({
                onClick: () => handleRowClick(record),
                style: { cursor: "pointer" },
              })}
            />
          )}
        </>
      )}
    </Modal>
  );
};
