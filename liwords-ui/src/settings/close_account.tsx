import React from 'react';
import { Button, Checkbox, Form } from 'antd';
import { PlayerAvatar } from '../shared/player_avatar';
import { PlayerMetadata } from '../gameroom/game_info';

type Props = {
  cancel: () => void;
  player: Partial<PlayerMetadata> | undefined;
  closeAccountNow: () => void;
};

export const CloseAccount = React.memo((props: Props) => {
  return (
    <div className="close-account">
      <h3>Close account</h3>
      <div className="avatar-container">
        <PlayerAvatar player={props.player} />
        <div className="full-name">{props.player?.full_name}</div>
      </div>
      <div className="deletion-rules">
        If you choose to delete your account, it will no longer be accessible.
      </div>
      <div className="deletion-rules">
        All of your data will be deleted, except for past games, per the Woogles
        Terms of Service.
      </div>
      <Form
        onFinish={() => {
          props.closeAccountNow();
        }}
      >
        <div className="stern-warning">
          <Form.Item
            rules={[
              {
                required: true,
                message:
                  'You must acknowledge the finality of closing your account',
                transform: (value) => value || undefined,
                type: 'boolean',
              },
            ]}
            valuePropName="checked"
            name="i-understand-deletion"
          >
            <Checkbox>
              <p className="i-understand">
                I understand that closing my account is an irreversible action
              </p>
            </Checkbox>
          </Form.Item>
        </div>
        <div className="row">
          <Form.Item>
            <Button
              className="cancel-button"
              type="primary"
              onClick={() => props.cancel()}
            >
              Just kidding
            </Button>
          </Form.Item>
          <Form.Item>
            <Button className="close-account-button" htmlType="submit">
              Yes, close my account
            </Button>
          </Form.Item>
        </div>
      </Form>
    </div>
  );
});
