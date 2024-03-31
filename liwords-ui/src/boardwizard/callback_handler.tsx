import { Button, Divider, Form, Select, Typography } from 'antd';
import { LexiconFormItem } from '../shared/lexicon_display';
import { Store } from 'antd/es/form/interface';
import { flashError, useClient } from '../utils/hooks/connect';
import { GameEventService } from '../gen/api/proto/omgwords_service/omgwords_connectweb';
import { GameRules } from '../gen/api/proto/ipc/omgwords_pb';
import { ChallengeRule } from '../gen/api/proto/ipc/omgwords_pb';
import { defaultLetterDistribution } from '../lobby/sought_game_interactions';
import { useCallback } from 'react';
import { useGameContextStoreContext } from '../store/store';
import { ActionType } from '../actions/actions';
import { useNavigate } from 'react-router-dom';

export const CallbackHandler = () => {
  const urlParams = new URLSearchParams(window.location.search);
  // console.log('urlParams were', Array.from(urlParams.entries()));
  const eventClient = useClient(GameEventService);
  const { dispatchGameContext, gameContext } = useGameContextStoreContext();
  const navigate = useNavigate();

  // See editor.tsx for this function definition.
  // XXX: combine with that code.
  const fetchAndDispatchDocument = useCallback(
    async (gid: string, redirect: boolean) => {
      try {
        const resp = await eventClient.getGameDocument({
          gameId: gid,
        });
        console.log('got a game document, dispatching, redirect is', redirect);
        dispatchGameContext({
          actionType: ActionType.InitFromDocument,
          payload: resp,
        });
        if (redirect) {
          // Also, redirect the URL so we can subscribe to the right channel
          // on the socket.
          navigate(`/editor/${encodeURIComponent(gid)}`, { replace: true });
        }
      } catch (e) {
        flashError(e);
      }
    },
    [dispatchGameContext, eventClient, navigate]
  );

  return (
    <div style={{ padding: 20, maxWidth: 400 }}>
      <Typography.Text>
        You're almost done. Select the lexicon and challenge rule for your
        annotated game and click Create new annotated game.
      </Typography.Text>
      <Divider />
      <Form
        onFinish={async (vals: Store) => {
          const gcg = urlParams.get('gcg');
          if (!gcg) {
            console.error('someone set up us the bomb');
            return;
          }

          const lexicon = vals.lexicon;
          try {
            const resp = await eventClient.importGCG({
              gcg: gcg,
              lexicon,
              rules: new GameRules({
                boardLayoutName: 'CrosswordGame',
                letterDistributionName: defaultLetterDistribution(lexicon),
                variantName: 'classic',
              }),
              challengeRule: vals.challengerule,
            });
            fetchAndDispatchDocument(resp.gameId, true);
          } catch (e) {
            flashError(e);
          }
        }}
      >
        <LexiconFormItem
          excludedLexica={new Set(['ECWL'])}
          additionalLexica={['NWL20', 'NWL18', 'CSW19']}
        />

        <Form.Item
          label="Challenge rule"
          name="challengerule"
          rules={[
            {
              required: true,
              message: 'Challenge rule is required',
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
          <Button type="primary" htmlType="submit">
            Create new annotated game
          </Button>
        </Form.Item>
      </Form>
    </div>
  );
};
