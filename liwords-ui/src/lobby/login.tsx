import React, { useState } from 'react';
import { Link, useHistory } from 'react-router-dom';
import { useResetStoreContext } from '../store/store';
import './accountForms.scss';

import { Form, Input, Button, Alert } from 'antd';
import axios from 'axios';
import { toAPIUrl } from '../api/api';

export const Login = React.memo(() => {
  const stillMountedRef = React.useRef(true);
  React.useEffect(() => () => void (stillMountedRef.current = false), []);
  const { resetStore } = useResetStoreContext();

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
        if (stillMountedRef.current) {
          setLoggedIn(true);
        }
      })
      .catch((e) => {
        if (e.response) {
          // From Twirp
          if (stillMountedRef.current) {
            setErr(e.response.data.msg);
          }
        } else {
          if (stillMountedRef.current) {
            setErr('unknown error, see console');
          }
          console.log(e);
        }
      });
  };

  const history = useHistory();
  React.useEffect(() => {
    if (loggedIn) {
      resetStore();
      history.replace('/');
    }
  }, [history, loggedIn, resetStore]);

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
        <Link to="/password/reset" onClick={resetStore}>
          I’m drawing a blank on my password. Help!
        </Link>
      </div>
    </div>
  );
});
