import { Card, Layout } from 'antd';
import React, { useCallback, useEffect } from 'react';
import { TopBar } from '../navigation/topbar';
import { EditorControl } from './editor_control';

import { ChallengeRule } from '../gen/api/proto/ipc/omgwords_pb';
import { useMountedState } from '../utils/mounted';
import { BroadcastGamesResponse_BroadcastGame } from '../gen/api/proto/omgwords_service/omgwords_pb';
import { useClient } from '../utils/hooks/connect';
import { GameEventService } from '../gen/api/proto/omgwords_service/omgwords_connectweb';
import { AnnotatedGamesHistoryCard } from '../profile/annotated_games_history';
import { useLoginStateStoreContext } from '../store/store';
import { Content } from 'antd/lib/layout/layout';
// When no game is visible, this is the page that is visible.

type Props = {
  createNewGame: (
    p1name: string,
    p2name: string,
    lex: string,
    chrule: ChallengeRule
  ) => void;
};

const annotatedPageSize = 40;

export const EditorLandingPage = (props: Props) => {
  const { useState } = useMountedState();
  const [recentAnnotatedGames, setRecentAnnotatedGames] = useState<
    Array<BroadcastGamesResponse_BroadcastGame>
  >([]);
  const { loginState } = useLoginStateStoreContext();
  const [recentAnnotatedGamesOffset, setRecentAnnotatedGamesOffset] =
    useState(0);
  const fetchPrevAnnotatedGames = useCallback(() => {
    setRecentAnnotatedGamesOffset((r) => Math.max(r - annotatedPageSize, 0));
  }, []);
  const fetchNextAnnotatedGames = useCallback(() => {
    setRecentAnnotatedGamesOffset((r) => r + annotatedPageSize);
  }, []);
  const gameEventClient = useClient(GameEventService);

  useEffect(() => {
    (async () => {
      try {
        const resp = await gameEventClient.getRecentAnnotatedGames({
          limit: annotatedPageSize,
          offset: recentAnnotatedGamesOffset,
        });
        setRecentAnnotatedGames(resp.games);
      } catch (e) {
        console.log(e);
      }
    })();
  }, [gameEventClient, recentAnnotatedGamesOffset]);

  return (
    <div className="game-container">
      <TopBar />

      <Layout>
        <Content>
          <Card
            title="Create a new annotated game"
            className="editor-new"
            style={{ maxWidth: 400, margin: 'auto', marginTop: 24 }}
          >
            <EditorControl
              createNewGame={props.createNewGame}
              deleteGame={() => {}}
              editGame={() => {}}
            />
          </Card>
          <div style={{ paddingTop: 24, paddingLeft: 24, paddingRight: 24 }}>
            <AnnotatedGamesHistoryCard
              games={recentAnnotatedGames}
              fetchPrev={fetchPrevAnnotatedGames}
              fetchNext={fetchNextAnnotatedGames}
              loggedInUserID={loginState.userID}
              showAnnotator
            />
          </div>
        </Content>
      </Layout>
    </div>
  );
};
