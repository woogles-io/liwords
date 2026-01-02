import { Alert, Button, Dropdown, Layout } from "antd";
import { useCallback, useEffect, useState } from "react";
import { TopBar } from "../navigation/topbar";
import { EditorControl } from "./editor_control";

import { ChallengeRule } from "../gen/api/proto/ipc/omgwords_pb";
import { BroadcastGamesResponse_BroadcastGame } from "../gen/api/proto/omgwords_service/omgwords_pb";
import { useClient } from "../utils/hooks/connect";
import { GameEventService } from "../gen/api/proto/omgwords_service/omgwords_pb";
import { AnnotatedGamesHistoryCard } from "../profile/annotated_games_history";
import { useLoginStateStoreContext } from "../store/store";
import { GameComment } from "../gen/api/proto/comments_service/comments_service_pb";
import { GameCommentService } from "../gen/api/proto/comments_service/comments_service_pb";
import { RecentCommentsCard } from "../components/RecentCommentsCard";
import { RecentCollectionsCard } from "../components/RecentCollectionsCard";
import {
  CollectionsService,
  Collection,
} from "../gen/api/proto/collections_service/collections_service_pb";
import { Content } from "antd/lib/layout/layout";
import {
  EditOutlined,
  FileImageOutlined,
  PlusOutlined,
  UploadOutlined,
} from "@ant-design/icons";
import { MenuProps } from "antd/lib";
import { GCGProcessForm } from "./gcg_process_form";
// When no game is visible, this is the page that is visible.

type Props = {
  createNewGame: (
    p1name: string,
    p2name: string,
    lex: string,
    chrule: ChallengeRule,
  ) => void;
};

const annotatedPageSize = 40;
const commentsPageSize = 10;
const collectionsPageSize = 10;

export const EditorLandingPage = (props: Props) => {
  const [recentAnnotatedGames, setRecentAnnotatedGames] = useState<
    Array<BroadcastGamesResponse_BroadcastGame>
  >([]);
  const [hasMoreAnnotatedGames, setHasMoreAnnotatedGames] = useState(true);
  const [recentComments, setRecentComments] = useState<Array<GameComment>>([]);
  const [recentCollections, setRecentCollections] = useState<Array<Collection>>(
    [],
  );
  const { loginState } = useLoginStateStoreContext();
  const [recentAnnotatedGamesOffset, setRecentAnnotatedGamesOffset] =
    useState(0);
  const [recentCommentsOffset, setRecentCommentsOffset] = useState(0);
  const [recentCollectionsOffset, setRecentCollectionsOffset] = useState(0);
  const [selectedMenu, setSelectedMenu] = useState("");
  const [isDesktop, setIsDesktop] = useState(window.innerWidth > 768);

  const fetchPrevAnnotatedGames = useCallback(() => {
    setRecentAnnotatedGamesOffset((r) => Math.max(r - annotatedPageSize, 0));
  }, []);
  const fetchNextAnnotatedGames = useCallback(() => {
    setRecentAnnotatedGamesOffset((r) => r + annotatedPageSize);
  }, []);
  const fetchPrevComments = useCallback(() => {
    setRecentCommentsOffset((r) => Math.max(r - commentsPageSize, 0));
  }, []);
  const fetchNextComments = useCallback(() => {
    setRecentCommentsOffset((r) => r + commentsPageSize);
  }, []);
  const fetchPrevCollections = useCallback(() => {
    setRecentCollectionsOffset((r) => Math.max(r - collectionsPageSize, 0));
  }, []);
  const fetchNextCollections = useCallback(() => {
    setRecentCollectionsOffset((r) => r + collectionsPageSize);
  }, []);
  const gameEventClient = useClient(GameEventService);
  const commentsClient = useClient(GameCommentService);
  const collectionsClient = useClient(CollectionsService);

  useEffect(() => {
    (async () => {
      try {
        const resp = await gameEventClient.getRecentAnnotatedGames({
          limit: annotatedPageSize,
          offset: recentAnnotatedGamesOffset,
        });
        setRecentAnnotatedGames(resp.games);
        // If we got fewer games than requested, there are no more
        setHasMoreAnnotatedGames(resp.games.length === annotatedPageSize);
      } catch (e) {
        console.log(e);
      }
    })();
  }, [gameEventClient, recentAnnotatedGamesOffset]);

  useEffect(() => {
    (async () => {
      try {
        const resp = await commentsClient.getCommentsForAllGames({
          limit: commentsPageSize,
          offset: recentCommentsOffset,
        });
        setRecentComments(resp.comments);
      } catch (e) {
        console.log(e);
      }
    })();
  }, [commentsClient, recentCommentsOffset]);

  useEffect(() => {
    (async () => {
      try {
        const resp = await collectionsClient.getRecentlyUpdatedCollections({
          limit: collectionsPageSize,
          offset: recentCollectionsOffset,
        });
        setRecentCollections(resp.collections);
      } catch (e) {
        console.log(e);
      }
    })();
  }, [collectionsClient, recentCollectionsOffset]);

  useEffect(() => {
    const handleResize = () => {
      setIsDesktop(window.innerWidth > 768);
    };

    window.addEventListener("resize", handleResize);
    return () => window.removeEventListener("resize", handleResize);
  }, []);

  const handleMenuClick = (e: { key: string }) => {
    setSelectedMenu(e.key);
  };

  const getCallbackURI = useCallback((): string => {
    const loc = window.location;

    return `${loc.protocol}//${loc.host}/scrabblecam/callback`;
  }, []);

  const menuItems: MenuProps["items"] = [
    {
      label: "Create a new game from scratch",
      key: "createGame",
      icon: <EditOutlined />,
    },
    {
      label: "Upload a GCG file",
      key: "uploadGCG",
      icon: <UploadOutlined />,
    },
    {
      label: (
        <a
          href={`https://www.scrabblecam.com/annotate?callback_url=${encodeURIComponent(getCallbackURI())}`}
        >
          Annotate from your camera with Scrabblecam
        </a>
      ),
      key: "image",
      icon: <FileImageOutlined />,
    },
  ];

  const menuProps = {
    items: menuItems,
    onClick: handleMenuClick,
  };

  return (
    <div className="game-container editor">
      <TopBar />

      <Layout>
        <Content style={{ minHeight: "100vh" }}>
          <div
            style={{
              display: "flex",
              justifyContent: "center",
              padding: 24,
            }}
          >
            <Dropdown menu={menuProps} trigger={["click"]}>
              <Button type="primary" icon={<PlusOutlined />}>
                Add an annotated game
              </Button>
            </Dropdown>
          </div>
          <div
            style={{
              display: "flex",
              justifyContent: "center",
              padding: "0 24px 24px 24px",
            }}
          >
            <Alert
              message="The annotator UX has been updated!"
              description={
                <span>
                  We've made improvements to the annotation interface.{" "}
                  <a
                    href="https://youtu.be/xdfh9JyGI0s"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    Watch our tutorial video
                  </a>{" "}
                  to learn about the new features.
                </span>
              }
              type="info"
              showIcon
              style={{ maxWidth: "600px" }}
            />
          </div>
          <div
            style={{
              display: "flex",
              justifyContent: "center",
            }}
          >
            {selectedMenu === "createGame" && (
              <div style={{ display: "flex", justifyContent: "center" }}>
                <EditorControl
                  createNewGame={props.createNewGame}
                  deleteGame={() => {}}
                  editGame={() => {}}
                />
              </div>
            )}
            {selectedMenu === "uploadGCG" && (
              <div
                style={{
                  display: "flex",
                  justifyContent: "center",
                }}
              >
                <GCGProcessForm gcg="" showUpload showPreview />
              </div>
            )}
          </div>

          <div
            className="editor-layout"
            style={{
              display: "flex",
              gap: "24px",
              padding: "0 24px 24px 24px",
              flexDirection: isDesktop ? "row" : "column",
            }}
          >
            <div
              className="games-section"
              style={{
                flex: isDesktop ? "1 1 70%" : "1",
                minWidth: "0", // Allow flex item to shrink
                overflow: "auto",
              }}
            >
              <AnnotatedGamesHistoryCard
                games={recentAnnotatedGames}
                fetchPrev={
                  recentAnnotatedGamesOffset > 0
                    ? fetchPrevAnnotatedGames
                    : undefined
                }
                fetchNext={
                  hasMoreAnnotatedGames ? fetchNextAnnotatedGames : undefined
                }
                loggedInUserID={loginState.userID}
                showAnnotator
              />
            </div>
            <div
              className="comments-sidebar"
              style={{
                flex: isDesktop ? "0 0 30%" : "1",
                maxWidth: isDesktop ? "400px" : "none",
                minWidth: isDesktop ? "300px" : "0",
                overflow: "auto",
              }}
            >
              <RecentCommentsCard
                comments={recentComments}
                fetchPrev={fetchPrevComments}
                fetchNext={fetchNextComments}
              />
            </div>
          </div>

          <div
            className="collections-section"
            style={{
              padding: "0 24px 24px 24px",
            }}
          >
            <RecentCollectionsCard
              collections={recentCollections}
              fetchPrev={fetchPrevCollections}
              fetchNext={fetchNextCollections}
              loggedInUserUuid={loginState.userID}
            />
          </div>
        </Content>
      </Layout>
    </div>
  );
};
