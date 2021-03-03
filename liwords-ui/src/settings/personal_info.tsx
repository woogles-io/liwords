import React, { useCallback, useEffect } from 'react';
import { Button, Form, Input, notification } from 'antd';
import { PlayerAvatar } from '../shared/player_avatar';
import { PlayerMetadata } from '../gameroom/game_info';
import { useMountedState } from '../utils/mounted';
import { AvatarEditModal } from './avatar_edit_modal';
import { AvatarRemoveModal } from './avatar_remove_modal';
import axios, { AxiosError } from 'axios';
import { toAPIUrl } from '../api/api';

type PersonalInfo = {
  email: string;
  firstName: string;
  lastName: string;
  countryCode: string;
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

  const [updateAvatarModalVisible, setUpdateAvatarModalVisible] = useState(
    false
  );
  const [removeAvatarModalVisible, setRemoveAvatarModalVisible] = useState(
    false
  );
  const [avatarErr, setAvatarErr] = useState('');

  const avatarErrorCatcher = (e: AxiosError) => {
    if (e.response) {
      // From Twirp
      console.log(e);
      setAvatarErr(e.response.data.msg);
    } else {
      setAvatarErr('unknown error, see console');
      console.log(e);
    }
  };

  const cancelUpdateAvatarModal = useCallback(() => {
    setUpdateAvatarModalVisible(false);
  }, []);

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
        props.updatedAvatar('');
      })
      .catch(avatarErrorCatcher);
  }, [props]);

  const updateAvatar = useCallback(
    (avatarFile: File) => {
      let reader = new FileReader();
      reader.onload = () => {
        axios
          .post(
            toAPIUrl('user_service.ProfileService', 'UpdateAvatar'),
            {
              jpg_data: btoa(String(reader.result)),
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
            setUpdateAvatarModalVisible(false);
            props.updatedAvatar(resp.data.avatar_url);
          })
          .catch(avatarErrorCatcher);
      };
      reader.readAsBinaryString(avatarFile);
    },
    [props]
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

  const [form] = Form.useForm();

  useEffect(() => form.resetFields(), [props.personalInfo, form]);

  return (
    <Form
      form={form}
      className="personal-info"
      onFinish={updateFields}
      initialValues={props.personalInfo}
    >
      <h3>Personal info</h3>
      <div className="section-header">Profile picture</div>
      {props.player?.avatar_url !== '' ? (
        <div className="avatar-section">
          <PlayerAvatar player={props.player} />
          <Button
            className="change-avatar"
            onClick={() => setUpdateAvatarModalVisible(true)}
          >
            Change
          </Button>
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
          <Button
            className="change-avatar"
            onClick={() => setUpdateAvatarModalVisible(true)}
          >
            Add a Profile photo
          </Button>
        </div>
      )}
      <AvatarEditModal
        visible={updateAvatarModalVisible}
        error={avatarErr}
        onOk={updateAvatar}
        onCancel={cancelUpdateAvatarModal}
      />
      <AvatarRemoveModal
        visible={removeAvatarModalVisible}
        error={avatarErr}
        onOk={removeAvatar}
        onCancel={cancelRemoveAvatarModal}
      />
      <div className="section-header">Player bio</div>
      <div>(the big bio box)</div>
      <div className="section-header">Account details</div>
      <div className="rows">
        <div className="row">
          <div className="element">
            <div>Email</div>
            <Form.Item name="email">
              <Input size="large" />
            </Form.Item>
          </div>
          <div className="element">
            <div>First name</div>
            <Form.Item name="firstName">
              <Input size="large" />
            </Form.Item>
          </div>
        </div>
        <div className="row">
          <div className="element">
            <div>Last name</div>
            <Form.Item name="lastName">
              <Input size="large" />
            </Form.Item>
          </div>
          <div className="element">
            <div>Country</div>
            <Form.Item name="countryCode">
              <Input size="large" />
            </Form.Item>
          </div>
        </div>
        <div className="row">
          <div
            className="personal-info-close-account-button"
            onClick={() => {
              props.startClosingAccount();
            }}
          >
            Close my account
          </div>
          <Button className="save-button" type="primary" htmlType="submit">
            Save
          </Button>
        </div>
      </div>
    </Form>
  );
});
