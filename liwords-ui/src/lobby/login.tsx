import React, { useState } from 'react';
import './accountForms.scss';

import { Form, Input, Button, Alert } from 'antd';
// import { Link } from 'react-router-dom';
import axios from 'axios';
import { toAPIUrl } from '../api/api';

export const Login = React.memo(() => {
  const [err, setErr] = useState('');
  const [loggedIn, setLoggedIn] = useState(false);
  const onFinish = (values: { [key: string]: string }) => {
    axios
      .post(
        toAPIUrl('user_service.AuthenticationService', 'Login'),
        {
          username: values.username,
          password: values.password,
        },
        { withCredentials: true }
      )
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
  };

  if (loggedIn) {
    window.location.replace('/');
  }

  return (
    <div className="account">
      <div className="account-form login">
        <Form name="login" onFinish={onFinish}>
          <Form.Item
            name="username"
            rules={[
              {
                required: true,
                message: 'Please input your username!',
              },
            ]}
          >
            <Input placeholder="Username" maxLength={20} />
          </Form.Item>

          <Form.Item
            name="password"
            rules={[
              {
                required: true,
                message: 'Please input your password!',
              },
            ]}
          >
            <Input.Password placeholder="Password" />
          </Form.Item>

          <Form.Item>
            <Button type="primary" htmlType="submit">
              Log In
            </Button>
          </Form.Item>
        </Form>
        {err !== '' ? <Alert message={err} type="error" /> : null}
        <a href="/password/reset">I’m drawing a blank on my password. Help!</a>
      </div>
    </div>
  );
});
