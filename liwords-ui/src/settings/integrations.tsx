import React, { useEffect, useRef, useState } from "react";
import {
  DeleteIntegrationRequestSchema,
  Integration,
  IntegrationService,
  OrganizationService,
  ConnectOrganizationRequestSchema,
  SubmitVerificationRequestSchema,
  GetMyOrganizationsRequestSchema,
  DisconnectOrganizationRequestSchema,
  RefreshTitlesRequestSchema,
  OrganizationTitle,
} from "../gen/api/proto/user_service/user_service_pb";
import { useLoginStateStoreContext } from "../store/store";
import { useClient, flashError } from "../utils/hooks/connect";
import {
  DeleteOutlined,
  TwitchOutlined,
  UploadOutlined,
  QuestionCircleOutlined,
  ReloadOutlined,
} from "@ant-design/icons";
import {
  Button,
  Card,
  Flex,
  Popconfirm,
  Tooltip,
  Form,
  Input,
  Upload,
  message,
  Space,
  Table,
  Tag,
  Divider,
  Modal,
} from "antd";
import PatreonLogo from "../assets/patreon.svg?react";
import { typedKeys } from "../utils/cwgame/common";
import "./settings.scss";
import { create } from "@bufbuild/protobuf";

export const usePatreonLogin = (wooglesRedirectUri?: string) => {
  const handleLogin = async (e?: React.MouseEvent) => {
    e?.preventDefault();
    // Stop event propagation to prevent dropdown closure
    e?.stopPropagation();
    const clientId = import.meta.env.PUBLIC_PATREON_CLIENT_ID;
    const redirectUri = encodeURIComponent(
      import.meta.env.PUBLIC_PATREON_REDIRECT_URL,
    );
    const scopes = encodeURIComponent("identity identity[email]");
    const csrfToken = Math.random().toString(36).substring(2);

    await fetch("/integrations/csrf", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ csrf: csrfToken }),
    });

    const state = btoa(
      JSON.stringify({
        csrfToken,
        redirectTo: wooglesRedirectUri || "/settings/integrations",
      }),
    );

    const authorizationUrl = `https://www.patreon.com/oauth2/authorize?response_type=code&client_id=${clientId}&redirect_uri=${redirectUri}&scope=${scopes}&state=${state}`;
    window.location.href = authorizationUrl;
  };

  return handleLogin;
};

type ButtonProps = React.ComponentProps<typeof Button>;

// Then extend those props with your custom ones
interface LoginWithPatreonButtonProps extends ButtonProps {
  label?: string;
  icon?: React.ReactNode;
}

// Button version
export const LoginWithPatreonButton: React.FC<LoginWithPatreonButtonProps> = ({
  label,
  icon,
  ...props
}) => {
  const handleLogin = usePatreonLogin();

  return (
    <Button onClick={handleLogin} icon={icon} {...props}>
      {label ? label : ""}
    </Button>
  );
};

// Link version
export const LoginWithPatreonLink: React.FC<{
  className?: string;
  children?: React.ReactNode;
}> = ({ className, children, ...props }) => {
  const handleLogin = usePatreonLogin("/?botdialog");

  const linkRef = useRef<HTMLAnchorElement>(null);
  // Handle mouse down to prevent dropdown toggle
  const handleMouseDown = (e: React.MouseEvent<HTMLAnchorElement>) => {
    e.preventDefault();
    e.stopPropagation();

    // Programmatically focus to maintain a11y
    linkRef.current?.focus();
  };

  return (
    <a
      onClick={handleLogin}
      className={className}
      onMouseDown={handleMouseDown}
      ref={linkRef}
      {...props}
    >
      {children || "Login with Patreon"}
    </a>
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

const organizationInfo = {
  naspa: {
    name: "NASPA",
    fullName: "North American SCRABBLE Players Association",
    description:
      "NASPA is the official Scrabble organization in North America. You can connect using your NASPA member ID and password.",
    requiresCredentials: true,
    requiresVerification: false,
  },
  wespa: {
    name: "WESPA",
    fullName: "World English-Language Scrabble Players Association",
    description:
      "WESPA is the worldwide Scrabble organization. Connection requires identity verification with a photo ID.",
    requiresCredentials: false,
    requiresVerification: true,
  },
  absp: {
    name: "ABSP",
    fullName: "Association of British Scrabble Players",
    description:
      "ABSP is the UK Scrabble organization. Connection requires identity verification with a photo ID.",
    requiresCredentials: false,
    requiresVerification: true,
  },
};

export const Integrations = () => {
  const { loginState } = useLoginStateStoreContext();
  const [integrations, setIntegrations] = useState<Integration[]>([]);
  const [organizations, setOrganizations] = useState<OrganizationTitle[]>([]);
  const [connectingOrg, setConnectingOrg] = useState<string | null>(null);
  const [naspaForm] = Form.useForm();
  const [verificationForm] = Form.useForm();
  const [uploadedFile, setUploadedFile] = useState<File | null>(null);
  const [loading, setLoading] = useState(false);
  const formRef = useRef<HTMLDivElement>(null);

  const integrationsClient = useClient(IntegrationService);
  const orgClient = useClient(OrganizationService);

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
    const fetchOrganizations = async () => {
      try {
        const response = await orgClient.getMyOrganizations(
          create(GetMyOrganizationsRequestSchema, {}),
        );
        setOrganizations(response.titles);
      } catch (e) {
        console.error(e);
      }
    };
    fetchIntegrations();
    fetchOrganizations();
  }, [integrationsClient, orgClient, loginState.loggedIn]);

  // Update the verification form when connectingOrg changes
  useEffect(() => {
    if (connectingOrg === "wespa" || connectingOrg === "absp") {
      verificationForm.setFieldsValue({ organizationCode: connectingOrg });
    }
  }, [connectingOrg, verificationForm]);

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

  const handleConnectNASPA = async (values: {
    memberId: string;
    password: string;
  }) => {
    setLoading(true);
    try {
      await orgClient.connectOrganization(
        create(ConnectOrganizationRequestSchema, {
          organizationCode: "naspa",
          memberId: values.memberId,
          credentials: {
            username: values.memberId, // NASPA username is the same as member ID
            password: values.password,
          },
        }),
      );
      message.success("Successfully connected to NASPA!");
      naspaForm.resetFields();
      setConnectingOrg(null);
      // Refresh organizations
      const response = await orgClient.getMyOrganizations(
        create(GetMyOrganizationsRequestSchema, {}),
      );
      setOrganizations(response.titles);
    } catch (e) {
      flashError(e);
    } finally {
      setLoading(false);
    }
  };

  const handleSubmitVerification = async (values: {
    organizationCode: string;
    memberId: string;
  }) => {
    if (!uploadedFile) {
      message.error("Please upload an ID photo");
      return;
    }

    setLoading(true);
    try {
      const reader = new FileReader();

      // Read the file as an ArrayBuffer (binary data)
      const arrayBuffer = await new Promise<ArrayBuffer>((resolve, reject) => {
        reader.onload = () => resolve(reader.result as ArrayBuffer);
        reader.onerror = reject;
        reader.readAsArrayBuffer(uploadedFile);
      });

      // Convert ArrayBuffer to Uint8Array for protobuf
      const imageBytes = new Uint8Array(arrayBuffer);
      const fileExt = uploadedFile.name.split(".").pop() || "jpg";

      await orgClient.submitVerification(
        create(SubmitVerificationRequestSchema, {
          organizationCode: values.organizationCode,
          memberId: values.memberId,
          imageData: imageBytes,
          imageExtension: `.${fileExt}`,
        }),
      );

      verificationForm.resetFields();
      setUploadedFile(null);
      setConnectingOrg(null);

      // Refresh organizations
      const response = await orgClient.getMyOrganizations(
        create(GetMyOrganizationsRequestSchema, {}),
      );
      setOrganizations(response.titles);

      // Show success modal
      Modal.success({
        title: (
          <div className="readable-text-color">
            Verification Request Submitted
          </div>
        ),
        content: (
          <div className="readable-text-color">
            <p>
              Your verification request has been submitted successfully. Please
              wait for a site administrator to review and approve your request.
            </p>
          </div>
        ),
      });
    } catch (e) {
      flashError(e);
    } finally {
      setLoading(false);
    }
  };

  const handleDisconnect = async (organizationCode: string) => {
    try {
      await orgClient.disconnectOrganization(
        create(DisconnectOrganizationRequestSchema, {
          organizationCode,
        }),
      );
      message.success("Organization disconnected");
      // Refresh organizations
      const response = await orgClient.getMyOrganizations(
        create(GetMyOrganizationsRequestSchema, {}),
      );
      setOrganizations(response.titles);
    } catch (e) {
      flashError(e);
    }
  };

  const handleRefreshTitles = async () => {
    setLoading(true);
    try {
      const response = await orgClient.refreshTitles(
        create(RefreshTitlesRequestSchema, {}),
      );
      message.success(response.message || "Titles refreshed successfully");
      setOrganizations(response.titles);
    } catch (e) {
      flashError(e);
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <h3>Your integrations</h3>

      <h4 style={{ marginTop: "2rem" }}>Third-party Services</h4>
      <p>Connect with the following apps:</p>

      <Flex gap="small" style={{ marginBottom: "2rem", marginTop: "1rem" }}>
        {typedKeys(apps).map((app) => (
          <Tooltip title={apps[app].name} key={app}>
            {apps[app].button}
          </Tooltip>
        ))}
      </Flex>

      {integrations.filter((i) => i.integrationName in apps).length ? (
        <p>You have the following connected apps:</p>
      ) : null}
      <Flex gap="small" style={{ marginTop: "1rem" }}>
        {integrations
          .filter((integration) => integration.integrationName in apps)
          .map((integration) => {
            const appName =
              apps[integration.integrationName as keyof typeof apps].name;
            let info =
              apps[integration.integrationName as keyof typeof apps]
                .information;
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
                    {
                      apps[integration.integrationName as keyof typeof apps]
                        .logo
                    }
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

      <Divider />

      <h4>Organization Memberships</h4>
      <p>
        Connect your account to official organizations to display your titles
        and verify your identity.
      </p>

      {organizations.length > 0 && (
        <div style={{ marginBottom: "2rem", marginTop: "2rem" }}>
          <Button
            icon={<ReloadOutlined />}
            onClick={handleRefreshTitles}
            loading={loading}
            style={{ marginBottom: 16 }}
          >
            Refresh All Titles
          </Button>
          <Table
            dataSource={organizations}
            rowKey={(record) => record.organizationCode}
            columns={[
              {
                title: "Organization",
                dataIndex: "organizationCode",
                key: "organizationCode",
                render: (code: string) =>
                  organizationInfo[code as keyof typeof organizationInfo]
                    ?.name || code,
              },
              {
                title: "Member ID",
                dataIndex: "memberId",
                key: "memberId",
              },
              {
                title: "Name",
                dataIndex: "fullName",
                key: "fullName",
              },
              {
                title: "Title",
                dataIndex: "normalizedTitle",
                key: "normalizedTitle",
                render: (title: string) =>
                  title ? <Tag color="blue">{title}</Tag> : "-",
              },
              {
                title: "Status",
                dataIndex: "verified",
                key: "verified",
                render: (verified: boolean) => (
                  <Tag color={verified ? "green" : "orange"}>
                    {verified ? "Verified" : "Pending"}
                  </Tag>
                ),
              },
              {
                title: "Action",
                key: "action",
                render: (_: unknown, record: OrganizationTitle) => (
                  <Popconfirm
                    title="Disconnect from this organization?"
                    description={`Are you sure you want to disconnect from ${organizationInfo[record.organizationCode as keyof typeof organizationInfo]?.name || record.organizationCode}?`}
                    okText="Yes"
                    cancelText="No"
                    onConfirm={() => handleDisconnect(record.organizationCode)}
                  >
                    <Button danger size="small" icon={<DeleteOutlined />}>
                      Disconnect
                    </Button>
                  </Popconfirm>
                ),
              },
            ]}
            pagination={false}
            size="small"
          />
        </div>
      )}

      <h5 style={{ marginTop: "2rem" }}>Connect to an Organization</h5>

      <Flex gap="middle" wrap="wrap" style={{ marginBottom: "2rem" }}>
        {Object.entries(organizationInfo).map(([code, info]) => {
          if (organizations.some((org) => org.organizationCode === code)) {
            return null;
          }

          return (
            <Card
              key={code}
              title={info.name}
              style={{ width: 350 }}
              extra={
                <Button
                  type="primary"
                  size="small"
                  onClick={() => {
                    setConnectingOrg(code);
                    setTimeout(() => {
                      formRef.current?.scrollIntoView({
                        behavior: "smooth",
                        block: "start",
                      });
                    }, 100);
                  }}
                >
                  Connect
                </Button>
              }
            >
              <strong>{info.fullName}</strong>
              <p style={{ marginTop: "0.5rem" }}>{info.description}</p>
            </Card>
          );
        })}
      </Flex>

      <div ref={formRef}>
        {connectingOrg === "naspa" && (
          <Card
            title="Connect to NASPA"
            style={{ marginTop: "2rem", maxWidth: 500 }}
            extra={
              <Button size="small" onClick={() => setConnectingOrg(null)}>
                Cancel
              </Button>
            }
          >
            <Form
              form={naspaForm}
              onFinish={handleConnectNASPA}
              layout="vertical"
            >
              <Form.Item
                label="Member ID"
                name="memberId"
                rules={[
                  {
                    required: true,
                    message: "Please enter your NASPA member ID",
                  },
                ]}
                help="Your NASPA member ID (e.g., AA000083)"
              >
                <Input placeholder="AA000083" />
              </Form.Item>
              <Form.Item
                label="Password"
                name="password"
                rules={[
                  {
                    required: true,
                    message: "Please enter your NASPA password",
                  },
                ]}
              >
                <Input.Password />
              </Form.Item>
              <Form.Item>
                <Space>
                  <Button type="primary" htmlType="submit" loading={loading}>
                    Connect
                  </Button>
                  <Button onClick={() => setConnectingOrg(null)}>Cancel</Button>
                </Space>
              </Form.Item>
            </Form>
          </Card>
        )}

        {(connectingOrg === "wespa" || connectingOrg === "absp") && (
          <Card
            title={`Connect to ${organizationInfo[connectingOrg as keyof typeof organizationInfo].name}`}
            style={{ marginTop: "2rem", maxWidth: 500 }}
            extra={
              <Button size="small" onClick={() => setConnectingOrg(null)}>
                Cancel
              </Button>
            }
          >
            <p>
              This organization requires identity verification. Please upload a
              selfie of you holding your government-issued ID. The photo will
              only be accessible to our site administrators for verification
              purposes and will be deleted immediately after verification is
              complete.
            </p>
            <Form
              form={verificationForm}
              onFinish={handleSubmitVerification}
              layout="vertical"
              initialValues={{ organizationCode: connectingOrg }}
            >
              <Form.Item name="organizationCode" hidden>
                <Input />
              </Form.Item>
              <Form.Item
                label={
                  <span>
                    Member ID{" "}
                    <Tooltip
                      title={
                        connectingOrg === "wespa" ? (
                          <span>
                            Visit{" "}
                            <a
                              href="https://wespa.org/ratings.shtml"
                              target="_blank"
                              rel="noopener noreferrer"
                              style={{ color: "#1890ff" }}
                            >
                              wespa.org/ratings.shtml
                            </a>
                            , scroll down to "Find a player", enter your name,
                            and click Submit. Your player ID is the number in
                            the URL (e.g., <strong>2145</strong> from
                            wespa.org/aardvark/html/players/2145.html).
                          </span>
                        ) : (
                          <span>
                            Visit{" "}
                            <a
                              href="https://absp-database.org/ratings"
                              target="_blank"
                              rel="noopener noreferrer"
                              style={{ color: "#1890ff" }}
                            >
                              absp-database.org/ratings
                            </a>
                            , find your name in the table. Your member ID is in
                            the <strong>MemNo</strong> column, immediately to
                            the right of your rating.
                          </span>
                        )
                      }
                      overlayStyle={{ maxWidth: 400 }}
                    >
                      <QuestionCircleOutlined style={{ cursor: "help" }} />
                    </Tooltip>
                  </span>
                }
                name="memberId"
                rules={[
                  {
                    required: true,
                    message: `Please enter your ${organizationInfo[connectingOrg as keyof typeof organizationInfo].name} member ID`,
                  },
                ]}
              >
                <Input
                  placeholder={
                    connectingOrg === "wespa" ? "e.g., 2145" : "e.g., 745"
                  }
                />
              </Form.Item>
              <Form.Item
                label="ID Photo"
                required
                help="Upload a clear selfie of you holding your government-issued ID"
              >
                <Upload
                  beforeUpload={(file) => {
                    const maxSize = 4 * 1024 * 1024; // 4MB
                    if (file.size > maxSize) {
                      message.error(
                        "Image must be smaller than 4MB. Please compress or resize your image.",
                      );
                      return Upload.LIST_IGNORE;
                    }
                    setUploadedFile(file);
                    return false;
                  }}
                  onRemove={() => setUploadedFile(null)}
                  maxCount={1}
                  accept="image/*"
                >
                  <Button icon={<UploadOutlined />}>Select Photo</Button>
                </Upload>
              </Form.Item>
              <Form.Item>
                <Space>
                  <Button type="primary" htmlType="submit" loading={loading}>
                    Submit Verification
                  </Button>
                  <Button onClick={() => setConnectingOrg(null)}>Cancel</Button>
                </Space>
              </Form.Item>
            </Form>
          </Card>
        )}
      </div>
    </>
  );
};
