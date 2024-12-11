import { Button, Form, Select, Typography, Upload, message } from 'antd';
import { Store } from 'antd/es/form/interface';
import { flashError, useClient } from '../utils/hooks/connect';
import { GameEventService } from '../gen/api/proto/omgwords_service/omgwords_pb';
import { useGameContextStoreContext } from '../store/store';
import { useNavigate } from 'react-router-dom';
import React, { useCallback, useState } from 'react';
import { ActionType } from '../actions/actions';
import { GameRules } from '../gen/api/proto/ipc/omgwords_pb';
import { defaultLetterDistribution } from '../lobby/sought_game_interactions';
import { LexiconFormItem } from '../shared/lexicon_display';
import { ChallengeRule } from '../gen/api/proto/ipc/omgwords_pb';
import { UploadOutlined } from '@ant-design/icons';

type Props = {
  gcg: string;
  showUpload: boolean;
  showPreview?: boolean;
};

const previewLength = 4;

export const GCGProcessForm = (props: Props) => {
  const eventClient = useClient(GameEventService);
  const [gcg, setGcg] = useState(props.gcg);
  const { dispatchGameContext } = useGameContextStoreContext();
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
    <Form
      style={{ minWidth: 400 }}
      onFinish={async (vals: Store) => {
        if (!gcg) {
          message.error({
            content: 'Please upload a valid gcg file',
          });
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
      {props.showUpload && (
        <Form.Item>
          <Upload
            accept=".gcg"
            beforeUpload={(file) => {
              return new Promise((resolve) => {
                const reader = new FileReader();
                reader.readAsText(file);
                reader.onload = () => {
                  setGcg(reader.result as string);
                };
              });
            }}
          >
            <Button icon={<UploadOutlined />}>Upload a file</Button>
          </Upload>
        </Form.Item>
      )}

      {props.showPreview && gcg && (
        <Form.Item>
          <Typography.Text code>
            {gcg
              .split('\n')
              .slice(0, previewLength)
              .map((line, index) => (
                <React.Fragment key={index}>
                  {line}
                  <br />
                </React.Fragment>
              ))}
            {gcg.split('\n').length > previewLength ? '...' : ''}
          </Typography.Text>
        </Form.Item>
      )}

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
        <Button htmlType="submit">Create new game</Button>
      </Form.Item>
    </Form>
  );
};
