import React, { useState } from 'react';
import { Row, Col, Input, Form, Alert, notification, Button } from 'antd';
import axios from 'axios';
import qs from 'qs';
import { useLocation } from 'react-router-dom';
import { TopBar } from '../topbar/topbar';

const layout = {
  labelCol: {
    span: 8,
  },
  wrapperCol: {
    span: 12,
  },
};
const tailLayout = {
  wrapperCol: {
    offset: 8,
    span: 12,
  },
};

type Props = {
  username: string;
  loggedIn: boolean;
  connectedToSocket: boolean;
};

export const NewPassword = (props: Props) => {
  const [err, setErr] = useState('');
  const location = useLocation();
  const params = qs.parse(location.search, { ignoreQueryPrefix: true });

  const onFinish = (values: { [key: string]: string }) => {
    if (values.newPassword !== values.confirmnewPassword) {
      setErr('New passwords must match');
      return;
    }
    setErr('');
    axios
      .post('/twirp/user_service.AuthenticationService/ResetPasswordStep2', {
        password: values.newPassword,
        resetCode: params.t,
      })
      .then(() => {
        notification.info({
          message: 'Changed',
          description:
            'Your password was successfully changed. Please Log In with your new password.',
        });
      })
      .catch((e) => {
        if (e.response) {
          setErr(e.response.data.msg);
        }
      });
  };

  return (
    <>
      <Row>
        <Col span={24}>
          <TopBar
            username={props.username}
            loggedIn={props.loggedIn}
            connectedToSocket={props.connectedToSocket}
          />
        </Col>
      </Row>
      <Row>
        <Col span={24}>
          <Form
            {...layout}
            name="resetpassword"
            onFinish={onFinish}
            style={{ marginTop: 20 }}
          >
            <Form.Item
              label="New Password"
              name="newPassword"
              rules={[
                {
                  required: true,
                  message: 'Please input your new password!',
                },
              ]}
            >
              <Input.Password />
            </Form.Item>

            <Form.Item
              label="Confirm New Password"
              name="confirmnewPassword"
              rules={[
                {
                  required: true,
                  message: 'Please confirm your new password!',
                },
              ]}
            >
              <Input.Password />
            </Form.Item>

            <Form.Item {...tailLayout}>
              <Button type="primary" htmlType="submit">
                Submit
              </Button>
            </Form.Item>
          </Form>
        </Col>
      </Row>

      {err !== '' ? <Alert message={err} type="error" /> : null}
    </>
  );
};
