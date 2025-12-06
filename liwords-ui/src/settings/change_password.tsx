import React, { useState } from "react";
import { Button, Input, Form, Row, Col, App } from "antd";
import { Link } from "react-router";
import { connectErrorMessage, useClient } from "../utils/hooks/connect";
import { AuthenticationService } from "../gen/api/proto/user_service/user_service_pb";

const layout = {
  labelCol: {
    span: 24,
  },
  wrapperCol: {
    span: 24,
  },
};

export const ChangePassword = React.memo(() => {
  const [err, setErr] = useState("");
  const [form] = Form.useForm();
  const { notification } = App.useApp();

  const authClient = useClient(AuthenticationService);

  const onFinish = async (values: { [key: string]: string }) => {
    try {
      await authClient.changePassword({
        oldPassword: values.oldPassword,
        newPassword: values.newPassword,
      });
      notification.info({
        message: "Success",
        description: "Your password was changed.",
      });
      setErr("");
    } catch (err) {
      setErr(connectErrorMessage(err));
      form.validateFields();
    }
  };

  return (
    <div className="change-password">
      <h3>Change password</h3>
      <Form
        form={form}
        {...layout}
        name="changepassword"
        onFinish={onFinish}
        onValuesChange={(changedValues, allValues) => {
          setErr("");
        }}
        style={{ marginTop: 20 }}
        requiredMark={false}
      >
        <Row>
          <Col span={11}>
            <Form.Item
              label="Old password"
              name="oldPassword"
              rules={[
                {
                  validator: async (_, oldPassword) => {
                    if (err !== "") {
                      return Promise.reject(new Error(err));
                    }
                  },
                },
                {
                  required: true,
                  message: "Please input your old password",
                },
              ]}
            >
              <Input.Password size="large" />
            </Form.Item>
          </Col>
          <Col span={1} />
          <Col span={11}>
            <Form.Item
              label="New password"
              name="newPassword"
              rules={[
                {
                  required: true,
                  message: "Please input your new password",
                },
              ]}
            >
              <Input.Password size="large" />
            </Form.Item>
          </Col>
        </Row>
        <Row>
          <Col span={12} className="button-row">
            <Link to="/password/reset">
              I’m drawing a blank on my password. Help!
            </Link>
          </Col>
          <Col span={11} className="button-row">
            <Form.Item>
              <Button className="save" type="primary" htmlType="submit">
                Save
              </Button>
            </Form.Item>
          </Col>
        </Row>
      </Form>
    </div>
  );
});
