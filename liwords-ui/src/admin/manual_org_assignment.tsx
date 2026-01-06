import React, { useState, useCallback } from "react";
import {
  Card,
  Input,
  Button,
  Form,
  Select,
  Space,
  Table,
  Tag,
  message,
  Tooltip,
  Alert,
  AutoComplete,
  Popconfirm,
} from "antd";
import {
  UserAddOutlined,
  DeleteOutlined,
  QuestionCircleOutlined,
  ReloadOutlined,
} from "@ant-design/icons";
import { useClient, flashError } from "../utils/hooks/connect";
import {
  OrganizationService,
  ManuallySetOrgMembershipRequestSchema,
  GetPublicOrganizationsRequestSchema,
  DisconnectOrganizationRequestSchema,
  AdminRefreshUserTitlesRequestSchema,
  OrganizationTitle,
  AutocompleteService,
} from "../gen/api/proto/user_service/user_service_pb";
import { create } from "@bufbuild/protobuf";
import { useDebounce } from "../utils/debounce";

const organizationInfo = {
  naspa: {
    name: "NASPA",
    fullName: "North American SCRABBLE Players Association",
    help: "Enter the NASPA member ID (e.g., AA000083)",
    helpTooltip: "NASPA member IDs are in the format AA000083",
  },
  wespa: {
    name: "WESPA",
    fullName: "World English-Language Scrabble Players Association",
    help: "Enter the WESPA player ID",
    helpTooltip: (
      <span>
        Visit{" "}
        <a
          href="https://legacy.wespa.org/ratings.shtml"
          target="_blank"
          rel="noopener noreferrer"
          style={{ color: "#1890ff" }}
        >
          legacy.wespa.org/ratings.shtml
        </a>
        , scroll down to "Find a player", enter the name, and click Submit. The
        player ID is the number in the URL (e.g., <strong>2145</strong> from
        legacy.wespa.org/aardvark/html/players/2145.html).
      </span>
    ),
  },
  absp: {
    name: "ABSP",
    fullName: "Association of British Scrabble Players",
    help: "Enter the ABSP member number",
    helpTooltip: (
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
        , find the name in the table. The member ID is in the{" "}
        <strong>MemNo</strong> column, immediately to the right of the rating.
      </span>
    ),
  },
};

type UserSearchResult = {
  uuid: string;
  username: string;
};

export const ManualOrgAssignment = () => {
  const [searchUsername, setSearchUsername] = useState("");
  const [currentUser, setCurrentUser] = useState<string | null>(null);
  const [organizations, setOrganizations] = useState<OrganizationTitle[]>([]);
  const [loading, setLoading] = useState(false);
  const [assignForm] = Form.useForm();
  const [usernameOptions, setUsernameOptions] = useState<UserSearchResult[]>(
    [],
  );

  const orgClient = useClient(OrganizationService);
  const acClient = useClient(AutocompleteService);

  const onUsernameSearch = useCallback(
    async (searchQuery: string) => {
      if (!searchQuery || searchQuery.length < 2) {
        setUsernameOptions([]);
        return;
      }

      try {
        const response = await acClient.getCompletion({
          prefix: searchQuery,
        });
        const users: UserSearchResult[] = (response.users || [])
          .map((user) => ({
            uuid: user.uuid,
            username: user.username,
          }))
          .sort((a, b) =>
            a.username.toLowerCase().localeCompare(b.username.toLowerCase()),
          );
        setUsernameOptions(users);
      } catch (error) {
        console.error("Error searching usernames:", error);
        setUsernameOptions([]);
      }
    },
    [acClient],
  );

  const searchUsernameDebounced = useDebounce(onUsernameSearch, 300);

  const handleUsernameSelect = useCallback((data: string) => {
    // Extract just the username from "username" format
    setSearchUsername(data);
  }, []);

  const handleSearch = async () => {
    if (!searchUsername.trim()) {
      message.error("Please enter a username");
      return;
    }

    setLoading(true);
    try {
      const response = await orgClient.getPublicOrganizations(
        create(GetPublicOrganizationsRequestSchema, {
          username: searchUsername.trim(),
        }),
      );
      setOrganizations(response.titles);
      setCurrentUser(searchUsername.trim());
      assignForm.resetFields();
    } catch (e) {
      flashError(e);
      setCurrentUser(null);
      setOrganizations([]);
    } finally {
      setLoading(false);
    }
  };

  const handleAssign = async (values: {
    organizationCode: string;
    memberId: string;
    username?: string;
    password?: string;
  }) => {
    if (!currentUser) {
      message.error("Please search for a user first");
      return;
    }

    // Build credentials map if username and password are both provided
    const credentials: { [key: string]: string } = {};
    if (values.username && values.password) {
      credentials.username = values.username;
      credentials.password = values.password;
    }

    setLoading(true);
    try {
      const response = await orgClient.manuallySetOrgMembership(
        create(ManuallySetOrgMembershipRequestSchema, {
          username: currentUser,
          organizationCode: values.organizationCode,
          memberId: values.memberId,
          credentials:
            Object.keys(credentials).length > 0 ? credentials : undefined,
        }),
      );

      if (response.success) {
        message.success(
          response.message || "Organization membership assigned successfully",
        );
        assignForm.resetFields();
        // Refresh the organizations list
        handleSearch();
      } else {
        message.error(response.message || "Failed to assign membership");
      }
    } catch (e) {
      flashError(e);
    } finally {
      setLoading(false);
    }
  };

  const handleDisconnect = async (organizationCode: string) => {
    if (!currentUser) return;

    setLoading(true);
    try {
      await orgClient.disconnectOrganization(
        create(DisconnectOrganizationRequestSchema, {
          organizationCode,
          username: currentUser,
        }),
      );
      message.success("Organization membership removed");
      // Refresh the organizations list
      handleSearch();
    } catch (e) {
      flashError(e);
    } finally {
      setLoading(false);
    }
  };

  const handleRefreshTitles = async () => {
    if (!currentUser) return;

    setLoading(true);
    try {
      const response = await orgClient.adminRefreshUserTitles(
        create(AdminRefreshUserTitlesRequestSchema, {
          username: currentUser,
        }),
      );
      message.success(response.message);
      // Refresh the organizations list to show updated titles
      handleSearch();
    } catch (e) {
      flashError(e);
    } finally {
      setLoading(false);
    }
  };

  const columns = [
    {
      title: "Organization",
      dataIndex: "organizationCode",
      key: "organizationCode",
      render: (code: string) =>
        organizationInfo[code as keyof typeof organizationInfo]?.name || code,
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
          title="Remove this organization membership?"
          description={`Are you sure you want to remove this user's ${organizationInfo[record.organizationCode as keyof typeof organizationInfo]?.name || record.organizationCode} membership?`}
          okText="Yes"
          cancelText="No"
          onConfirm={() => handleDisconnect(record.organizationCode)}
        >
          <Button danger size="small" icon={<DeleteOutlined />}>
            Remove
          </Button>
        </Popconfirm>
      ),
    },
  ];

  return (
    <div style={{ padding: "20px" }}>
      <Card title="Manual Organization Assignment" style={{ marginBottom: 20 }}>
        <Alert
          message="Admin Tool"
          description="Manually assign organization memberships to users. The user's name and title will be automatically fetched from the organization's database."
          type="info"
          showIcon
          style={{ marginBottom: 20 }}
        />

        <Space.Compact style={{ width: "100%", maxWidth: 500 }}>
          <AutoComplete
            value={searchUsername}
            placeholder="Search username..."
            onSearch={searchUsernameDebounced}
            onSelect={handleUsernameSelect}
            onChange={(value) => setSearchUsername(value)}
            style={{ width: "100%" }}
          >
            {usernameOptions.map((user) => (
              <AutoComplete.Option key={user.uuid} value={user.username}>
                {user.username}
              </AutoComplete.Option>
            ))}
          </AutoComplete>
          <Button type="primary" onClick={handleSearch} loading={loading}>
            Search
          </Button>
        </Space.Compact>
      </Card>

      {currentUser && (
        <>
          <Card
            title={`Current Memberships for ${currentUser}`}
            style={{ marginBottom: 20 }}
            extra={
              <Button
                type="default"
                icon={<ReloadOutlined />}
                onClick={handleRefreshTitles}
                loading={loading}
                disabled={organizations.length === 0}
              >
                Refresh Titles
              </Button>
            }
          >
            {organizations.length > 0 ? (
              <Table
                dataSource={organizations}
                columns={columns}
                rowKey={(record) => record.organizationCode}
                pagination={false}
                size="small"
              />
            ) : (
              <p style={{ color: "#999" }}>
                No organization memberships found for this user.
              </p>
            )}
          </Card>

          <Card title="Assign New Membership">
            <Form
              form={assignForm}
              onFinish={handleAssign}
              layout="vertical"
              style={{ maxWidth: 600 }}
            >
              <Form.Item
                label="Organization"
                name="organizationCode"
                rules={[
                  { required: true, message: "Please select an organization" },
                ]}
              >
                <Select placeholder="Select organization">
                  {Object.entries(organizationInfo)
                    .filter(
                      ([code]) =>
                        !organizations.some(
                          (org) => org.organizationCode === code,
                        ),
                    )
                    .map(([code, info]) => (
                      <Select.Option key={code} value={code}>
                        {info.name} - {info.fullName}
                      </Select.Option>
                    ))}
                </Select>
              </Form.Item>

              <Form.Item noStyle shouldUpdate>
                {({ getFieldValue }) => {
                  const selectedOrg = getFieldValue("organizationCode");
                  const orgInfo =
                    organizationInfo[
                      selectedOrg as keyof typeof organizationInfo
                    ];
                  const supportsCredentials =
                    selectedOrg === "naspa" || selectedOrg === "absp";

                  return selectedOrg && orgInfo ? (
                    <>
                      <Form.Item
                        label={
                          <span>
                            Member ID{" "}
                            <Tooltip
                              title={orgInfo.helpTooltip}
                              overlayStyle={{ maxWidth: 400 }}
                            >
                              <QuestionCircleOutlined
                                style={{ cursor: "help" }}
                              />
                            </Tooltip>
                          </span>
                        }
                        name="memberId"
                        rules={[
                          {
                            required: true,
                            message: "Please enter the member ID",
                          },
                        ]}
                        help={orgInfo.help}
                      >
                        <Input
                          placeholder={
                            selectedOrg === "naspa"
                              ? "AA000083"
                              : selectedOrg === "wespa"
                                ? "2145"
                                : "745"
                          }
                        />
                      </Form.Item>

                      {supportsCredentials && (
                        <>
                          <Alert
                            message="Optional: User Credentials"
                            description="If you have the user's login credentials, you can enter them below. This allows automatic title refreshes in the future. If left blank, the public database will be used."
                            type="info"
                            showIcon
                            style={{ marginBottom: 16 }}
                          />
                          <Form.Item
                            label={
                              selectedOrg === "naspa"
                                ? "NASPA Username (Member ID)"
                                : "ABSP Username (Email)"
                            }
                            name="username"
                          >
                            <Input
                              placeholder={
                                selectedOrg === "naspa"
                                  ? "AA000083"
                                  : "user@example.com"
                              }
                            />
                          </Form.Item>
                          <Form.Item label="Password" name="password">
                            <Input.Password />
                          </Form.Item>
                        </>
                      )}
                    </>
                  ) : null;
                }}
              </Form.Item>

              <Form.Item>
                <Space>
                  <Button
                    type="primary"
                    htmlType="submit"
                    icon={<UserAddOutlined />}
                    loading={loading}
                  >
                    Assign Membership
                  </Button>
                  <Button onClick={() => assignForm.resetFields()}>
                    Clear
                  </Button>
                </Space>
              </Form.Item>
            </Form>

            <Alert
              message="Note"
              description="When you assign a membership, the system will automatically fetch the user's name and title from the organization's database. The membership will be marked as verified."
              type="warning"
              showIcon
              style={{ marginTop: 20 }}
            />
          </Card>
        </>
      )}
    </div>
  );
};
