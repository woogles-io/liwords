// Control the editor

import {
  Button,
  Form,
  Input,
  Popconfirm,
  Select,
  Typography,
  Card,
  Space,
  Tag,
  message,
} from "antd";
import { BookOutlined, CloseOutlined, FolderOutlined } from "@ant-design/icons";
import { Store } from "antd/lib/form/interface";
import { useEffect, useState, useCallback } from "react";
import { ChallengeRule } from "../gen/api/proto/ipc/omgwords_pb";
import { LexiconFormItem } from "../shared/lexicon_display";
import { useGameContextStoreContext } from "../store/store";
import { baseURL, useClient } from "../utils/hooks/connect";
import { AddToCollectionModal } from "../collections/AddToCollectionModal";
import {
  CollectionsService,
  Collection,
} from "../gen/api/proto/collections_service/collections_service_pb";
import { Link } from "react-router";

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
  let form;

  if (!props.gameID) {
    form = <CreationForm createNewGame={props.createNewGame} />;
  } else {
    form = <EditForm editGame={props.editGame} />;
  }

  let gameURL = "";

  if (props.gameID) {
    gameURL = `${baseURL}/anno/${props.gameID}`;
  }

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

        // Refresh the collections list
        await fetchGameCollections();

        // Show success message
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

  // Fetch collections that contain this game
  useEffect(() => {
    if (props.gameID) {
      fetchGameCollections();
    }
  }, [props.gameID, fetchGameCollections]);

  return (
    <div className="editor-control">
      {form}
      {props.gameID && (
        <>
          Link to game:
          <Typography.Paragraph copyable className="readable-text-color">
            {gameURL}
          </Typography.Paragraph>
          {gameCollections.length > 0 && (
            <Card
              size="small"
              title={
                <>
                  <FolderOutlined /> In Collections
                </>
              }
              style={{ marginBottom: 16 }}
            >
              <Space wrap style={{ paddingLeft: 4, paddingBottom: 4 }}>
                {gameCollections.map((collection) => (
                  <div
                    key={collection.uuid}
                    style={{ display: "inline-block" }}
                  >
                    <Tag
                      color="blue"
                      style={{
                        margin: "2px",
                        cursor: "default",
                        display: "flex",
                        alignItems: "center",
                        gap: "8px",
                      }}
                    >
                      <Link
                        to={`/collections/${collection.uuid}`}
                        style={{ color: "inherit", textDecoration: "none" }}
                      >
                        <span style={{ cursor: "pointer" }}>
                          {collection.title}
                          {collection.games &&
                            collection.games[0] &&
                            collection.games[0].chapterTitle &&
                            ` (Ch. ${collection.games[0].chapterNumber})`}
                        </span>
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
                          style={{
                            padding: "0 4px",
                            height: "16px",
                            minWidth: "16px",
                            fontSize: "10px",
                            color: "rgba(255, 255, 255, 0.7)",
                          }}
                          onClick={(e) => {
                            e.preventDefault();
                            e.stopPropagation();
                          }}
                        />
                      </Popconfirm>
                    </Tag>
                  </div>
                ))}
              </Space>
            </Card>
          )}
          <p>
            <Button
              onClick={() => setCollectionModalVisible(true)}
              type="default"
              icon={<BookOutlined />}
              style={{ marginBottom: 8, marginRight: 8 }}
            >
              Add to Collection
            </Button>
          </p>
          <p>
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
          </p>
          <AddToCollectionModal
            visible={collectionModalVisible}
            gameId={props.gameID}
            isAnnotated={true}
            onClose={() => setCollectionModalVisible(false)}
            onSuccess={(collectionUuid) => {
              // Refresh the collections list
              fetchGameCollections();
              console.log("Game added to collection:", collectionUuid);
            }}
          />
        </>
      )}
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
        additionalLexica={["NWL20", "NWL18", "CSW19", "CSW21", "OSPS49"]}
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
      {/* <Form.Item label="Show in lobby" name="private">
        <Switch />
      </Form.Item> */}
      <Form.Item>
        <Button type="primary" htmlType="submit">
          Save settings
        </Button>
      </Form.Item>
    </Form>
  );
};
