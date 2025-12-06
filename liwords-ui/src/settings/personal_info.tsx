import React, { useCallback, useState } from "react";
import {
  Alert,
  Button,
  Col,
  Form,
  Input,
  Upload,
  Row,
  Select,
  App,
  DatePicker,
} from "antd";
import { Modal } from "../utils/focus_modal";
import moment from "moment";
import { PlayerAvatar } from "../shared/player_avatar";
import { AvatarRemoveModal } from "./avatar_remove_modal";
import { countryArray } from "./country_map";
import { MarkdownTips } from "./markdown_tips";
import { AvatarCropper } from "./avatar_cropper";
import { UploadChangeParam } from "antd/lib/upload";
import { PersonalInfoResponse } from "../gen/api/proto/user_service/user_service_pb";
import { PlayerInfo } from "../gen/api/proto/ipc/omgwords_pb";
import {
  connectErrorMessage,
  flashError,
  useClient,
} from "../utils/hooks/connect";
import { ProfileService } from "../gen/api/proto/user_service/user_service_pb";

type Props = {
  player: Partial<PlayerInfo> | undefined;
  personalInfo: PersonalInfoResponse;
  updatedAvatar: (avatarUrl: string) => void;
  startClosingAccount: () => void;
};

export const PersonalInfoWidget = React.memo((props: Props) => {
  const { TextArea } = Input;
  const { notification } = App.useApp();

  const [removeAvatarModalVisible, setRemoveAvatarModalVisible] =
    useState(false);
  const [bioTipsModalVisible, setBioTipsModalVisible] = useState(false);
  const [avatarErr, setAvatarErr] = useState("");
  const [uploadPending, setUploadPending] = useState(false);
  const [cropperOpen, setCropperOpen] = useState(false);
  const [imageToUpload, setImageToUpload] = useState<Blob | undefined>(
    undefined,
  );

  const avatarErrorCatcher = useCallback((e: unknown) => {
    console.log(e);
    setAvatarErr(connectErrorMessage(e));
    setUploadPending(false);
  }, []);
  const propsUpdatedAvatar = props.updatedAvatar;
  const fileProps = {
    beforeUpload: (file: File) => {
      return false;
    },
    maxCount: 1,
    onChange: (info: UploadChangeParam) => {
      if (info.fileList.length > 0) {
        // If they try again, the new one goes to the end of the list
        setImageToUpload(info.fileList[info.fileList.length - 1].originFileObj);
        setCropperOpen(true);
      }
    },
    accept: "image/*",
    showUploadList: false,
  };

  const cancelRemoveAvatarModal = useCallback(() => {
    setRemoveAvatarModalVisible(false);
  }, []);

  const profileClient = useClient(ProfileService);
  const removeAvatar = useCallback(async () => {
    try {
      await profileClient.removeAvatar({});
      notification.info({
        message: "Success",
        description: "Your avatar was removed.",
      });
      setRemoveAvatarModalVisible(false);
      propsUpdatedAvatar("");
    } catch (e) {
      avatarErrorCatcher(e);
    }
  }, [propsUpdatedAvatar, avatarErrorCatcher, profileClient, notification]);

  const saveAvatar = useCallback(
    async (imageDataUrl: string) => {
      let jpegUint8 = new Uint8Array();

      try {
        const b = await fetch(imageDataUrl);
        const buff = await b.arrayBuffer();
        jpegUint8 = new Uint8Array(buff);
      } catch (e) {
        avatarErrorCatcher(e);
      }

      setUploadPending(true);
      try {
        const resp = await profileClient.updateAvatar({ jpgData: jpegUint8 });
        notification.info({
          message: "Success",
          description: "Your avatar was updated.",
        });
        setUploadPending(false);
        propsUpdatedAvatar(resp.avatarUrl);
      } catch (e) {
        avatarErrorCatcher(e);
      }
    },
    [propsUpdatedAvatar, avatarErrorCatcher, profileClient, notification],
  );

  const updateFields = async (values: { [key: string]: string }) => {
    const birthDate = values.birthDate
      ? moment(values.birthDate).format("YYYY-MM-DD")
      : "";
    try {
      await profileClient.updatePersonalInfo({
        ...values,
        birthDate,
      });
      notification.info({
        message: "Success",
        description: "Your personal info was changed.",
      });
    } catch (e) {
      flashError(e);
    }
  };

  const countrySelector = (
    <Select size="large" bordered={false}>
      {countryArray.map((country) => {
        return (
          <Select.Option key={country.code} value={country.code.toLowerCase()}>
            {country.emoji} {country.name}
          </Select.Option>
        );
      })}
    </Select>
  );

  const [form] = Form.useForm();

  const layout = {
    labelCol: {
      span: 24,
    },
    wrapperCol: {
      span: 24,
    },
  };

  const bioTipsModal = (
    <Modal
      className="bio-tips-modal"
      title="Tips for editing your bio"
      width="60%"
      open={bioTipsModalVisible}
      onCancel={() => {
        setBioTipsModalVisible(false);
      }}
      footer={[
        <Button
          key="cheat-sheet"
          onClick={() => {
            const newWindow = window.open(
              "https://www.markdownguide.org/cheat-sheet/",
              "_blank",
              "noopener,noreferrer",
            );
            if (newWindow) newWindow.opener = null;
          }}
        >
          See full guide
        </Button>,
        <Button
          key="ok"
          onClick={() => {
            setBioTipsModalVisible(false);
          }}
        >
          OK
        </Button>,
      ]}
    >
      <MarkdownTips />
    </Modal>
  );
  return (
    <Form
      form={form}
      {...layout}
      className="personal-info"
      onFinish={updateFields}
      initialValues={{
        ...props.personalInfo,
        birthDate: props.personalInfo.birthDate
          ? moment(props.personalInfo.birthDate, "YYYY-MM-DD")
          : null,
      }}
    >
      <h3>Personal info</h3>
      <div className="section-header">Profile picture</div>
      {props.personalInfo?.avatarUrl !== "" ? (
        <div className="avatar-section">
          <PlayerAvatar
            player={props.player}
            avatarUrl={props.personalInfo.avatarUrl}
          />
          <Upload {...fileProps}>
            <Button className="change-avatar">
              {uploadPending ? "Uploading..." : "Change"}
            </Button>
          </Upload>
          <Button
            className="remove-avatar"
            onClick={() => setRemoveAvatarModalVisible(true)}
            type="link"
          >
            Remove
          </Button>
        </div>
      ) : (
        <div className="no-avatar-section">
          {" "}
          <Upload {...fileProps}>
            <Button className="change-avatar" disabled={uploadPending}>
              {uploadPending ? "Uploading..." : "Add a profile photo"}
            </Button>
          </Upload>
        </div>
      )}
      {avatarErr !== "" ? <Alert message={avatarErr} type="error" /> : null}
      {bioTipsModal}
      <AvatarRemoveModal
        visible={removeAvatarModalVisible}
        error={avatarErr}
        onOk={removeAvatar}
        onCancel={cancelRemoveAvatarModal}
      />
      {cropperOpen && (
        <AvatarCropper
          file={imageToUpload}
          onCancel={() => {
            setCropperOpen(false);
          }}
          onError={(errorMessage) => {
            setAvatarErr(errorMessage);
            setCropperOpen(false);
          }}
          onSave={(imageDataUrl) => {
            setCropperOpen(false);
            saveAvatar(imageDataUrl);
          }}
        />
      )}
      <Row>
        <Col span={23}>
          <div className="section-header bio-section-header">
            Player bio
            <span
              className="bio-tips"
              onClick={() => {
                setBioTipsModalVisible(true);
              }}
            >
              Tips
            </span>
          </div>
          <Form.Item name="about">
            <TextArea className="bio-editor" rows={4} />
          </Form.Item>
        </Col>
      </Row>

      <div className="section-header">Account details</div>
      <Row>
        <Col span={23}>
          <Form.Item
            name="birthDate"
            label={
              <>
                Date of birth{" "}
                <span className="notice">(This will not be displayed.)</span>
              </>
            }
            rules={[
              {
                required: true,
                message:
                  "Your profile information will be private unless you provide a birthdate.",
              },
            ]}
          >
            <DatePicker
              format={"YYYY-MM-DD"}
              placeholder="YYYY-MM-DD"
              showToday={false}
            />
          </Form.Item>
        </Col>
      </Row>
      <Row>
        <Col span={11}>
          <Form.Item
            name="email"
            label="Email"
            rules={[
              {
                required: true,
                type: "email",
                message: "Enter a valid email address",
              },
            ]}
          >
            <Input size="large" />
          </Form.Item>
        </Col>
        <Col span={1} />
        <Col span={11}>
          <Form.Item name="firstName" label="First name">
            <Input size="large" />
          </Form.Item>
        </Col>
      </Row>
      <Row>
        <Col span={11}>
          <Form.Item name="lastName" label="Last name">
            <Input size="large" />
          </Form.Item>
        </Col>
        <Col span={1} />
        <Col span={11}>
          <Form.Item name="countryCode" label="Country">
            {countrySelector}
          </Form.Item>
        </Col>
      </Row>
      <Row align="middle">
        <Col
          span={8}
          className="close-account-button"
          onClick={() => {
            props.startClosingAccount();
          }}
        >
          Delete my account
        </Col>
        <Col span={16}>
          <Form.Item>
            <Button className="save-button" type="primary" htmlType="submit">
              Save
            </Button>
          </Form.Item>
        </Col>
      </Row>
    </Form>
  );
});
