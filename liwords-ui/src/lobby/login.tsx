import React, { useCallback } from 'react';
import { Link } from 'react-router-dom';
import { useMountedState } from '../utils/mounted';
import { useResetStoreContext } from '../store/store';
import './accountForms.scss';

import { Form, Input, Button, Alert } from 'antd';
import { Modal } from '../utils/focus_modal';
import axios from 'axios';
import { toAPIUrl } from '../api/api';

export const Login = React.memo(() => {
  const { useState } = useMountedState();
  const { resetLoginStateStore } = useResetStoreContext();

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

  React.useEffect(() => {
    if (loggedIn) {
      resetLoginStateStore();
    }
  }, [loggedIn, resetLoginStateStore]);

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
              Log in
            </Button>
          </Form.Item>
        </Form>
        {err !== '' ? <Alert message={err} type="error" /> : null}
        <Link to="/password/reset">
          I’m drawing a blank on my password. Help!
        </Link>
      </div>
    </div>
  );
});

export const LoginModal = (props: {
  loginModalVisible: boolean;
  setLoginModalVisible: React.Dispatch<React.SetStateAction<boolean>>;
}) => {
  const { loginModalVisible, setLoginModalVisible } = props;
  const handleHideLoginModal = useCallback(() => {
    setLoginModalVisible(false);
  }, [setLoginModalVisible]);

  return (
    <Modal
      className="login-modal"
      title="Welcome back, friend!"
      visible={loginModalVisible}
      onCancel={handleHideLoginModal}
      footer={null}
      width={332}
      zIndex={1150}
    >
      <Login />
    </Modal>
  );
};
