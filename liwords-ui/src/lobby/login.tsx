import React, { useState } from 'react';

import { Form, Input, Button, Alert } from 'antd';
import { Redirect, Link } from 'react-router-dom';
import axios from 'axios';

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
  loggedIn: boolean;
  setLoggedIn: (v: boolean) => void;
};

export const Login = (props: Props) => {
  const [err, setErr] = useState('');

  const onFinish = (values: { [key: string]: string }) => {
    axios
      .post('/twirp/liwords.AuthenticationService/Login', {
        username: values.username,
        password: values.password,
      })
      .then(() => {
        // Automatically will set cookie
        props.setLoggedIn(true);
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

  if (props.loggedIn) {
    return <Redirect push to="/" />;
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

        <Form.Item {...tailLayout}>
          <Button type="primary" htmlType="submit">
            Submit
          </Button>
        </Form.Item>
      </Form>
      {err !== '' ? <Alert message={err} type="error" /> : null}
      <Link to="/register">Register a new name</Link>
    </>
  );
};
