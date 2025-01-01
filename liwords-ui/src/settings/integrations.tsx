import React, { useEffect, useState } from "react";
import {
  DeleteIntegrationRequestSchema,
  Integration,
  IntegrationService,
} from "../gen/api/proto/user_service/user_service_pb";
import { useLoginStateStoreContext } from "../store/store";
import { useClient } from "../utils/hooks/connect";
import { DeleteOutlined, TwitchOutlined } from "@ant-design/icons";
import { Button, Card, Flex, Popconfirm, Tooltip } from "antd";
import PatreonLogo from "../assets/patreon.svg?react";
import { typedKeys } from "../utils/cwgame/common";
import "./settings.scss";
import { create } from "@bufbuild/protobuf";

export const LoginWithPatreonButton: React.FC<{
  label?: string;
  icon?: React.ReactNode;
}> = ({ label, icon }) => {
  const handleLogin = async () => {
    const clientId = import.meta.env.PUBLIC_PATREON_CLIENT_ID;
    const redirectUri = encodeURIComponent(
      import.meta.env.PUBLIC_PATREON_REDIRECT_URL,
    );
    const scopes = encodeURIComponent("identity identity[email]");
    const csrfToken = Math.random().toString(36).substring(2);

    // Save the CSRF token on the backend
    await fetch("/integrations/csrf", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ csrf: csrfToken }),
    });

    // Combine the CSRF token and the current page's URL
    const state = btoa(
      JSON.stringify({
        csrfToken,
        redirectTo: "/settings/integrations", // Current page URL
      }),
    );

    const authorizationUrl = `https://www.patreon.com/oauth2/authorize?response_type=code&client_id=${clientId}&redirect_uri=${redirectUri}&scope=${scopes}&state=${state}`;

    window.location.href = authorizationUrl;
  };

  const style = label ? { minWidth: 300 } : {};

  return (
    <Button onClick={handleLogin} style={style} icon={icon}>
      {label ? label : ""}
    </Button>
  );
};

export const LoginWithTwitchButton: React.FC<{
  label?: string;
  icon?: React.ReactNode;
}> = ({ label, icon }) => {
  const handleLogin = async () => {
    const clientId = import.meta.env.PUBLIC_TWITCH_CLIENT_ID;
    const redirectUri = encodeURIComponent(
      import.meta.env.PUBLIC_TWITCH_REDIRECT_URL,
    );

    // Define the scopes you need. Adjust these based on your application's requirements.
    const scopes = encodeURIComponent("user:read:email");

    // Generate a CSRF token
    const csrfToken = Math.random().toString(36).substring(2);

    // Save the CSRF token on the backend
    await fetch("/integrations/csrf", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ csrf: csrfToken }),
    });

    // Combine the CSRF token and the current page's URL
    const state = btoa(
      JSON.stringify({
        csrfToken,
        redirectTo: "/settings/integrations",
      }),
    );
    // see https://dev.twitch.tv/docs/authentication/getting-tokens-oauth/#authorization-code-grant-flow
    // Construct the Twitch authorization URL
    const authorizationUrl = `https://id.twitch.tv/oauth2/authorize?response_type=code&client_id=${clientId}&redirect_uri=${redirectUri}&scope=${scopes}&state=${state}`;

    // Redirect the user to Twitch's authorization page
    window.location.href = authorizationUrl;
  };

  // Apply styling based on whether a label is provided
  const style = label ? { minWidth: 300 } : {};

  return (
    <Button onClick={handleLogin} style={style} icon={icon}>
      {label ? label : ""}
    </Button>
  );
};

const apps = {
  patreon: {
    name: "Patreon",
    logo: <PatreonLogo width="16" fill="currentColor" />,
    information: (
      <div>
        Patreon is a membership platform. You are entitled to a number of perks
        based on your tier.
      </div>
    ),
    button: (
      <LoginWithPatreonButton
        icon={<PatreonLogo width="16" fill="currentColor" />}
      />
    ),
  },
  twitch: {
    name: "Twitch",
    logo: <TwitchOutlined />,
    information: (
      <div>
        Twitch is a live streaming platform for gamers. You can connect your
        Twitch account to show yourself as streaming on Woogles (coming soon).
      </div>
    ),
    button: <LoginWithTwitchButton icon={<TwitchOutlined />} />,
  },
};

export const Integrations = () => {
  const { loginState } = useLoginStateStoreContext();
  const [integrations, setIntegrations] = useState<Integration[]>([]);

  const integrationsClient = useClient(IntegrationService);

  useEffect(() => {
    if (!loginState.loggedIn) {
      return;
    }
    const fetchIntegrations = async () => {
      try {
        const integrations = await integrationsClient.getIntegrations({});
        setIntegrations(integrations.integrations);
      } catch (e) {
        console.error(e);
      }
    };
    fetchIntegrations();
  }, [integrationsClient, loginState.loggedIn]);

  const deleteIntegration = async (integration: Integration) => {
    try {
      const ireq = create(DeleteIntegrationRequestSchema, {
        uuid: integration.uuid,
      });
      await integrationsClient.deleteIntegration(ireq);
      setIntegrations(integrations.filter((i) => i.uuid !== integration.uuid));
    } catch (e) {
      console.error(e);
    }
  };

  return (
    <>
      <h3>Your integrations</h3>

      <p>Connect with the following apps:</p>

      <Flex gap="small" style={{ marginBottom: "2rem", marginTop: "1rem" }}>
        {typedKeys(apps).map((app) => (
          <Tooltip title={apps[app].name} key={app}>
            {apps[app].button}
          </Tooltip>
        ))}
      </Flex>

      {integrations.length ? (
        <p>You have the following connected apps:</p>
      ) : null}
      <Flex gap="small" style={{ marginTop: "1rem" }}>
        {integrations.map((integration) => {
          const appName =
            apps[integration.integrationName as keyof typeof apps].name;
          let info =
            apps[integration.integrationName as keyof typeof apps].information;
          if (appName === "Twitch") {
            info = (
              <>
                {info}
                <div style={{ marginTop: 10 }}>
                  {`You are connected as ${integration.integrationDetails["twitch_username"]}. ` +
                    `If you have changed your Twitch username, you will need to reconnect.`}
                </div>
              </>
            );
          }
          return (
            <Card
              key={integration.integrationName}
              title={appName}
              className="integration-card"
              extra={
                <div style={{ marginTop: 5 }}>
                  {apps[integration.integrationName as keyof typeof apps].logo}
                </div>
              }
              style={{ maxWidth: 300 }}
              actions={[
                <Popconfirm
                  title="Delete this integration?"
                  description={`Are you sure you wish to delete your ${appName} connection?`}
                  okText="Yes"
                  cancelText="No"
                  key="delete"
                  onConfirm={() => {
                    deleteIntegration(integration);
                  }}
                >
                  <Button
                    danger
                    style={{ marginTop: -8, marginLeft: 10 }}
                    icon={<DeleteOutlined />}
                  />
                </Popconfirm>,
              ]}
            >
              {info ? info : ""}
            </Card>
          );
        })}
      </Flex>
    </>
  );
};
