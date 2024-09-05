import { Button, Card, Col, Dropdown, Form, Layout, Menu, Row } from 'antd';
import { useCallback, useEffect, useState } from 'react';
import { TopBar } from '../navigation/topbar';
import { EditorControl } from './editor_control';

import { ChallengeRule } from '../gen/api/proto/ipc/omgwords_pb';
import { BroadcastGamesResponse_BroadcastGame } from '../gen/api/proto/omgwords_service/omgwords_pb';
import { useClient } from '../utils/hooks/connect';
import { GameEventService } from '../gen/api/proto/omgwords_service/omgwords_connect';
import { AnnotatedGamesHistoryCard } from '../profile/annotated_games_history';
import { useLoginStateStoreContext } from '../store/store';
import { Content } from 'antd/lib/layout/layout';
import {
  EditOutlined,
  FileImageOutlined,
  PlusOutlined,
  UploadOutlined,
} from '@ant-design/icons';
import { MenuProps } from 'antd/lib';
import { LexiconFormItem } from '../shared/lexicon_display';
import { GCGProcessForm } from './gcg_process_form';
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
  const [recentAnnotatedGames, setRecentAnnotatedGames] = useState<
    Array<BroadcastGamesResponse_BroadcastGame>
  >([]);
  const { loginState } = useLoginStateStoreContext();
  const [recentAnnotatedGamesOffset, setRecentAnnotatedGamesOffset] =
    useState(0);
  const [selectedMenu, setSelectedMenu] = useState('');

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

  const handleMenuClick = (e: { key: string }) => {
    setSelectedMenu(e.key);
  };

  const getCallbackURI = useCallback((): string => {
    const loc = window.location;

    return `${loc.protocol}//${loc.host}/scrabblecam/callback`;
  }, []);

  const menuItems: MenuProps['items'] = [
    {
      label: 'Create a new game from scratch',
      key: 'createGame',
      icon: <EditOutlined />,
    },
    {
      label: 'Upload a GCG file',
      key: 'uploadGCG',
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
      key: 'image',
      icon: <FileImageOutlined />,
    },
  ];

  const menuProps = {
    items: menuItems,
    onClick: handleMenuClick,
  };

  return (
    <div className="game-container">
      <TopBar />

      <Layout>
        <Content>
          <div
            style={{
              display: 'flex',
              justifyContent: 'center',
              padding: 24,
            }}
          >
            <Dropdown menu={menuProps} trigger={['click']}>
              <Button type="primary" icon={<PlusOutlined />}>
                Add an annotated game
              </Button>
            </Dropdown>
          </div>
          <div
            style={{
              display: 'flex',
              justifyContent: 'center',
            }}
          >
            {selectedMenu === 'createGame' && (
              <div style={{ display: 'flex', justifyContent: 'center' }}>
                <EditorControl
                  createNewGame={props.createNewGame}
                  deleteGame={() => {}}
                  editGame={() => {}}
                />
              </div>
            )}
            {selectedMenu === 'uploadGCG' && (
              <div
                style={{
                  display: 'flex',
                  justifyContent: 'center',
                }}
              >
                <GCGProcessForm gcg="" showUpload showPreview />
              </div>
            )}
          </div>

          <div style={{ padding: 24 }}>
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
