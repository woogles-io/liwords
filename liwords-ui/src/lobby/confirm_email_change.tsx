import React, { useCallback } from "react";
import { useNavigate } from "react-router";
import { Button } from "antd";
import { useClient } from "../utils/hooks/connect";
import { RegistrationService } from "../gen/api/proto/user_service/user_service_pb";
import { TokenLinkPage } from "./token_link_page";

// ConfirmEmailChange is where the link sent to a new email address lands. It
// needs no login: the token in the link is the proof.
export const ConfirmEmailChange = () => {
  const navigate = useNavigate();
  const registrationClient = useClient(RegistrationService);
  const confirm = useCallback(
    async (token: string) =>
      (await registrationClient.confirmEmailChange({ token })).email,
    [registrationClient],
  );
  const home = (
    <Button type="primary" key="home" onClick={() => navigate("/")}>
      Go to Home
    </Button>
  );

  return (
    <TokenLinkPage
      confirm={confirm}
      loadingText="Changing your email address..."
      successTitle="Email Changed"
      successSubTitle={(email) =>
        `Your account's email address is now ${email}.`
      }
      successExtra={[home]}
      errorTitle="Email Not Changed"
      errorSubTitle={(message) =>
        message ||
        "Unable to change your email address. Please try again from your settings."
      }
      errorExtra={() => [home]}
    />
  );
};
