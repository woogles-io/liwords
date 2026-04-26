// Control the editor

import {
  Button,
  Collapse,
  Form,
  Input,
  Popconfirm,
  Select,
  Typography,
  Card,
  Space,
  message,
} from "antd";
import { BookOutlined, CloseOutlined } from "@ant-design/icons";
import { Store } from "antd/lib/form/interface";
import { useEffect, useState, useCallback } from "react";
import { ChallengeRule } from "../gen/api/proto/ipc/omgwords_pb";
import { LexiconFormItem, historicalLexica } from "../shared/lexicon_display";
import {
  useGameContextStoreContext,
  useLoginStateStoreContext,
} from "../store/store";
import { baseURL, useClient, flashError } from "../utils/hooks/connect";
import { AddToCollectionModal } from "../collections/AddToCollectionModal";
import {
  CollectionsService,
  Collection,
} from "../gen/api/proto/collections_service/collections_service_pb";
import { Link, useNavigate } from "react-router";
import { useQuery, useMutation } from "@connectrpc/connect-query";
import {
  getBroadcastGameContext,
  unclaimGame,
} from "../gen/api/proto/broadcast_service/broadcast_service-BroadcastService_connectquery";
import { OBSPanel } from "../broadcasts/OBSPanel";

type Props = {
  gameID?: string;
  createNewGame: (
    p1name: string,
    p2name: string,
    lex: string,
    chrule: ChallengeRule,
  ) => void;
  deleteGame: (gid: string) => void;
  editGame: (p1name: string, p2name: string, description: string) => void;
};

export const EditorControl = (props: Props) => {
  const navigate = useNavigate();
  const { loginState } = useLoginStateStoreContext();

  const gameURL = props.gameID ? `${baseURL}/anno/${props.gameID}` : "";

  const { data: broadcastCtx } = useQuery(
    getBroadcastGameContext,
    { gameUuid: props.gameID ?? "" },
    { enabled: !!props.gameID },
  );

  const unclaimMutation = useMutation(unclaimGame, {
    onSuccess: () => {
      if (broadcastCtx) {
        navigate(`/broadcasts/${broadcastCtx.broadcastSlug}`);
      }
    },
    onError: (e) => flashError(e),
  });

  const [confirmDelVisible, setConfirmDelVisible] = useState(false);
  const [collectionModalVisible, setCollectionModalVisible] = useState(false);
  const [gameCollections, setGameCollections] = useState<Collection[]>([]);
  const [removingCollectionId, setRemovingCollectionId] = useState<
    string | null
  >(null);

  const collectionsClient = useClient(CollectionsService);

  const fetchGameCollections = useCallback(async () => {
    if (!props.gameID) return;

    try {
      const response = await collectionsClient.getCollectionsForGame({
        gameId: props.gameID,
      });
      setGameCollections(response.collections || []);
    } catch (err) {
      console.error("Failed to fetch collections for game:", err);
    }
  }, [props.gameID, collectionsClient]);

  const removeGameFromCollection = useCallback(
    async (collectionUuid: string, collectionTitle: string) => {
      if (!props.gameID) return;

      setRemovingCollectionId(collectionUuid);
      try {
        await collectionsClient.removeGameFromCollection({
          collectionUuid,
          gameId: props.gameID,
        });
        await fetchGameCollections();
        message.success(`Game removed from "${collectionTitle}"`);
      } catch (err) {
        console.error("Failed to remove game from collection:", err);
        message.error(
          "Failed to remove game from collection. Please try again.",
        );
      } finally {
        setRemovingCollectionId(null);
      }
    },
    [props.gameID, collectionsClient, fetchGameCollections],
  );

  useEffect(() => {
    if (props.gameID) {
      fetchGameCollections();
    }
  }, [props.gameID, fetchGameCollections]);

  if (!props.gameID) {
    return (
      <div className="editor-control">
        <CreationForm createNewGame={props.createNewGame} />
      </div>
    );
  }

  const collectionsExtra = (
    <Button
      size="small"
      icon={<BookOutlined />}
      onClick={(e) => {
        e.stopPropagation();
        setCollectionModalVisible(true);
      }}
    >
      Add
    </Button>
  );

  const collapseItems = [
    {
      key: "details",
      label: "Game details",
      children: <EditForm editGame={props.editGame} />,
    },
    {
      key: "share",
      label: "Share & broadcast",
      children: (
        <Space direction="vertical" style={{ width: "100%" }}>
          <div>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              Link to game
            </Typography.Text>
            <Typography.Paragraph
              copyable
              className="readable-text-color"
              style={{ marginBottom: 0 }}
            >
              {gameURL}
            </Typography.Paragraph>
          </div>
          <OBSPanel
            gameID={props.gameID}
            broadcastSlug={broadcastCtx?.broadcastSlug}
            slotName={broadcastCtx?.slotName}
            username={loginState.loggedIn ? loginState.username : undefined}
            defaultMode={
              broadcastCtx?.broadcastSlug && broadcastCtx?.slotName
                ? "slot"
                : "game"
            }
          />
        </Space>
      ),
    },
    {
      key: "collections",
      label: "Collections",
      extra: collectionsExtra,
      children:
        gameCollections.length === 0 ? (
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            This game is not in any collection yet.
          </Typography.Text>
        ) : (
          <Space direction="vertical" style={{ width: "100%" }}>
            {gameCollections.map((collection) => (
              <Card
                key={collection.uuid}
                size="small"
                style={{ background: "rgba(255,255,255,0.04)" }}
              >
                <Space
                  style={{ width: "100%", justifyContent: "space-between" }}
                >
                  <Link
                    to={`/collections/${collection.uuid}`}
                    style={{ fontSize: 13 }}
                  >
                    {collection.title}
                    {collection.games?.[0]?.chapterTitle &&
                      ` (Ch. ${collection.games[0].chapterNumber})`}
                  </Link>
                  <Popconfirm
                    title={`Remove game from "${collection.title}"?`}
                    description="This will remove the game from this collection."
                    onConfirm={() =>
                      removeGameFromCollection(
                        collection.uuid,
                        collection.title,
                      )
                    }
                    okText="Remove"
                    cancelText="Cancel"
                    okButtonProps={{ danger: true }}
                  >
                    <Button
                      type="text"
                      size="small"
                      icon={<CloseOutlined />}
                      loading={removingCollectionId === collection.uuid}
                      onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                      }}
                    />
                  </Popconfirm>
                </Space>
              </Card>
            ))}
          </Space>
        ),
    },
  ];

  return (
    <div className="editor-control">
      <Collapse
        accordion
        defaultActiveKey={["details"]}
        items={collapseItems}
      />
      <div style={{ textAlign: "right", marginTop: 12 }}>
        {broadcastCtx ? (
          <Popconfirm
            title="Unclaim this game? The annotation will be deleted."
            onConfirm={() =>
              unclaimMutation.mutate({
                slug: broadcastCtx.broadcastSlug,
                round: broadcastCtx.round,
                tableNumber: broadcastCtx.tableNumber,
                division: broadcastCtx.division,
              })
            }
            okText="Unclaim"
            cancelText="Cancel"
            okButtonProps={{ danger: true }}
          >
            <Button type="primary" danger loading={unclaimMutation.isPending}>
              Unclaim this game
            </Button>
          </Popconfirm>
        ) : (
          <Popconfirm
            title="Are you sure you wish to delete this game? This action can not be undone!"
            onConfirm={() => {
              props.deleteGame(props.gameID!);
              setConfirmDelVisible(false);
            }}
            onCancel={() => setConfirmDelVisible(false)}
            okText="Yes"
            cancelText="No"
            open={confirmDelVisible}
          >
            <Button
              onClick={() => setConfirmDelVisible(true)}
              type="primary"
              danger
            >
              Delete this game
            </Button>
          </Popconfirm>
        )}
      </div>
      <AddToCollectionModal
        visible={collectionModalVisible}
        gameId={props.gameID}
        isAnnotated={true}
        onClose={() => setCollectionModalVisible(false)}
        onSuccess={(collectionUuid) => {
          fetchGameCollections();
          console.log("Game added to collection:", collectionUuid);
        }}
      />
    </div>
  );
};

type CreationFormProps = {
  createNewGame: (
    p1name: string,
    p2name: string,
    lex: string,
    chrule: ChallengeRule,
  ) => void;
};
const CreationForm = (props: CreationFormProps) => {
  return (
    <Form
      layout="vertical"
      onFinish={(vals: Store) =>
        props.createNewGame(
          vals.p1name,
          vals.p2name,
          vals.lexicon,
          vals.challengerule,
        )
      }
    >
      <Form.Item
        label="Player 1 name"
        name="p1name"
        rules={[
          {
            required: true,
            message: "Player name is required",
          },
        ]}
      >
        <Input maxLength={50} />
      </Form.Item>
      <Form.Item
        label="Player 2 name"
        name="p2name"
        rules={[
          {
            required: true,
            message: "Player name is required",
          },
        ]}
      >
        <Input maxLength={50} />
      </Form.Item>
      {/* exclude ECWL since that lexicon can't be played without VOID for now */}
      <LexiconFormItem
        excludedLexica={new Set(["ECWL"])}
        additionalLexica={historicalLexica}
      />
      <Form.Item
        label="Challenge rule"
        name="challengerule"
        rules={[
          {
            required: true,
            message: "Challenge rule is required",
          },
        ]}
      >
        <Select>
          <Select.Option value={ChallengeRule.ChallengeRule_FIVE_POINT}>
            5 points
          </Select.Option>
          <Select.Option value={ChallengeRule.ChallengeRule_DOUBLE}>
            Double
          </Select.Option>
          <Select.Option value={ChallengeRule.ChallengeRule_TEN_POINT}>
            10 points
          </Select.Option>
          <Select.Option value={ChallengeRule.ChallengeRule_SINGLE}>
            Single
          </Select.Option>
        </Select>
      </Form.Item>

      <Form.Item>
        <Button type="default" htmlType="submit">
          Create new game
        </Button>
      </Form.Item>
    </Form>
  );
};

type EditFormProps = {
  editGame: (p1name: string, p2name: string, description: string) => void;
};

const EditForm = (props: EditFormProps) => {
  const { gameContext } = useGameContextStoreContext();
  const [formref] = Form.useForm();

  useEffect(() => {
    formref.resetFields();
  }, [gameContext.gameDocument, formref]);

  return (
    <Form
      layout="vertical"
      form={formref}
      initialValues={{
        p1name: gameContext.gameDocument.players[0].realName,
        p2name: gameContext.gameDocument.players[1].realName,
        description: gameContext.gameDocument.description,
      }}
      onFinish={(vals: Store) =>
        props.editGame(vals.p1name, vals.p2name, vals.description)
      }
    >
      <Form.Item label="Player 1 name" name="p1name">
        <Input maxLength={50} required />
      </Form.Item>
      <Form.Item label="Player 2 name" name="p2name">
        <Input maxLength={50} required />
      </Form.Item>
      <Form.Item label="Game description" name="description">
        <Input.TextArea maxLength={140} rows={2} />
      </Form.Item>
      <Form.Item>
        <Button type="primary" htmlType="submit">
          Save settings
        </Button>
      </Form.Item>
    </Form>
  );
};
