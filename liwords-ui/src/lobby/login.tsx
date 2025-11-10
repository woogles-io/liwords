import React, { useCallback, useState } from "react";
import { Link } from "react-router";
import { useResetStoreContext } from "../store/store";
import "./accountForms.scss";

import { Form, Input, Button, Alert } from "antd";
import { Modal } from "../utils/focus_modal";
import {
  connectErrorMessage,
  flashError,
  useClient,
} from "../utils/hooks/connect";
import {
  AuthenticationService,
  RegistrationService,
} from "../gen/api/proto/user_service/user_service_pb";

export const Login = React.memo(() => {
  const { resetStore } = useResetStoreContext();

  const [err, setErr] = useState("");
  const [loggedIn, setLoggedIn] = useState(false);
  const [showResendVerification, setShowResendVerification] = useState(false);
  const [resendExpanded, setResendExpanded] = useState(false);
  const [resendEmail, setResendEmail] = useState("");
  const [resendMessage, setResendMessage] = useState("");
  const [resendLoading, setResendLoading] = useState(false);

  const authClient = useClient(AuthenticationService);
  const registrationClient = useClient(RegistrationService);

  const onFinish = async (values: { [key: string]: string }) => {
    try {
      await authClient.login({
        username: values.username,
        password: values.password,
      });
      setLoggedIn(true);
    } catch (e) {
      const errorMsg = connectErrorMessage(e);
      setErr(errorMsg);
      flashError(e);

      // Check if error is due to unverified email
      if (errorMsg.toLowerCase().includes('verify your email')) {
        setShowResendVerification(true);
        setResendExpanded(false); // Reset expansion state
      } else {
        setShowResendVerification(false);
        setResendExpanded(false);
      }
    }
  };

  const handleResendVerification = async () => {
    if (!resendEmail) {
      setResendMessage("Please enter your email address");
      return;
    }

    setResendLoading(true);
    setResendMessage("");

    try {
      await registrationClient.resendVerificationEmail({ email: resendEmail });
      setResendMessage("Verification email sent! Please check your inbox and spam folder.");
      setShowResendVerification(false);
    } catch (e) {
      setResendMessage(connectErrorMessage(e));
    } finally {
      setResendLoading(false);
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

        {showResendVerification && (
          <div style={{ marginTop: "15px", marginBottom: "15px" }}>
            <a
              onClick={() => setResendExpanded(!resendExpanded)}
              style={{
                cursor: "pointer",
                textDecoration: "underline",
                color: "#1890ff",
                display: "block",
                marginBottom: resendExpanded ? "15px" : "0"
              }}
            >
              Need a new verification email?
            </a>
            {resendExpanded && (
              <Form style={{ marginTop: "10px" }}>
                <Form.Item>
                  <Input
                    placeholder="Enter your email address"
                    value={resendEmail}
                    onChange={(e) => setResendEmail(e.target.value)}
                    size="large"
                  />
                </Form.Item>
                <Form.Item>
                  <Button
                    type="primary"
                    onClick={handleResendVerification}
                    loading={resendLoading}
                  >
                    Resend Verification Email
                  </Button>
                </Form.Item>
                {resendMessage && (
                  <Alert
                    message={resendMessage}
                    type={resendMessage.includes('sent') ? 'success' : 'error'}
                  />
                )}
              </Form>
            )}
          </div>
        )}

        <Link to="/password/reset">
          I'm drawing a blank on my password. Help!
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
