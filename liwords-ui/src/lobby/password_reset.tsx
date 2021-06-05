import React from 'react';
import { useMountedState } from '../utils/mounted';
import { Row, Col, Input, Form, Alert, notification, Button } from 'antd';
import axios from 'axios';
import { toAPIUrl } from '../api/api';
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

export const PasswordReset = () => {
  const { useState } = useMountedState();

  const [err, setErr] = useState('');

  const onFinish = (values: { [key: string]: string }) => {
    axios
      .post(
        toAPIUrl('user_service.AuthenticationService', 'ResetPasswordStep1'),
        {
          email: values.email,
        }
      )
      .then(() => {
        notification.info({
          message: 'Sent',
          description: 'Please check your email; a reset code was sent.',
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
          <Form
            {...layout}
            name="resetpassword"
            onFinish={onFinish}
            style={{ marginTop: 20 }}
          >
            <Form.Item
              label="Email address"
              name="email"
              rules={[
                {
                  required: true,
                  message: 'Please input your email address!',
                },
              ]}
            >
              <Input />
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
