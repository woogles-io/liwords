import React from 'react';
import { Alert, Button, Checkbox, Col, Form, Input, Row } from 'antd';
import { PlayerAvatar } from '../shared/player_avatar';
import { PlayerMetadata } from '../gameroom/game_info';

type Props = {
  cancel: () => void;
  player: Partial<PlayerMetadata> | undefined;
  closeAccountNow: (password: string) => void;
  err: string;
};

export const CloseAccount = React.memo((props: Props) => {
  return (
    <div className="close-account">
      <h3>Delete account</h3>
      <div className="avatar-container">
        <PlayerAvatar player={props.player} />
        <div className="full-name">{props.player?.full_name}</div>
      </div>
      <div className="deletion-rules">
        If you choose to delete your account, it will{' '}
        <strong>no longer be accessible.</strong>
      </div>
      <div className="deletion-rules">
        All of your data will be deleted, except for past games, per the Woogles
        Terms of Service.
      </div>
      <div className="deletion-rules">
        <strong>
          Note: Account deletion is not a way to change usernames. It will not
          work.
        </strong>
      </div>
      <Form
        onFinish={(values: { [key: string]: string }) => {
          props.closeAccountNow(values.password);
        }}
      >
        <Form.Item
          label="Please confirm your password"
          name="password"
          rules={[
            {
              required: true,
              message: 'Confirm your identity by entering your password',
            },
          ]}
        >
          <Input.Password />
        </Form.Item>
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
              <div className="i-understand">
                I understand that closing my account is an irreversible action.
              </div>
            </Checkbox>
          </Form.Item>
        </div>
        <Row>
          <Col span={12}>
            <Form.Item>
              <Button
                className="cancel-button"
                type="primary"
                onClick={() => props.cancel()}
              >
                No, keep my account
              </Button>
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item>
              <Button className="close-account-button" htmlType="submit">
                Yes, delete my account
              </Button>
            </Form.Item>
          </Col>
        </Row>
      </Form>
      {props.err !== '' ? <Alert message={props.err} type="error" /> : null}
    </div>
  );
});
