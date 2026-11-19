import React, { useCallback, useEffect, useState } from "react";
import { useNavigate, Link } from "react-router";
import { Button } from "antd";
import { LoginModal } from "./login";
import { useClient } from "../utils/hooks/connect";
import { RegistrationService } from "../gen/api/proto/user_service/user_service_pb";
import { useLoginStateStoreContext } from "../store/store";
import { TokenLinkPage } from "./token_link_page";

const isExpired = (message: string) =>
  message.toLowerCase().includes("expired");

export const VerifyEmail = () => {
  const navigate = useNavigate();
  const registrationClient = useClient(RegistrationService);
  const { loginState } = useLoginStateStoreContext();
  const [loginModalVisible, setLoginModalVisible] = useState(false);

  // Redirect to home after successful login
  useEffect(() => {
    if (loginState.loggedIn) {
      navigate("/", { replace: true });
    }
  }, [loginState.loggedIn, navigate]);

  const confirm = useCallback(
    async (token: string) => {
      await registrationClient.verifyEmail({ token });
    },
    [registrationClient],
  );

  return (
    <TokenLinkPage
      confirm={confirm}
      loadingText="Verifying your email address..."
      successTitle="Email Verified Successfully!"
      successSubTitle={() =>
        "Your email has been verified. You can now log in to your account."
      }
      successExtra={[
        <Button
          type="primary"
          key="login"
          onClick={() => setLoginModalVisible(true)}
        >
          Log In
        </Button>,
        <Button key="home" onClick={() => navigate("/")}>
          Go to Home
        </Button>,
      ]}
      errorTitle="Email Verification Failed"
      errorSubTitle={(message) =>
        isExpired(message)
          ? "Your verification link has expired. Please request a new one from the home page."
          : message || "Unable to verify your email address."
      }
      errorExtra={() => [
        <Button type="primary" key="home" onClick={() => navigate("/")}>
          Go to Home
        </Button>,
        <Link to="/register" key="register">
          <Button>Register Again</Button>
        </Link>,
      ]}
      errorDetail={(message) =>
        isExpired(message) && (
          <p style={{ marginTop: "20px" }}>
            You can request a new verification email by trying to log in on the
            home page.
          </p>
        )
      }
    >
      <LoginModal
        loginModalVisible={loginModalVisible}
        setLoginModalVisible={setLoginModalVisible}
      />
    </TokenLinkPage>
  );
};
