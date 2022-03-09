import React from 'react';
import { useMountedState } from '../utils/mounted';
import { Row, Col, Input, Form, Alert, notification, Button } from 'antd';
import axios from 'axios';
import qs from 'qs';
import { useLocation } from 'react-router-dom';
import { TopBar } from '../navigation/topbar';
import { toAPIUrl } from '../api/api';

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

type Props = {};

export const NewPassword = (props: Props) => {
  const { useState } = useMountedState();

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
      .post(
        toAPIUrl('user_service.AuthenticationService', 'ResetPasswordStep2'),
        {
          password: values.newPassword,
          resetCode: params.t,
        },
        { withCredentials: true }
      )
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
          <TopBar />
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
              label="New password"
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
              label="Confirm new password"
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
