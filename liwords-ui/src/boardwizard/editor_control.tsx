// Control the editor

import { CopyOutlined } from '@ant-design/icons';
import {
  Button,
  Form,
  Input,
  Popconfirm,
  Select,
  Switch,
  Tooltip,
  Typography,
} from 'antd';
import { Store } from 'antd/lib/form/interface';
import React, { useCallback, useEffect, useMemo } from 'react';
import { ChallengeRule } from '../gen/api/proto/ipc/omgwords_pb';
import { LexiconFormItem } from '../shared/lexicon_display';
import { useGameContextStoreContext } from '../store/store';
import { baseURL } from '../utils/hooks/connect';
import { useMountedState } from '../utils/mounted';

type Props = {
  gameID?: string;
  createNewGame: (
    p1name: string,
    p2name: string,
    lex: string,
    chrule: ChallengeRule
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
  const { useState } = useMountedState();

  let gameURL = '';

  if (props.gameID) {
    gameURL = `${baseURL}/anno/${props.gameID}`;
  }

  const [confirmDelVisible, setConfirmDelVisible] = useState(false);

  return (
    <div className="editor-control">
      {form}
      {props.gameID && (
        <>
          Link to game:
          <Typography.Paragraph copyable className="readable-text-color">
            {gameURL}
          </Typography.Paragraph>
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
              visible={confirmDelVisible}
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
    chrule: ChallengeRule
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
          vals.challengerule
        )
      }
    >
      <Form.Item
        label="Player 1 name"
        name="p1name"
        rules={[
          {
            required: true,
            message: 'Player name is required',
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
            message: 'Player name is required',
          },
        ]}
      >
        <Input maxLength={50} />
      </Form.Item>
      {/* exclude ECWL since that lexicon can't be played without VOID for now */}
      <LexiconFormItem excludedLexica={new Set(['ECWL'])} />
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
  useEffect(() => {
    formref.resetFields();
  }, [gameContext.gameDocument]);
  const [formref] = Form.useForm();

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
