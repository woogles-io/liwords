import React from 'react';
import { useMountedState } from '../utils/mounted';
import { Row, Col, Input, Form, Alert, notification, Button } from 'antd';
import qs from 'qs';
import { useLocation } from 'react-router-dom';
import { TopBar } from '../navigation/topbar';
import { connectErrorMessage, useClient } from '../utils/hooks/connect';
import { AuthenticationService } from '../gen/api/proto/user_service/user_service_connectweb';

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

export const NewPassword = () => {
  const { useState } = useMountedState();

  const [err, setErr] = useState('');
  const location = useLocation();
  const params = qs.parse(location.search, { ignoreQueryPrefix: true });
  const authClient = useClient(AuthenticationService);
  const onFinish = async (values: { [key: string]: string }) => {
    if (values.newPassword !== values.confirmnewPassword) {
      setErr('New passwords must match');
      return;
    }
    setErr('');
    try {
      await authClient.resetPasswordStep2({
        password: values.newPassword,
        resetCode: typeof params.t === 'string' ? params.t : '',
      });
      notification.info({
        message: 'Changed',
        description:
          'Your password was successfully changed. Please Log In with your new password.',
      });
    } catch (e) {
      setErr(connectErrorMessage(e));
    }
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
