import React, { useEffect, useState } from "react";
import { useSearchParams, useNavigate } from "react-router";
import { Spin, Result, Button } from "antd";
import { TopBar } from "../navigation/topbar";
import { connectErrorMessage, useClient } from "../utils/hooks/connect";
import { RegistrationService } from "../gen/api/proto/user_service/user_service_pb";
import "./accountForms.scss";

// ConfirmEmailChange is where the link sent to a new email address lands. It
// needs no login: the token in the link is the proof.
export const ConfirmEmailChange = () => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const registrationClient = useClient(RegistrationService);

  const [status, setStatus] = useState<"loading" | "success" | "error">(
    "loading",
  );
  const [email, setEmail] = useState("");
  const [errorMessage, setErrorMessage] = useState("");

  useEffect(() => {
    const token = searchParams.get("token");
    if (!token) {
      setStatus("error");
      setErrorMessage("This link has no token in it.");
      return;
    }
    (async () => {
      try {
        const resp = await registrationClient.confirmEmailChange({ token });
        setEmail(resp.email);
        setStatus("success");
      } catch (e) {
        setStatus("error");
        setErrorMessage(connectErrorMessage(e));
      }
    })();
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
                Changing your email address...
              </p>
            </div>
          )}

          {status === "success" && (
            <Result
              status="success"
              title="Email Changed"
              subTitle={`Your account's email address is now ${email}.`}
              extra={[
                <Button type="primary" key="home" onClick={() => navigate("/")}>
                  Go to Home
                </Button>,
              ]}
            />
          )}

          {status === "error" && (
            <Result
              status="error"
              title="Email Not Changed"
              subTitle={
                errorMessage ||
                "Unable to change your email address. Please try again from your settings."
              }
              extra={[
                <Button type="primary" key="home" onClick={() => navigate("/")}>
                  Go to Home
                </Button>,
              ]}
            />
          )}
        </div>
      </div>
    </>
  );
};
