import React, { useCallback } from 'react';
import {
  Alert,
  Button,
  Col,
  Form,
  Input,
  Upload,
  Row,
  Select,
  notification,
  DatePicker,
  Switch,
  Divider,
} from 'antd';
import { Modal } from '../utils/focus_modal';
import moment from 'moment';
import { PlayerAvatar } from '../shared/player_avatar';
import { PlayerMetadata } from '../gameroom/game_info';
import { useMountedState } from '../utils/mounted';
import { AvatarRemoveModal } from './avatar_remove_modal';
import axios, { AxiosError } from 'axios';
import { toAPIUrl } from '../api/api';
import { countryArray } from './country_map';
import { MarkdownTips } from './markdown_tips';
import { AvatarCropper } from './avatar_cropper';
import { UploadChangeParam } from 'antd/lib/upload';

export type PersonalInfo = {
  birthDate: string;
  email: string;
  firstName: string;
  lastName: string;
  countryCode: string;
  about: string;
  silentMode: boolean;
};

type Props = {
  player: Partial<PlayerMetadata> | undefined;
  personalInfo: PersonalInfo;
  updatedAvatar: (avatarUrl: string) => void;
  startClosingAccount: () => void;
};

const errorCatcher = (e: AxiosError) => {
  if (e.response) {
    notification.warning({
      message: 'Fetch Error',
      description: e.response.data.msg,
      duration: 4,
    });
  }
};

export const PersonalInfo = React.memo((props: Props) => {
  const { useState } = useMountedState();
  const { TextArea } = Input;

  const [removeAvatarModalVisible, setRemoveAvatarModalVisible] =
    useState(false);
  const [bioTipsModalVisible, setBioTipsModalVisible] = useState(false);
  const [avatarErr, setAvatarErr] = useState('');
  const [uploadPending, setUploadPending] = useState(false);
  const [cropperOpen, setCropperOpen] = useState(false);
  const [imageToUpload, setImageToUpload] = useState<Blob | undefined>(
    undefined
  );

  const avatarErrorCatcher = useCallback((e: AxiosError) => {
    if (e.response) {
      // From Twirp
      console.log(e);
      setAvatarErr(e.response.data.msg);
      setUploadPending(false);
    } else {
      setAvatarErr('unknown error, see console');
      console.log(e);
      setUploadPending(false);
    }
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
    accept: 'image/*',
    showUploadList: false,
  };

  const cancelRemoveAvatarModal = useCallback(() => {
    setRemoveAvatarModalVisible(false);
  }, []);

  const removeAvatar = useCallback(() => {
    axios
      .post(
        toAPIUrl('user_service.ProfileService', 'RemoveAvatar'),
        {},
        {
          withCredentials: true,
        }
      )
      .then((resp) => {
        notification.info({
          message: 'Success',
          description: 'Your avatar was removed.',
        });
        setRemoveAvatarModalVisible(false);
        propsUpdatedAvatar('');
      })
      .catch(avatarErrorCatcher);
  }, [propsUpdatedAvatar, avatarErrorCatcher]);

  const saveAvatar = useCallback(
    (imageDataUrl: string) => {
      const jpegString = imageDataUrl.split(',')[1];
      setUploadPending(true);
      axios
        .post(
          toAPIUrl('user_service.ProfileService', 'UpdateAvatar'),
          {
            jpg_data: jpegString,
          },
          {
            withCredentials: true,
          }
        )
        .then((resp) => {
          notification.info({
            message: 'Success',
            description: 'Your avatar was updated.',
          });
          setUploadPending(false);
          propsUpdatedAvatar(resp.data.avatar_url);
        })
        .catch(avatarErrorCatcher);
    },
    [propsUpdatedAvatar, avatarErrorCatcher]
  );

  const updateFields = (values: { [key: string]: string }) => {
    const birthDate = values.birthDate
      ? moment(values.birthDate).format('YYYY-MM-DD')
      : '';
    axios
      .post(
        toAPIUrl('user_service.ProfileService', 'UpdatePersonalInfo'),
        {
          ...values,
          birthDate,
        },
        {
          withCredentials: true,
        }
      )
      .then(() => {
        notification.info({
          message: 'Success',
          description: 'Your personal info was changed.',
        });
      })
      .catch(errorCatcher);
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
      visible={bioTipsModalVisible}
      onCancel={() => {
        setBioTipsModalVisible(false);
      }}
      footer={[
        <Button
          key="cheat-sheet"
          onClick={() => {
            const newWindow = window.open(
              'https://www.markdownguide.org/cheat-sheet/',
              '_blank',
              'noopener,noreferrer'
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
          ? moment(props.personalInfo.birthDate, 'YYYY-MM-DD')
          : null,
      }}
    >
      <h3>Personal info</h3>
      <div className="section-header">Profile picture</div>
      {props.player?.avatar_url !== '' ? (
        <div className="avatar-section">
          <PlayerAvatar player={props.player} />
          <Upload {...fileProps}>
            <Button className="change-avatar">
              {uploadPending ? 'Uploading...' : 'Change'}
            </Button>
          </Upload>
          <Button
            className="remove-avatar"
            onClick={() => setRemoveAvatarModalVisible(true)}
          >
            Remove
          </Button>
        </div>
      ) : (
        <div className="no-avatar-section">
          {' '}
          <Upload {...fileProps}>
            <Button className="change-avatar" disabled={uploadPending}>
              {uploadPending ? 'Uploading...' : 'Add a profile photo'}
            </Button>
          </Upload>
        </div>
      )}
      {avatarErr !== '' ? <Alert message={avatarErr} type="error" /> : null}
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
                Date of birth{' '}
                <span className="notice">(This will not be displayed.)</span>
              </>
            }
            rules={[
              {
                required: true,
                message:
                  'Your profile information will be private unless you provide a birthdate.',
              },
            ]}
          >
            <DatePicker
              format={'YYYY-MM-DD'}
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
                type: 'email',
                message: 'Enter a valid email address',
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
      <Row>
        <Col span={11}>
          <Form.Item
            name="silentMode"
            label="Silent Mode (turn off all private chat)"
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>
        </Col>
      </Row>
      <Divider />
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
