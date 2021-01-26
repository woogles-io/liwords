import React from 'react';
import { fixedCharAt } from '../utils/cwgame/common';
import { Link } from 'react-router-dom';
import './avatar.scss';
import axios from 'axios';
import { toAPIUrl } from '../api/api';
import { useMountedState } from '../utils/mounted';
import { notification, Tooltip, Modal, Alert, Button } from 'antd';
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
  const [err, setErr] = useState('');
  const [avatarFile, setAvatarFile] = useState(new File([""], ""));

  const handleChange = (files: FileList | null) => {
    const file: File = (files as FileList)[0];
    setAvatarFile(file);
  };     
  
  const onAvatarUpdate = () => { 
    const formData = new FormData(); 
    formData.append( 
      "avatarFile", 
      avatarFile,
      avatarFile.name 
    ); 
   
    // Details of the updated file 
    console.log(avatarFile); 
   
    // Request made to the backend api 
    // Send formData object 
    //axios.post("api/uploadfile", formData); 
  }; 
  
  var okButtonDisabled = (avatarFile == null || avatarFile.name.length === 0);
  const updateModal = 
      <Modal
        className="avatar-update-modal"
        title="Update avatar"
        visible={updateModalVisible}
        okText="Upload Avatar"
        okButtonProps={{ disabled: okButtonDisabled }}
        onCancel={() => {
          setUpdateModalVisible(false);
        }}
        onOk={() => {
          axios
            .post(
              toAPIUrl('user_service.ProfileService', 'UpdateProfile'),
              {
                about: 'testing'
              },
              {
                withCredentials: true,
              }
            )
            .then(() => {
              notification.info({
                message: 'Success',
                description: 'Your avatar was updated.',
              });
              setUpdateModalVisible(false);              
            })
            .catch((e) => {
              if (e.response) {
                // From Twirp
                console.log(e);
                setErr(e.response.data.msg);
              } else {
                setErr('unknown error, see console');
                console.log(e);
              }
            });
        }}
      >
        <div> 
            <input type="file" onChange={(e) => handleChange(e.target.files) } /> 
        </div> 
        {err !== '' ? <Alert message={err} type="error" /> : null}
      </Modal>
 
   let avatarStyle = {};

  if (props.player?.first) {
    avatarStyle = {
      transform: 'rotate(-10deg)',
    };
  }

  if (props.player?.avatar_url) {
    avatarStyle = {
      backgroundImage: `url(${props.player?.avatar_url})`,
    };
  }

  const editControl = props.editable ? (
    <Link to="/" 
      onClick={(e) => {
        e.preventDefault()
        setUpdateModalVisible(true)
      }}
      >
      Edit
    </Link>
  ) : null;

  const renderAvatar = (
    <div>
      <div className="player-avatar" style={avatarStyle}>
        {!props.player?.avatar_url
          ? fixedCharAt(
              props.player?.full_name || props.player?.nickname || '?',
              0,
              1
            )
          : ''}
      </div>
      {editControl}
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
