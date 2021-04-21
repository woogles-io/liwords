import React, { useCallback, useEffect } from 'react';
import {
  Alert,
  Button,
  Col,
  Form,
  Input,
  Modal,
  Upload,
  Row,
  Select,
  notification,
} from 'antd';
import { PlayerAvatar } from '../shared/player_avatar';
import { PlayerMetadata } from '../gameroom/game_info';
import { useMountedState } from '../utils/mounted';
import { AvatarRemoveModal } from './avatar_remove_modal';
import axios, { AxiosError } from 'axios';
import { toAPIUrl } from '../api/api';
import { countryArray } from './country_map';
import { MarkdownTips } from './markdown_tips';

type PersonalInfo = {
  email: string;
  firstName: string;
  lastName: string;
  countryCode: string;
  about: string;
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

  const [removeAvatarModalVisible, setRemoveAvatarModalVisible] = useState(
    false
  );
  const [bioTipsModalVisible, setBioTipsModalVisible] = useState(false);
  const [avatarErr, setAvatarErr] = useState('');
  const avatarErrorCatcher = useCallback((e: AxiosError) => {
    if (e.response) {
      // From Twirp
      console.log(e);
      setAvatarErr(e.response.data.msg);
    } else {
      setAvatarErr('unknown error, see console');
      console.log(e);
    }
  }, []);
  const propsUpdatedAvatar = props.updatedAvatar;
  const fileProps = {
    beforeUpload: (file: File) => {
      return false;
    },
    maxCount: 1,
    onChange: (info: any) => {
      if (info.fileList.length > 0) {
        // If they try again, the new one goes to the end of the list
        updateAvatar(info.fileList[info.fileList.length - 1].originFileObj);
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

  const updateAvatar = useCallback(
    (avatarFile: Blob) => {
      let reader = new FileReader();
      reader.onload = (readerEvent) => {
        let image = new Image();
        image.onload = () => {
          const canvas = document.createElement('canvas'),
            width = image.width,
            height = image.height;
          if (width < 96 || width !== height) {
            setAvatarErr('Image must be square and at least 96x96.');
          } else {
            canvas.width = 96;
            canvas.height = 96;
            canvas.getContext('2d')?.drawImage(image, 0, 0, width, height);
            // The endpoint doesn't want the file type data so cut that off
            const jpegString = canvas.toDataURL('image/jpeg', 1).split(',')[1];
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
                propsUpdatedAvatar(resp.data.avatar_url);
              })
              .catch(avatarErrorCatcher);
          }
        };
        image.src = String(reader.result);
      };
      reader.readAsDataURL(avatarFile);
    },
    [propsUpdatedAvatar, avatarErrorCatcher]
  );

  const updateFields = (values: { [key: string]: string }) => {
    axios
      .post(
        toAPIUrl('user_service.ProfileService', 'UpdatePersonalInfo'),
        values,
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

  useEffect(() => form.resetFields(), [props.personalInfo, form]);

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
          onClick={() => {
            const newWindow = window.open(
              'https://www.markdownguide.org/cheat-sheet/',
              '_blank',
              'noopener,noreferrer'
            );
            if (newWindow) newWindow.opener = null;
          }}
        >
          See Full Guide
        </Button>,
        <Button
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
      initialValues={props.personalInfo}
    >
      <h3>Personal info</h3>
      <div className="section-header">Profile picture</div>
      {props.player?.avatar_url !== '' ? (
        <div className="avatar-section">
          <PlayerAvatar player={props.player} />
          <Upload {...fileProps}>
            <Button className="change-avatar">Change</Button>
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
            <Button className="change-avatar">Add a Profile photo</Button>
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
      <Row align="middle">
        <Col
          span={8}
          className="close-account-button"
          onClick={() => {
            props.startClosingAccount();
          }}
        >
          Close my account
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
