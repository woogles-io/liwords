import React, { useState } from 'react';
import axios from 'axios';
import { Input, Form, Button, Alert, Switch, Typography } from 'antd';

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

export const Register = () => {
  const [err, setErr] = useState('');
  const [loggedIn, setLoggedIn] = useState(false);

  const onFinish = (values: { [key: string]: string }) => {
    axios
      .post('/twirp/liwords.RegistrationService/Register', {
        username: values.username,
        password: values.password,
        email: values.email,
      })
      .then(() => {
        // Try logging in after registering.
        axios
          .post('/twirp/liwords.AuthenticationService/Login', {
            username: values.username,
            password: values.password,
          })
          .then(() => {
            // Automatically will set cookie
            setLoggedIn(true);
          })
          .catch((e) => {
            if (e.response) {
              // From Twirp
              setErr(e.response.data.msg);
            } else {
              setErr('unknown error, see console');
              console.log(e);
            }
          });
      })
      .catch((e) => {
        if (e.response) {
          // From Twirp
          setErr(e.response.data.msg);
        } else {
          setErr('unknown error, see console');
          console.log(e);
        }
      });
  };

  if (loggedIn) {
    window.location.replace('/');
  }

  return (
    <>
      <Form
        {...layout}
        name="login"
        onFinish={onFinish}
        style={{ marginTop: 20 }}
      >
        <Form.Item
          label="Username"
          name="username"
          rules={[
            {
              required: true,
              message: 'Please input your username!',
            },
          ]}
        >
          <Input />
        </Form.Item>

        <Form.Item
          label="Password"
          name="password"
          rules={[
            {
              required: true,
              message: 'Please input your password!',
            },
          ]}
        >
          <Input.Password />
        </Form.Item>

        <Form.Item
          label="Email"
          name="email"
          rules={[
            {
              required: true,
              message: 'Please input your email. We promise not to spam you.',
            },
          ]}
        >
          <Input />
        </Form.Item>

        <Typography.Title level={4}>
          Cheating of any form is not allowed on this site. We are very serious
          about this and will ban any cheaters. Please do not take advantage of
          your fellow word game lovers. By registering on this site, you promise
          to never cheat on it.
        </Typography.Title>

        <Form.Item
          rules={[
            {
              required: true,
              message: 'You must agree to this condition',
              transform: (value) => value || undefined,
              type: 'boolean',
            },
          ]}
          valuePropName="checked"
          initialValue={false}
          name="nocheating"
          label="I promise never to cheat"
        >
          <Switch />
        </Form.Item>

        <Form.Item {...tailLayout}>
          <Button type="primary" htmlType="submit">
            Submit
          </Button>
        </Form.Item>
      </Form>
      {err !== '' ? <Alert message={err} type="error" /> : null}
    </>
  );
};
