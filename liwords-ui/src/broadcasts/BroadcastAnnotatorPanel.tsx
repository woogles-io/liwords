import React from "react";
import { Card, List, Tag, Button, Space, Popconfirm, App } from "antd";
import { EditOutlined } from "@ant-design/icons";
import { Link } from "react-router";
import { useQuery, useMutation } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import {
  getMyClaimedGames,
  unclaimGame,
} from "../gen/api/proto/broadcast_service/broadcast_service-BroadcastService_connectquery";
import { flashError } from "../utils/hooks/connect";

type Props = {
  slug: string;
};

export const BroadcastAnnotatorPanel: React.FC<Props> = ({ slug }) => {
  const { notification } = App.useApp();
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery(getMyClaimedGames, { slug, limit: 20 });

  const invalidate = () => {
    queryClient.invalidateQueries({
      queryKey: ["connect-query", { methodName: "GetMyClaimedGames" }],
    });
    queryClient.invalidateQueries({
      queryKey: ["connect-query", { methodName: "GetBroadcastGames" }],
    });
  };

  const unclaimMutation = useMutation(unclaimGame, {
    onSuccess: () => {
      notification.success({ message: "Game unclaimed" });
      invalidate();
    },
    onError: (e) => flashError(e),
  });

  const games = data?.games ?? [];

  return (
    <Card
      title="My Annotations"
      size="small"
      style={{ marginTop: 24 }}
      loading={isLoading}
    >
      {games.length === 0 ? (
        <p style={{ color: "#888" }}>No games claimed yet.</p>
      ) : (
        <List
          size="small"
          dataSource={games}
          renderItem={(game) => (
            <List.Item
              actions={[
                <Link key="edit" to={`/editor/${game.gameUuid}`}>
                  <Button size="small" icon={<EditOutlined />}>
                    Edit
                  </Button>
                </Link>,
                <Popconfirm
                  key="unclaim"
                  title="Unclaim this game? The annotation will be deleted."
                  onConfirm={() =>
                    unclaimMutation.mutate({
                      slug,
                      round: game.round,
                      tableNumber: game.tableNumber,
                      division: game.division,
                    })
                  }
                >
                  <Button
                    size="small"
                    danger
                    loading={
                      unclaimMutation.isPending &&
                      unclaimMutation.variables?.round === game.round &&
                      unclaimMutation.variables?.tableNumber ===
                        game.tableNumber
                    }
                  >
                    Unclaim
                  </Button>
                </Popconfirm>,
              ]}
            >
              <Space>
                <span>
                  {game.division && <strong>Div {game.division} / </strong>}
                  <strong>
                    R{game.round} / T{game.tableNumber}
                  </strong>
                  {" — "}
                  {game.player1Name} vs {game.player2Name}
                </span>
                {game.annotationDone ? (
                  <Tag color="green">Done</Tag>
                ) : (
                  <Tag color="blue">In progress</Tag>
                )}
              </Space>
            </List.Item>
          )}
        />
      )}
    </Card>
  );
};
