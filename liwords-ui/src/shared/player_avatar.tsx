import React, { useEffect } from 'react';
import { fixedCharAt } from '../utils/cwgame/common';
import './avatar.scss';
import axios from 'axios';
import { toAPIUrl } from '../api/api';
import { useMountedState } from '../utils/mounted';
import { notification, Tooltip, Modal, Alert, Button, Upload } from 'antd';
import { EditOutlined, UploadOutlined } from '@ant-design/icons';
import { PlayerMetadata } from '../gameroom/game_info';
const colors = require('../base.scss');

type AvatarProps = {
  player: Partial<PlayerMetadata> | undefined;
  withTooltip?: boolean;
  editable?: boolean;
};

export const PlayerAvatar = (props: AvatarProps) => {
  const { useState } = useMountedState();

  const [updateModalVisible, setUpdateModalVisible] = useState(false);
  const [avatarErr, setAvatarErr] = useState('');
  const [avatarUrl, setAvatarUrl] = useState<string | undefined>("");
  const [avatarFile, setAvatarFile] = useState(new File([""], ""));

  useEffect(() => {
    setAvatarUrl(props.player?.avatar_url);
  }, [props.player]);

  useEffect(() => {
    setAvatarErr('');
    var fileInput = (document.getElementById('avatar-file-input') as HTMLInputElement);
    if (fileInput !== null) {
      fileInput.value = '';
    }
  }, [updateModalVisible]);

  var okButtonDisabled = (avatarFile == null || avatarFile.name.length === 0);
  const fileProps = {
    beforeUpload: (file: File) => { return false },
    maxCount: 1,
    onChange: (info: any) => {
      if (info.fileList.length > 0) {
        setAvatarFile(info.fileList.slice(-1)[0].originFileObj);
      } else {
        setAvatarFile(new File([""], ""));
      }
    },
    accept: "image/jpeg",
    showUploadList: false,
  }

  const updateModal = 
      <Modal
        className="avatar-update-modal"
        title="Update avatar"
        visible={updateModalVisible}
        okText="Upload"
        okButtonProps={{ disabled: okButtonDisabled }}
        onCancel={() => {
          setUpdateModalVisible(false);
        }}
        onOk={() => {
          console.log(avatarFile);
          var reader = new FileReader();
          reader.onload = function () {
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
                setUpdateModalVisible(false);              
                setAvatarUrl(resp.data.avatar_url);
                console.log(resp.data.avatar_url);
              })
              .catch((e) => {
                if (e.response) {
                  // From Twirp
                  console.log(e);
                  setAvatarErr(e.response.data.msg);
                } else {
                  setAvatarErr('unknown error, see console');
                  console.log(e);
                }
              });
          }
          reader.readAsBinaryString(avatarFile);
        }}
      >
        <Upload {...fileProps}>
          <Button icon={<UploadOutlined />}>Select avatar</Button>
        </Upload>
        <>{avatarFile.name}</>
        {avatarErr !== '' ? <Alert message={avatarErr} type="error" /> : null}
      </Modal>
 
   let avatarStyle = {};

  if (props.player?.first) {
    avatarStyle = {
      transform: 'rotate(-10deg)',
    };
  }

  if (avatarUrl) {
    avatarStyle = {
      backgroundImage: `url(${avatarUrl})`,
    };
  }

  const editControl = props.editable ? (
    <EditOutlined
      onClick={(e) => {
        e.preventDefault()
        setUpdateModalVisible(true)
      }}
    />
  ) : null;

  const renderAvatar = (
    <div>
      <div className="player-avatar" style={avatarStyle}>
        {!avatarUrl
          ? fixedCharAt(
              props.player?.full_name || props.player?.nickname || '?',
              0,
              1
            )
          : ''}
        {editControl}
      </div>
      {updateModal}
    </div>
  );
  if (!props.withTooltip) {
    return renderAvatar;
  }
  return (
    <Tooltip
      title={props.player?.nickname}
      placement="left"
      mouseEnterDelay={0.1}
      mouseLeaveDelay={0.01}
      color={colors.colorPrimary}
    >
      {renderAvatar}
    </Tooltip>
  );
};
