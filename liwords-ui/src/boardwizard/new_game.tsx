import { Card, Col, Row } from 'antd';
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
      <Row style={{ marginTop: 48 }}>
        <Col lg={6} offset={4}>
          <Card title="Editor controls" className="editor-new">
            <EditorControl
              createNewGame={props.createNewGame}
              deleteGame={() => {}}
              editGame={() => {}}
            />
          </Card>
        </Col>
      </Row>
      <Row style={{ marginTop: 48 }}>
        <Col lg={16} offset={4}>
          <AnnotatedGamesHistoryCard
            games={recentAnnotatedGames}
            fetchPrev={fetchPrevAnnotatedGames}
            fetchNext={fetchNextAnnotatedGames}
            loggedInUserID={loginState.userID}
            showAnnotator
          />
        </Col>
      </Row>
    </div>
  );
};
