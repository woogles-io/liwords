import React, { useState } from "react";
import { Row, Col, Input, Form, Alert, App, Button } from "antd";
import { TopBar } from "../navigation/topbar";
import { connectErrorMessage, useClient } from "../utils/hooks/connect";
import { AuthenticationService } from "../gen/api/proto/user_service/user_service_pb";

const layout = {
  labelCol: {
    span: 8,
  },
  wrapperCol: {
    span: 12,
  },
};
const tailLayout = {
  wrapperCol: {
    offset: 8,
    span: 12,
  },
};

export const PasswordReset = () => {
  const [err, setErr] = useState("");
  const authClient = useClient(AuthenticationService);
  const { notification } = App.useApp();

  const onFinish = async (values: { [key: string]: string }) => {
    try {
      await authClient.resetPasswordStep1({ email: values.email });
      notification.info({
        message: "Sent",
        description: "Please check your email; a reset code was sent.",
      });
    } catch (e) {
      setErr(connectErrorMessage(e));
    }
  };

  return (
    <>
      <Row>
        <Col span={24}>
          <TopBar />
          <Form
            {...layout}
            name="resetpassword"
            onFinish={onFinish}
            style={{ marginTop: 20 }}
          >
            <Form.Item
              label="Email address"
              name="email"
              rules={[
                {
                  required: true,
                  message: "Please input your email address!",
                },
              ]}
            >
              <Input />
            </Form.Item>

            <Form.Item {...tailLayout}>
              <Button type="primary" htmlType="submit">
                Submit
              </Button>
            </Form.Item>
          </Form>
        </Col>
      </Row>

      {err !== "" ? <Alert message={err} type="error" /> : null}
    </>
  );
};
