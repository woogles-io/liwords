import React, { useEffect, useState } from "react";
import { useSearchParams, useNavigate, Link } from "react-router";
import { Spin, Result, Button } from "antd";
import { TopBar } from "../navigation/topbar";
import { LoginModal } from "./login";
import { connectErrorMessage, useClient } from "../utils/hooks/connect";
import { RegistrationService } from "../gen/api/proto/user_service/user_service_pb";
import { useLoginStateStoreContext } from "../store/store";
import "./accountForms.scss";

export const VerifyEmail = () => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const registrationClient = useClient(RegistrationService);
  const { loginState } = useLoginStateStoreContext();

  const [status, setStatus] = useState<"loading" | "success" | "error">(
    "loading",
  );
  const [errorMessage, setErrorMessage] = useState("");
  const [loginModalVisible, setLoginModalVisible] = useState(false);

  // Redirect to home after successful login
  useEffect(() => {
    if (loginState.loggedIn) {
      navigate("/", { replace: true });
    }
  }, [loginState.loggedIn, navigate]);

  useEffect(() => {
    const verifyEmail = async () => {
      const token = searchParams.get("token");

      if (!token) {
        setStatus("error");
        setErrorMessage("No verification token provided");
        return;
      }

      try {
        await registrationClient.verifyEmail({ token });
        setStatus("success");
      } catch (e) {
        setStatus("error");
        setErrorMessage(connectErrorMessage(e));
      }
    };

    verifyEmail();
  }, [searchParams, registrationClient]);

  return (
    <>
      <TopBar />
      <div className="account-form">
        <div className="form-container">
          {status === "loading" && (
            <div style={{ textAlign: "center", padding: "60px 0" }}>
              <Spin size="large" />
              <p style={{ marginTop: "20px", fontSize: "16px" }}>
                Verifying your email address...
              </p>
            </div>
          )}

          {status === "success" && (
            <Result
              status="success"
              title="Email Verified Successfully!"
              subTitle="Your email has been verified. You can now log in to your account."
              extra={[
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
            />
          )}

          {status === "error" && (
            <Result
              status="error"
              title="Email Verification Failed"
              subTitle={
                errorMessage.toLowerCase().includes("expired")
                  ? "Your verification link has expired. Please request a new one from the home page."
                  : errorMessage || "Unable to verify your email address."
              }
              extra={[
                <Button type="primary" key="home" onClick={() => navigate("/")}>
                  Go to Home
                </Button>,
                <Link to="/register" key="register">
                  <Button>Register Again</Button>
                </Link>,
              ]}
            >
              {errorMessage.toLowerCase().includes("expired") && (
                <p style={{ marginTop: "20px" }}>
                  You can request a new verification email by trying to log in
                  on the home page.
                </p>
              )}
            </Result>
          )}
        </div>
      </div>
      <LoginModal
        loginModalVisible={loginModalVisible}
        setLoginModalVisible={setLoginModalVisible}
      />
    </>
  );
};
