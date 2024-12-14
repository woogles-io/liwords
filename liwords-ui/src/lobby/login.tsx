import React, { useCallback, useState } from "react";
import { Link } from "react-router-dom";
import { useResetStoreContext } from "../store/store";
import "./accountForms.scss";

import { Form, Input, Button, Alert } from "antd";
import { Modal } from "../utils/focus_modal";
import {
  connectErrorMessage,
  flashError,
  useClient,
} from "../utils/hooks/connect";
import { AuthenticationService } from "../gen/api/proto/user_service/user_service_pb";

export const Login = React.memo(() => {
  const { resetStore } = useResetStoreContext();

  const [err, setErr] = useState("");
  const [loggedIn, setLoggedIn] = useState(false);
  const authClient = useClient(AuthenticationService);
  const onFinish = async (values: { [key: string]: string }) => {
    try {
      await authClient.login({
        username: values.username,
        password: values.password,
      });
      setLoggedIn(true);
    } catch (e) {
      setErr(connectErrorMessage(e));
      flashError(e);
    }
  };

  React.useEffect(() => {
    if (loggedIn) {
      resetStore();
    }
  }, [loggedIn, resetStore]);

  return (
    <div className="account">
      <div className="account-form login">
        <Form name="login" onFinish={onFinish}>
          <Form.Item
            name="username"
            rules={[
              {
                required: true,
                message: "Please input your username!",
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
                message: "Please input your password!",
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
        {err !== "" ? <Alert message={err} type="error" /> : null}
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
      open={loginModalVisible}
      onCancel={handleHideLoginModal}
      footer={null}
      width={332}
      zIndex={1150}
    >
      <Login />
    </Modal>
  );
};
